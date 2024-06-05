package client

import (
	"context"
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/Ydb_Discovery_V1"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Discovery struct {
	logger *zap.SugaredLogger
	f      *Factory
}

func NewDiscoveryClient(f *Factory, logger *zap.SugaredLogger) *Discovery {
	return &Discovery{
		logger: logger,
		f:      f,
	}
}

func (c *Discovery) ListEndpoints(database string) ([]*Ydb_Discovery.EndpointInfo, error) {
	result := Ydb_Discovery.ListEndpointsResult{}
	_, err := c.ExecuteDiscoveryMethod(&result, func(ctx context.Context, cl Ydb_Discovery_V1.DiscoveryServiceClient) (OperationResponse, error) {
		c.logger.Debug("Invoke ListEndpoints method")
		return cl.ListEndpoints(ctx, &Ydb_Discovery.ListEndpointsRequest{
			Database: database,
		})
	})
	if err != nil {
		return nil, err
	}

	return result.Endpoints, nil
}

func (c *Discovery) WhoAmI() (string, error) {
	result := Ydb_Discovery.WhoAmIResult{}
	c.logger.Debug("Invoke WhoAmI method")
	_, err := c.ExecuteDiscoveryMethod(&result, func(ctx context.Context, cl Ydb_Discovery_V1.DiscoveryServiceClient) (OperationResponse, error) {
		return cl.WhoAmI(ctx, &Ydb_Discovery.WhoAmIRequest{IncludeGroups: false})
	})
	if err != nil {
		return "", err
	}
	c.logger.Debugf("WhoAmI response: %s", result.User)
	return result.User, nil
}

func (c *Discovery) ExecuteDiscoveryMethod(
	out proto.Message,
	method func(context.Context, Ydb_Discovery_V1.DiscoveryServiceClient) (OperationResponse, error),
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
