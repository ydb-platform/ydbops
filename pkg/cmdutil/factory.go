package cmdutil

import (
	"context"
	"fmt"
	"time"

	"firebase.google.com/go/auth"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/ydb-platform/ydbops/pkg/client/cms"
	connectionsfactory "github.com/ydb-platform/ydbops/pkg/client/connections_factory"
	"github.com/ydb-platform/ydbops/pkg/client/discovery"
	"github.com/ydb-platform/ydbops/pkg/command"
)

type Factory interface {
	Init() error
	GetCMSClient() cms.Client
	GetDiscoveryClient() discovery.Client
	GetAuthClient() auth.Client

	OperationParams() *Ydb_Operations.OperationParams
}

type OperationResponse interface {
	GetOperation() *Ydb_Operations.Operation
}

type factory struct {
	connectionsFactory connectionsfactory.Factory
	options            command.BaseOptions
	retryNumber        int
	token              string
}

func New(
	cf connectionsfactory.Factory,
	options command.BaseOptions,
) Factory {
	return &factory{
		connectionsFactory: cf,
		options:            options,
	}
}

func (f *factory) initAuthToken() error {
	return nil
}

func (f *factory) Init() error {
	return nil
}

func (f *factory) GetCMSClient() cms.Client {
	return nil
}

func (f *factory) GetDiscoveryClient() discovery.Client {
	return nil
}

func InitConnectionFactory(
	rootOpts command.BaseOptions,
	logger *zap.SugaredLogger,
	retryNumber int,
) error {
	once.Do(func() {
		factory = &Factory{
			auth:        rootOpts.Auth,
			grpc:        rootOpts.GRPC,
			retryNumber: retryNumber,
		}

		initErr = initAuthToken(rootOpts, logger, factory)

		if initErr != nil {
			initErr = fmt.Errorf("failed to receive an auth token, rolling restart not started: %w", initErr)
		}
	})

	if initErr != nil {
		return initErr
	}

	return nil
}

func (f *Factory) OperationParams() *Ydb_Operations.OperationParams {
	return &Ydb_Operations.OperationParams{
		OperationMode:    Ydb_Operations.OperationParams_SYNC,
		OperationTimeout: durationpb.New(time.Duration(f.grpc.TimeoutSeconds) * time.Second),
		CancelAfter:      durationpb.New(time.Duration(f.grpc.TimeoutSeconds) * time.Second),
	}
}

func (f *Factory) ContextWithAuth() (context.Context, context.CancelFunc) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second*time.Duration(f.grpc.TimeoutSeconds))

	return metadata.AppendToOutgoingContext(ctx,
		"x-ydb-auth-ticket", f.token), cf
}

func (f *Factory) ContextWithoutAuth() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Second*time.Duration(f.grpc.TimeoutSeconds))
}

func (f *Factory) GetRetryNumber() int {
	return f.retryNumber
}
