package connection

import (
	"fmt"

	"github.com/ydb-platform/ydbops/pkg/auth"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/cms"
	"github.com/ydb-platform/ydbops/pkg/discovery"
	"github.com/ydb-platform/ydbops/pkg/options"
	"go.uber.org/zap"
)

func PrepareClients(
	rootOpts options.RootOptions,
	retryNumber int,
	logger *zap.SugaredLogger,
) (*cms.Client, *discovery.Client, error) {
	factory := client.NewConnectionFactory(rootOpts.Auth, rootOpts.GRPC, retryNumber)

	err := InitAuthToken(rootOpts, logger, factory)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to receive an auth token, rolling restart not started: %w", err)
	}

	cmsClient := cms.NewCMSClient(logger, factory)
	discoveryClient := discovery.NewDiscoveryClient(logger, factory)

	return cmsClient, discoveryClient, nil
}

func InitAuthToken(
	rootOpts options.RootOptions,
	logger *zap.SugaredLogger,
	factory *client.Factory,
) error {
	switch rootOpts.Auth.Type {
	case options.Static:
		authClient := auth.NewAuthClient(logger, factory)
		staticCreds := rootOpts.Auth.Creds.(*options.AuthStatic)
		user := staticCreds.User
		password := staticCreds.Password
		logger.Debugf("Endpoint: %v", rootOpts.GRPC.Endpoint)
		token, err := authClient.Auth(rootOpts.GRPC, user, password)
		if err != nil {
			return fmt.Errorf("failed to initialize static auth token: %w", err)
		}
		factory.SetAuthToken(token)
	case options.IamToken:
		factory.SetAuthToken(rootOpts.Auth.Creds.(*options.AuthIAMToken).Token)
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
