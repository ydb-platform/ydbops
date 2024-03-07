package discovery

import (
	"context"
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/Ydb_Discovery_V1"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"github.com/ydb-platform/ydb-ops/pkg/client"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type DiscoveryClient struct {
	logger *zap.SugaredLogger
	f      *client.Factory
}

func NewDiscoveryClient(logger *zap.SugaredLogger, f *client.Factory) *DiscoveryClient {
	return &DiscoveryClient{
		logger: logger,
		f:      f,
	}
}

func (c *DiscoveryClient) WhoAmI() (string, error) {
	result := Ydb_Discovery.WhoAmIResult{}
	_, err := c.ExecuteDiscoveryMethod(&result, func(ctx context.Context, cl Ydb_Discovery_V1.DiscoveryServiceClient) (client.OperationResponse, error) {

		c.logger.Debug("Invoke WhoAmI method")
		return cl.WhoAmI(ctx, &Ydb_Discovery.WhoAmIRequest{IncludeGroups: false})
	})
	if err != nil {
		return "", err
	}
	c.logger.Debugf("WhoAmI response: %s", result.User)
	return result.User, nil
}

func (c *DiscoveryClient) ExecuteDiscoveryMethod(
	out proto.Message,
	method func(context.Context, Ydb_Discovery_V1.DiscoveryServiceClient) (client.OperationResponse, error),
) (*Ydb_Operations.Operation, error) {
	cc, err := c.f.Connection()
	if err != nil {
		return nil, err
	}

	ctx, cancel, err := c.f.ContextWithAuth()
	if err != nil {
		return nil, err
	}
	defer cancel()

	cl := Ydb_Discovery_V1.NewDiscoveryServiceClient(cc)
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
