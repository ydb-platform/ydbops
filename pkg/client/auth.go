package client

import (
	"context"
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/Ydb_Auth_V1"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Auth"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/ydb-platform/ydbops/pkg/options"
)

type Auth struct {
	logger *zap.SugaredLogger
	f      *Factory
}

func NewAuthClient(logger *zap.SugaredLogger, f *Factory) *Auth {
	return &Auth{
		logger: logger,
		f:      f,
	}
}

func (c *Auth) Auth(grpcOpts options.GRPC, user, password string) (string, error) {
	result := Ydb_Auth.LoginResult{}

	_, err := c.ExecuteAuthMethod(&result, func(ctx context.Context, cl Ydb_Auth_V1.AuthServiceClient) (OperationResponse, error) {
		c.logger.Debug("Invoke Auth method")
		return cl.Login(ctx, &Ydb_Auth.LoginRequest{
			OperationParams: c.f.OperationParams(),
			User:            user,
			Password:        password,
		})
	}, grpcOpts)
	if err != nil {
		return "", err
	}
	c.logger.Debugf("Login response: %s... (token contents masked)", string([]rune(result.Token)[:20]))
	return result.Token, nil
}

func (c *Auth) ExecuteAuthMethod(
	out proto.Message,
	method func(context.Context, Ydb_Auth_V1.AuthServiceClient) (OperationResponse, error),
	grpcOpts options.GRPC,
) (*Ydb_Operations.Operation, error) {
	cc, err := c.f.Connection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := c.f.ContextWithoutAuth()
	defer cancel()

	cl := Ydb_Auth_V1.NewAuthServiceClient(cc)
	r, err := method(ctx, cl)
	if err != nil {
		c.logger.Errorf("Invocation error: %+v", err)
		return nil, err
	}
	op := r.GetOperation()
	LogOperation(c.logger, op)

	if out == nil {
		return op, nil
	}

	if err := op.Result.UnmarshalTo(out); err != nil {
		return op, err
	}

	if op.Status != Ydb.StatusIds_SUCCESS {
		return op, fmt.Errorf("unsuccessful status code: %s", op.Status)
	}

	return op, nil
}

func initAuthToken(
	rootOpts options.RootOptions,
	logger *zap.SugaredLogger,
	factory *Factory,
) error {
	switch rootOpts.Auth.Type {
	case options.Static:
		authClient := NewAuthClient(logger, factory)
		staticCreds := rootOpts.Auth.Creds.(*options.AuthStatic)
		user := staticCreds.User
		password := staticCreds.Password
		logger.Debugf("Endpoint: %v", rootOpts.GRPC.Endpoint)
		token, err := authClient.Auth(rootOpts.GRPC, user, password)
		if err != nil {
			return fmt.Errorf("failed to initialize static auth token: %w", err)
		}
		factory.token = token
	case options.IamToken:
		factory.token = rootOpts.Auth.Creds.(*options.AuthIAMToken).Token
	case options.IamCreds:
		return fmt.Errorf("TODO: IAM authorization from SA key not implemented yet")
	case options.None:
		return fmt.Errorf("determined credentials to be anonymous. Anonymous credentials are currently unsupported")
	default:
		return fmt.Errorf(
			"internal error: authorization type not recognized after options validation, this should never happen",
		)
	}

	return nil
}
