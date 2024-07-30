package credentials

import (
	"context"
	"sync"

	"github.com/ydb-platform/ydbops/pkg/client/auth"
	"github.com/ydb-platform/ydbops/pkg/client/connectionsfactory"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

type staticCredentialsProvider struct {
	authClient         auth.Client
	connectionsFactory connectionsfactory.Factory
	logger             *zap.SugaredLogger

	initOnce sync.Once

	tokenOnce sync.Once
	tokenErr  error

	params *staticCredentialsProviderParams
	token  string
}

type staticCredentialsProviderParams struct {
	user     string
	password string
}

// Init implements Provider.
func (s *staticCredentialsProvider) Init() error {
	s.initOnce.Do(func() {
		s.authClient = auth.NewClient(s.logger, s.connectionsFactory)
	})
	return nil
}

// GetToken implements Provider.
func (s *staticCredentialsProvider) GetToken() (string, error) {
	// TODO(shmel1k@): probably, token can change time to time.
	s.tokenOnce.Do(func() {
		s.token, s.tokenErr = s.authClient.Auth(s.params.user, s.params.password)
	})
	return s.token, s.tokenErr
}

// ContextWithAuth implements Provider.
func (s *staticCredentialsProvider) ContextWithAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	tok, _ := s.GetToken() // TODO(shmel1k@): return err as params
	ctx, cf := context.WithCancel(ctx)
	return metadata.AppendToOutgoingContext(ctx,
		"x-ydb-auth-ticket", tok), cf
}

// ContextWithoutAuth implements Provider.
func (s *staticCredentialsProvider) ContextWithoutAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}

func NewStatic(
	login, password string,
	connectionsFactory connectionsfactory.Factory,
	logger *zap.SugaredLogger,
) Provider {
	return &staticCredentialsProvider{
		connectionsFactory: connectionsFactory,
		logger:             logger,
		params: &staticCredentialsProviderParams{
			user:     login,
			password: password,
		},
	}
}
