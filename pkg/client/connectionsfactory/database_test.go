package connectionsfactory

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var _ = Describe("Database metadata", func() {
	DescribeTable("unary interceptor",
		func(database string, initialDatabase string, expectedDatabase []string) {
			ctx := metadata.AppendToOutgoingContext(context.Background(), "x-ydb-auth-ticket", "token")
			if initialDatabase != "" {
				ctx = metadata.AppendToOutgoingContext(ctx, databaseHeader, initialDatabase)
			}

			var got metadata.MD
			invoker := func(
				ctx context.Context,
				_ string,
				_ any,
				_ any,
				_ *grpc.ClientConn,
				_ ...grpc.CallOption,
			) error {
				got, _ = metadata.FromOutgoingContext(ctx)
				return nil
			}

			Expect(databaseUnaryInterceptor(database)(ctx, "method", nil, nil, nil, invoker)).To(Succeed())
			Expect(got.Get(databaseHeader)).To(Equal(expectedDatabase))
			Expect(got.Get("x-ydb-auth-ticket")).To(Equal([]string{"token"}))
		},
		Entry("adds database", "/Root", "", []string{"/Root"}),
		Entry("overrides database", "/Root", "/Old", []string{"/Root"}),
		Entry("keeps database unset", "", "", []string(nil)),
	)
})
