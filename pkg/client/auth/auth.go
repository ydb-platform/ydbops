package auth

import (
	"context"
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/Ydb_Auth_V1"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Auth"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/client/connectionsfactory"
	"github.com/ydb-platform/ydbops/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Client interface {
	Auth(string, string) (string, error) // TODO(shmel1k@): add context to params
}

type defaultAuthClient struct {
	logger *zap.SugaredLogger
	f      connectionsfactory.Factory
}

func NewClient(
	logger *zap.SugaredLogger,
	f connectionsfactory.Factory,
) Client {
	return &defaultAuthClient{
		logger: logger,
		f:      f,
	}
}

func (c *defaultAuthClient) executeAuthMethod(
	out proto.Message,
	method func(context.Context, Ydb_Auth_V1.AuthServiceClient) (client.OperationResponse, error),
) (*Ydb_Operations.Operation, error) {
	cc, err := c.f.Create()
	if err != nil {
		return nil, err
	}

	ctx := context.TODO() // XXX(shmel1k@): improve context behavior.

	cl := Ydb_Auth_V1.NewAuthServiceClient(cc)
	r, err := method(ctx, cl)
	if err != nil {
		c.logger.Errorf("Invocation error: %+v", err)
		return nil, err
	}
	op := r.GetOperation()
	utils.LogOperation(c.logger, op)

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

func (c *defaultAuthClient) Auth(user, password string) (string, error) {
	result := Ydb_Auth.LoginResult{}

	c.logger.Debug("Invoke Auth method")
	_, err := c.executeAuthMethod(&result, func(ctx context.Context, cl Ydb_Auth_V1.AuthServiceClient) (client.OperationResponse, error) {
		return cl.Login(ctx, &Ydb_Auth.LoginRequest{
			OperationParams: c.f.OperationParams(),
			User:            user,
			Password:        password,
		})
	})
	if err != nil {
		return "", err
	}
	c.logger.Debugf("Login response: %s... (token contents masked)", string([]rune(result.Token)[:20]))
	return result.Token, nil
}
