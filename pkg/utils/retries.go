package utils

import (
	"fmt"
	"math"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RetryExceededError struct {
	msg string
	err error
}

func (e *RetryExceededError) Error() string {
	return e.msg + ". Last error:" + e.err.Error()
}

func (e *RetryExceededError) Unwrap() error {
	return e.err
}

func (e *RetryExceededError) Is(targetErr error) bool {
	_, ok := targetErr.(*RetryExceededError)
	return ok
}

func backoffTimeAfter(attempt int) time.Duration {
	return time.Second * time.Duration(int(math.Pow(2, float64(attempt))))
}

func shouldRetry(code codes.Code) bool {
	// TODO what other error codes?
	return code == codes.Unavailable
}

func WrapWithRetries(
	maxAttempts int,
	f func() (*Ydb_Operations.Operation, error),
) (*Ydb_Operations.Operation, error) {
	var lastError error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		op, err := f()
		if err == nil {
			return op, nil
		}

		if s, ok := status.FromError(err); ok && shouldRetry(s.Code()) {
			delay := backoffTimeAfter(attempt)
			if attempt < maxAttempts-1 {
				zap.S().Debugf("Retrying after %v seconds...\n", delay.Seconds())
				time.Sleep(delay)
			}
			lastError = err
		} else {
			// Don't retry for non-transient errors
			return nil, err
		}
	}

	return nil, &RetryExceededError{
		msg: fmt.Sprintf("number of retries exceeded: %v", maxAttempts),
		err: lastError,
	}
}
