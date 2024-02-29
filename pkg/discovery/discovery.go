package discovery

import (
	"context"
	"fmt"

	"github.com/ydb-platform/ydb-go-sdk/v3"
)

type DiscoveryClient struct {
	endpoint   string
}

func buildEndpoint(endpoint string, grpcPort int, grpcSecure bool) string {
	proto := "grpc"
	if grpcSecure {
		proto = "grpcs"
	}

	return fmt.Sprintf("%s://%s:%s", proto, endpoint, grpcPort)
}


func NewDiscoveryClient(endpoint string, grpcPort int, grpcSecure bool) *DiscoveryClient {
  return &DiscoveryClient{
    endpoint: buildEndpoint(endpoint, grpcPort, grpcSecure),
  }
}

func (d DiscoveryClient) WhoAmI(ctx context.Context) (string, error) {
	db, err := ydb.Open(ctx, d.endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to connect for whoami: %v", err)
	}
	whoAmI, err := db.Discovery().WhoAmI(ctx)
	if err != nil {
		return "", fmt.Errorf("whoami failed: %v", err)
	}
	defer db.Close(ctx)
	return whoAmI.User, nil
}
