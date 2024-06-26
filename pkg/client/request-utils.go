package client

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Issue"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ydb-platform/ydbops/internal/collections"
)

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

	return nil, fmt.Errorf("number of retries exceeded: %v. Last error: %w", maxAttempts, lastError)
}

func LogOperation(logger *zap.SugaredLogger, op *Ydb_Operations.Operation) {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Operation status: %s", op.Status))

	if len(op.Issues) > 0 {
		sb.WriteString(
			fmt.Sprintf("\nIssues:\n%s",
				strings.Join(collections.Convert(op.Issues,
					func(issue *Ydb_Issue.IssueMessage) string {
						return fmt.Sprintf("  Severity: %d, code: %d, message: %s", issue.Severity, issue.IssueCode, issue.Message)
					},
				), "\n"),
			))
	}

	if op.Status != Ydb.StatusIds_SUCCESS {
		logger.Errorf("GRPC invocation unsuccessful:\n%s", sb.String())
	} else {
		logger.Debugf("Invocation result:\n%s", sb.String())
	}
}
