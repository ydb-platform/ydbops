package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Issue"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/options"
)

const (
	BufferSize = 32 << 20
)

type OperationResponse interface {
	GetOperation() *Ydb_Operations.Operation
}

type Factory struct {
	auth               options.AuthOptions
	rootOpts           options.RootOptions
	grpcTimeoutSeconds int
	token              string
}

func NewConnectionFactory(
	grpcTimeoutSeconds int,
	rootOpts options.RootOptions,
) *Factory {
	return &Factory{
		auth:               rootOpts.Auth,
		grpcTimeoutSeconds: grpcTimeoutSeconds,
		rootOpts:           rootOpts,
	}
}

func (f *Factory) SetAuthToken(t string) {
	f.token = t
}

func (f *Factory) Connection() (*grpc.ClientConn, error) {
	cr, err := f.makeCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %v", err)
	}

	return grpc.Dial(f.endpoint(),
		grpc.WithTransportCredentials(cr),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(BufferSize),
			grpc.MaxCallRecvMsgSize(BufferSize)))
}

func (f *Factory) OperationParams() *Ydb_Operations.OperationParams {
	return &Ydb_Operations.OperationParams{
		OperationMode:    Ydb_Operations.OperationParams_SYNC,
		OperationTimeout: durationpb.New(time.Duration(f.grpcTimeoutSeconds) * time.Second),
		CancelAfter:      durationpb.New(time.Duration(f.grpcTimeoutSeconds) * time.Second),
	}
}

func (f *Factory) makeCredentials() (credentials.TransportCredentials, error) {
	opts := f.rootOpts.GRPC
	if !opts.GRPCSecure {
		return insecure.NewCredentials(), nil
	}

	if opts.CaFile == "" {
		// TODO verify that this will use system pool
		return credentials.NewClientTLSFromCert(nil, ""), nil
	}

	return credentials.NewClientTLSFromFile(opts.CaFile, "")
}

func (f *Factory) endpoint() string {
	// TODO decide if we want to support multiple endpoints or just one
	// Endpoint in rootOpts will turn from string -> []string in this case
	return fmt.Sprintf("%s:%d", f.rootOpts.GRPC.Endpoint, f.rootOpts.GRPC.GRPCPort)
}

func (f Factory) ContextWithAuth() (context.Context, context.CancelFunc, error) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second*time.Duration(f.grpcTimeoutSeconds))

	return metadata.AppendToOutgoingContext(ctx,
		"x-ydb-auth-ticket", f.token), cf, nil
}

func (f Factory) ContextWithoutAuth() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Second*time.Duration(f.grpcTimeoutSeconds))
}

func LogOperation(logger *zap.SugaredLogger, op *Ydb_Operations.Operation) {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Operation status: %s", op.Status))

	if len(op.Issues) > 0 {
		sb.WriteString(
			fmt.Sprintf("\nIssues:\n%s",
				strings.Join(util.Convert(op.Issues,
					func(issue *Ydb_Issue.IssueMessage) string {
						return fmt.Sprintf("  Severity: %d, code: %d, message: %s", issue.Severity, issue.IssueCode, issue.Message)
					},
				), "\n"),
			))
	}

	logger.Debugf("Invocation result:\n%s", sb.String())
}
