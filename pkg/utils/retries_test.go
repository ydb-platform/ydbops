package utils

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("WrapWithRetries", func() {
	BeforeEach(func() {
		// Initialize zap with a no-op logger to avoid panics
		logger := zap.NewNop()
		zap.ReplaceGlobals(logger)
	})

	AfterEach(func() {
		// Reset zap global logger
		zap.ReplaceGlobals(zap.NewNop())
	})

	Describe("Successful operation", func() {
		It("should return operation immediately", func() {
			callCount := 0
			op := &Ydb_Operations.Operation{Status: Ydb.StatusIds_SUCCESS}

			result, err := WrapWithRetries(3, func() (*Ydb_Operations.Operation, error) {
				callCount++
				return op, nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(op))
			Expect(callCount).To(Equal(1))
		})
	})

	Describe("Non-retryable error", func() {
		It("should return error immediately", func() {
			callCount := 0
			expectedErr := status.Error(codes.Internal, "internal error")

			result, err := WrapWithRetries(3, func() (*Ydb_Operations.Operation, error) {
				callCount++
				return nil, expectedErr
			})

			Expect(err).To(MatchError(expectedErr))
			Expect(result).To(BeNil())
			Expect(callCount).To(Equal(1))
		})
	})

	Describe("Retryable gRPC error", func() {
		It("should retry and eventually succeed", func() {
			callCount := 0
			successOp := &Ydb_Operations.Operation{Status: Ydb.StatusIds_SUCCESS}

			result, err := WrapWithRetries(5, func() (*Ydb_Operations.Operation, error) {
				callCount++
				if callCount < 3 {
					return nil, status.Error(codes.Unavailable, "unavailable")
				}
				return successOp, nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(successOp))
			Expect(callCount).To(Equal(3))
		})

		It("should exceed max attempts and return RetryExceededError", func() {
			callCount := 0
			expectedErr := status.Error(codes.Unavailable, "unavailable")

			result, err := WrapWithRetries(3, func() (*Ydb_Operations.Operation, error) {
				callCount++
				return nil, expectedErr
			})

			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&RetryExceededError{}))
			Expect(result).To(BeNil())
			Expect(callCount).To(Equal(3))
		})
	})

	Describe("Operation status UNAVAILABLE", func() {
		It("should retry and eventually succeed", func() {
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

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(successOp))
			Expect(callCount).To(Equal(3))
		})

		It("should exceed max attempts and return RetryExceededError", func() {
			callCount := 0
			unavailableOp := &Ydb_Operations.Operation{Status: Ydb.StatusIds_UNAVAILABLE}

			result, err := WrapWithRetries(3, func() (*Ydb_Operations.Operation, error) {
				callCount++
				return unavailableOp, nil
			})

			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&RetryExceededError{}))
			Expect(result).To(BeNil())
			Expect(callCount).To(Equal(3))
		})
	})

	Describe("Operation status BAD_REQUEST", func() {
		It("should not retry and return operation", func() {
			callCount := 0
			badRequestOp := &Ydb_Operations.Operation{Status: Ydb.StatusIds_BAD_REQUEST}

			result, err := WrapWithRetries(3, func() (*Ydb_Operations.Operation, error) {
				callCount++
				return badRequestOp, nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(badRequestOp))
			Expect(callCount).To(Equal(1))
		})
	})

	Describe("Backoff timing", func() {
		It("should sleep between retries", func() {
			callCount := 0
			start := time.Now()
			result, err := WrapWithRetries(2, func() (*Ydb_Operations.Operation, error) {
				callCount++
				return nil, status.Error(codes.Unavailable, "unavailable")
			})
			elapsed := time.Since(start)

			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&RetryExceededError{}))
			Expect(result).To(BeNil())
			Expect(callCount).To(Equal(2))
			Expect(elapsed).To(BeNumerically(">=", time.Second))
		})
	})
})
