package credentials

import (
	"context"
	"fmt"

	"github.com/ydb-platform/ydbops/pkg/client/connectionsfactory"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
	"go.uber.org/zap"
)

type Provider interface {
	ContextWithAuth(context.Context) (context.Context, context.CancelFunc) // TODO(shmel1k@): think about compatibility
	// with ydb-go-sdk
	ContextWithoutAuth(context.Context) (context.Context, context.CancelFunc)

	GetToken() (string, error)
	Init() error
}

func CreateFromOptions(
	opts *command.BaseOptions,
	connectionsFactory connectionsfactory.Factory,
	logger *zap.SugaredLogger,
) (Provider, error) {
	switch opts.Auth.Type {
	case options.Static:
		staticCreds := opts.Auth.Creds.(*options.AuthStatic)
		return NewStatic(staticCreds.User, staticCreds.Password, connectionsFactory, logger), nil
	case options.IamToken:
		return NewIamToken(opts.Auth.Creds.(*options.AuthIAMToken).Token), nil
	case options.IamCreds:
		return nil, fmt.Errorf("TODO: IAM authorization from SA key not implemented yet")
	case options.None:
		return nil, fmt.Errorf("determined credentials to be anonymous. Anonymous credentials are currently unsupported")
	default:
		return nil, fmt.Errorf(
			"internal error: authorization type not recognized after options validation, this should never happen",
		)
	}
}
