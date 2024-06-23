package credentials

import (
	"context"
	"sync"

	"github.com/ydb-platform/ydbops/pkg/client/auth"
)

type staticCredentialsProvider struct {
	authClient auth.Client
	once       sync.Once
}

// GetToken implements Provider.
func (s *staticCredentialsProvider) GetToken() string {
	s.authClient.Auth("", "")
	return ""
}

// ContextWithAuth implements Provider.
func (s *staticCredentialsProvider) ContextWithAuth(context.Context) (context.Context, context.CancelFunc) {
	panic("unimplemented")
}

// ContextWithoutAuth implements Provider.
func (s *staticCredentialsProvider) ContextWithoutAuth(context.Context) (context.Context, context.CancelFunc) {
	panic("unimplemented")
}

func NewStatic(authClient auth.Client) Provider {
	return &staticCredentialsProvider{
		authClient: authClient,
	}
}
