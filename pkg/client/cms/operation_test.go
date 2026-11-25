package cms

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/ydb-platform/ydbops/pkg/client"
	authprovider "github.com/ydb-platform/ydbops/pkg/client/auth/provider"
)

type operationTestFactory struct{}

func (operationTestFactory) Create() (*grpc.ClientConn, error) {
	return grpc.NewClient("passthrough:///unused", grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func (operationTestFactory) OperationParams() *Ydb_Operations.OperationParams {
	return &Ydb_Operations.OperationParams{}
}

func (operationTestFactory) OperationTimeout() time.Duration { return time.Second }
func (operationTestFactory) CallTimeout() time.Duration      { return time.Second }

type operationTestResponse struct{ op *Ydb_Operations.Operation }

func (r operationTestResponse) GetOperation() *Ydb_Operations.Operation { return r.op }

func TestOperationParentCancellation(t *testing.T) {
	c := NewCMSClient(operationTestFactory{}, zap.NewNop().Sugar(), authprovider.NewProviderFromToken("ticket")).(*defaultCMSClient)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.SetContext(ctx)
	_, err := c.executeOperation(nil, func(context.Context, *grpc.ClientConn) (client.OperationResponse, error) {
		cancel()
		return nil, status.Error(codes.Canceled, "context canceled")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("operation error = %v; cleanup requires context.Canceled", err)
	}
}

func TestOperationPerCallDeadlineRetries(t *testing.T) {
	c := NewCMSClient(operationTestFactory{}, zap.NewNop().Sugar(), authprovider.NewProviderFromToken("ticket")).(*defaultCMSClient)
	attempts := 0
	op, err := c.executeOperation(nil, func(context.Context, *grpc.ClientConn) (client.OperationResponse, error) {
		attempts++
		if attempts == 1 {
			return nil, status.Error(codes.DeadlineExceeded, "single RPC timeout")
		}
		return operationTestResponse{&Ydb_Operations.Operation{Status: Ydb.StatusIds_SUCCESS}}, nil
	})
	if err != nil || op == nil || attempts != 2 {
		t.Fatalf("operation = %v, error = %v, attempts = %d; want success on retry", op, err, attempts)
	}
}
