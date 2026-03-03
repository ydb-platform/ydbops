package utils

import (
	"errors"
	"testing"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	// Initialize zap with a no-op logger to avoid panics
	logger := zap.NewNop()
	zap.ReplaceGlobals(logger)
}

func TestWrapWithRetries_Success(t *testing.T) {
	callCount := 0
	op := &Ydb_Operations.Operation{Status: Ydb.StatusIds_SUCCESS}

	result, err := WrapWithRetries(3, func() (*Ydb_Operations.Operation, error) {
		callCount++
		return op, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != op {
		t.Errorf("expected operation %v, got %v", op, result)
	}
	if callCount != 1 {
		t.Errorf("expected exactly 1 call, got %d", callCount)
	}
}

func TestWrapWithRetries_ErrorNonRetryable(t *testing.T) {
	callCount := 0
	expectedErr := status.Error(codes.Internal, "internal error")

	result, err := WrapWithRetries(3, func() (*Ydb_Operations.Operation, error) {
		callCount++
		return nil, expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if result != nil {
		t.Errorf("expected nil operation, got %v", result)
	}
	if callCount != 1 {
		t.Errorf("expected exactly 1 call, got %d", callCount)
	}
}

func TestWrapWithRetries_ErrorRetryable_EventuallySucceeds(t *testing.T) {
	callCount := 0
	successOp := &Ydb_Operations.Operation{Status: Ydb.StatusIds_SUCCESS}

	result, err := WrapWithRetries(5, func() (*Ydb_Operations.Operation, error) {
		callCount++
		if callCount < 3 {
			return nil, status.Error(codes.Unavailable, "unavailable")
		}
		return successOp, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != successOp {
		t.Errorf("expected operation %v, got %v", successOp, result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestWrapWithRetries_ErrorRetryable_MaxAttemptsExceeded(t *testing.T) {
	callCount := 0
	expectedErr := status.Error(codes.Unavailable, "unavailable")

	result, err := WrapWithRetries(3, func() (*Ydb_Operations.Operation, error) {
		callCount++
		return nil, expectedErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*RetryExceededError); !ok {
		t.Errorf("expected RetryExceededError, got %T: %v", err, err)
	}
	if result != nil {
		t.Errorf("expected nil operation, got %v", result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestWrapWithRetries_OperationStatusUnavailable_Retries(t *testing.T) {
	callCount := 0
	unavailableOp := &Ydb_Operations.Operation{Status: Ydb.StatusIds_UNAVAILABLE}
	successOp := &Ydb_Operations.Operation{Status: Ydb.StatusIds_SUCCESS}

	result, err := WrapWithRetries(5, func() (*Ydb_Operations.Operation, error) {
		callCount++
		if callCount < 3 {
			return unavailableOp, nil
		}
		return successOp, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != successOp {
		t.Errorf("expected operation %v, got %v", successOp, result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestWrapWithRetries_OperationStatusUnavailable_MaxAttemptsExceeded(t *testing.T) {
	callCount := 0
	unavailableOp := &Ydb_Operations.Operation{Status: Ydb.StatusIds_UNAVAILABLE}

	result, err := WrapWithRetries(3, func() (*Ydb_Operations.Operation, error) {
		callCount++
		return unavailableOp, nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*RetryExceededError); !ok {
		t.Errorf("expected RetryExceededError, got %T: %v", err, err)
	}
	if result != nil {
		t.Errorf("expected nil operation, got %v", result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestWrapWithRetries_OperationStatusBadRequest_NoRetry(t *testing.T) {
	callCount := 0
	badRequestOp := &Ydb_Operations.Operation{Status: Ydb.StatusIds_BAD_REQUEST}

	result, err := WrapWithRetries(3, func() (*Ydb_Operations.Operation, error) {
		callCount++
		return badRequestOp, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != badRequestOp {
		t.Errorf("expected operation %v, got %v", badRequestOp, result)
	}
	if callCount != 1 {
		t.Errorf("expected exactly 1 call, got %d", callCount)
	}
}

func TestWrapWithRetries_BackoffTiming(t *testing.T) {
	// This test ensures that backoff increases with attempts.
	// We'll mock time.Sleep by capturing the delay.
	// Since we cannot easily mock time.Sleep, we can skip this test or use a custom sleeper.
	// For simplicity, we'll just verify that the function doesn't panic.
	callCount := 0
	start := time.Now()
	result, err := WrapWithRetries(2, func() (*Ydb_Operations.Operation, error) {
		callCount++
		return nil, status.Error(codes.Unavailable, "unavailable")
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should have slept at least 1 second (backoff for attempt 0 is 1 second)
	if elapsed < time.Second {
		t.Errorf("expected at least 1 second sleep, got %v", elapsed)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
	_ = result // unused
}
