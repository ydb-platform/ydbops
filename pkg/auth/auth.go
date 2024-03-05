package auth

import (
	"context"
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/Ydb_Auth_V1"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Auth"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"google.golang.org/protobuf/proto"

	"github.com/ydb-platform/ydb-ops/pkg/client"
	"github.com/ydb-platform/ydb-ops/pkg/options"
	"go.uber.org/zap"
)

type AuthClient struct {
	logger *zap.SugaredLogger
	f *client.Factory
}

func NewAuthClient(logger *zap.SugaredLogger, f *client.Factory) *AuthClient {
	return &AuthClient{
		logger: logger,
		f: f,
	}
}

// TODO move grpcTimeoutSeconds from CMS opts to GRPC opts, it makes more sense there
func (c *AuthClient) Auth(
	grpcOpts options.GRPC,
	grpcTimeoutSeconds int,
	user, password string,
) (string, error) {
	result := Ydb_Auth.LoginResult{}

	_, err := c.ExecuteAuthMethod(&result, func(ctx context.Context, cl Ydb_Auth_V1.AuthServiceClient) (client.OperationResponse, error) {
		c.logger.Debug("Invoke Auth method")
		return cl.Login(ctx, &Ydb_Auth.LoginRequest{
			OperationParams: c.f.OperationParams(),
			User:            user,
			Password:        password,
		})
	}, grpcOpts, grpcTimeoutSeconds)

	if err != nil {
		return "", err
	}
	c.logger.Debug(fmt.Sprintf("Login response: %s", string([]rune(result.Token)[:20])))
	return result.Token, nil
}

func (c *AuthClient) ExecuteAuthMethod(
	out proto.Message,
	method func(context.Context, Ydb_Auth_V1.AuthServiceClient) (client.OperationResponse, error),
	grpcOpts options.GRPC,
	grpcTimeoutSeconds int,
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
		c.logger.Debugf("Invocation error: %+v", err)
		return nil, err
	}
	op := r.GetOperation()
	client.LogOperation(c.logger, op)

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
