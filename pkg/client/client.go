package client

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Issue"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"github.com/ydb-platform/ydbops/internal/collections"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	for attempt := 0; attempt < maxAttempts; attempt++ {
		op, err := f()
		if err == nil {
			return op, nil
		}

		if s, ok := status.FromError(err); ok && shouldRetry(s.Code()) {
			delay := backoffTimeAfter(attempt)
			zap.S().Debugf("Retrying after %v seconds...\n", delay.Seconds())
			time.Sleep(delay)
		} else {
			// Don't retry for non-transient errors
			return nil, err
		}
	}

	return nil, fmt.Errorf("Number of retries exceeded: %v", maxAttempts)
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

	logger.Debugf("Invocation result:\n%s", sb.String())
}
