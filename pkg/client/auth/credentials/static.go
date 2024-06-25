package credentials

import (
	"context"
	"sync"

	"github.com/ydb-platform/ydbops/pkg/client/auth"
	"github.com/ydb-platform/ydbops/pkg/client/connectionsfactory"
	"go.uber.org/zap"
)

type staticCredentialsProvider struct {
	authClient         auth.Client
	connectionsFactory connectionsfactory.Factory
	logger             *zap.SugaredLogger
	once               sync.Once
	initErr            error

	params *staticCredentialsProviderParams
	token  string
}

type staticCredentialsProviderParams struct {
	user     string
	password string
}

// Init implements Provider.
func (s *staticCredentialsProvider) Init() error {
	s.authClient = auth.NewClient(s.logger, s.connectionsFactory)
	return nil
}

// GetToken implements Provider.
func (s *staticCredentialsProvider) GetToken() (string, error) {
	// TODO(shmel1k@): probably, token can change time to time.
	s.once.Do(func() {
		s.token, s.initErr = s.authClient.Auth(s.params.user, s.params.password)
	})
	return s.token, s.initErr
}

// ContextWithAuth implements Provider.
func (s *staticCredentialsProvider) ContextWithAuth(context.Context) (context.Context, context.CancelFunc) {
	panic("unimplemented")
}

// ContextWithoutAuth implements Provider.
func (s *staticCredentialsProvider) ContextWithoutAuth(context.Context) (context.Context, context.CancelFunc) {
	panic("unimplemented")
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
