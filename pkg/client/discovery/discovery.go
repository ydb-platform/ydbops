package discovery

import (
	"context"
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/Ydb_Discovery_V1"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/client/auth/credentials"
	"github.com/ydb-platform/ydbops/pkg/client/connectionsfactory"
	"github.com/ydb-platform/ydbops/pkg/utils"
)

type Discovery struct {
	logger              *zap.SugaredLogger
	connectionsFactory  connectionsfactory.Factory
	credentialsProvider credentials.Provider
}

type Client interface {
	ListEndpoints(string) ([]*Ydb_Discovery.EndpointInfo, error)
	WhoAmI() (string, error)
	Close() error
}

func NewDiscoveryClient(
	f connectionsfactory.Factory,
	logger *zap.SugaredLogger,
	cp credentials.Provider,
) *Discovery {
	return &Discovery{
		logger:              logger,
		connectionsFactory:  f,
		credentialsProvider: cp,
	}
}

func (c *Discovery) ListEndpoints(database string) ([]*Ydb_Discovery.EndpointInfo, error) {
	result := Ydb_Discovery.ListEndpointsResult{}
	_, err := c.ExecuteDiscoveryMethod(&result, func(ctx context.Context, cl Ydb_Discovery_V1.DiscoveryServiceClient) (client.OperationResponse, error) {
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
	_, err := c.ExecuteDiscoveryMethod(&result, func(ctx context.Context, cl Ydb_Discovery_V1.DiscoveryServiceClient) (client.OperationResponse, error) {
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
	method func(context.Context, Ydb_Discovery_V1.DiscoveryServiceClient) (client.OperationResponse, error),
) (*Ydb_Operations.Operation, error) {
	cc, err := c.connectionsFactory.Create()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cc.Close()
	}()

	ctx, cancel := c.credentialsProvider.ContextWithAuth(context.TODO())
	defer cancel()

	cl := Ydb_Discovery_V1.NewDiscoveryServiceClient(cc)
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

func (c *Discovery) Close() error {
	return nil
}
