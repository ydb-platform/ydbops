package credentials

import (
	"context"
	"fmt"
	"sync"

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

type baseProvider struct {
	impl               Provider
	opts               *command.BaseOptions
	connectionsFactory connectionsfactory.Factory
	logger             *zap.SugaredLogger

	once    sync.Once
	initErr error
}

// ContextWithAuth implements Provider.
func (b *baseProvider) ContextWithAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	err := b.Init()
	if err != nil {
		b.logger.Fatal(err)
	}
	return b.impl.ContextWithAuth(ctx)
}

// ContextWithoutAuth implements Provider.
func (b *baseProvider) ContextWithoutAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	err := b.Init()
	if err != nil {
		b.logger.Fatal(err)
	}
	return b.impl.ContextWithoutAuth(ctx)
}

// GetToken implements Provider.
func (b *baseProvider) GetToken() (string, error) {
	err := b.Init()
	if err != nil {
		b.logger.Fatal(err)
	}
	return b.impl.GetToken()
}

// Init implements Provider.
func (b *baseProvider) Init() error {
	b.once.Do(func() {
		// NOTE(shmel1k@): impl can be overridden during provider initialization.
		// For example, in main, when credentials provider is known and initialized.
		if b.impl != nil {
			return
		}

		switch b.opts.Auth.Type {
		case options.Static:
			staticCreds := b.opts.Auth.Creds.(*options.AuthStatic)
			b.impl = NewStatic(staticCreds.User, staticCreds.Password, b.connectionsFactory, b.logger)
		case options.IamToken:
			b.impl = NewIamToken(b.opts.Auth.Creds.(*options.AuthIAMToken).Token)
		case options.IamCreds:
			creds := b.opts.Auth.Creds.(*options.AuthIAMCreds)
			b.impl = NewIamCreds(creds.KeyFilename, creds.Endpoint)
		case options.IamMetadata:
			b.impl = NewMetadata(b.logger)
		case options.None:
			b.initErr = fmt.Errorf("determined credentials to be anonymous. Anonymous credentials are currently unsupported")
		default:
			b.initErr = fmt.Errorf(
				"internal error: authorization type not recognized after options validation, this should never happen",
			)
		}
		if b.initErr == nil {
			b.initErr = b.impl.Init()
		}
	})
	return b.initErr
}

func New(
	opts *command.BaseOptions,
	connectionsFactory connectionsfactory.Factory,
	logger *zap.SugaredLogger,
	impl Provider,
) Provider {
	return &baseProvider{
		impl:               impl,
		opts:               opts,
		connectionsFactory: connectionsFactory,
		logger:             logger,
	}
}
