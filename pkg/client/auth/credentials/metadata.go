package credentials

import (
	"context"
	"sync"

	yc "github.com/ydb-platform/ydb-go-yc-metadata"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

type metadataProvider struct {
	once  sync.Once
	creds *yc.InstanceServiceAccountCredentials
}

// ContextWithAuth implements Provider.
func (m *metadataProvider) ContextWithAuth(ctx context.Context) (context.Context, context.CancelFunc, error) {
	token, err := m.GetToken(ctx)
	if err != nil {
		return nil, nil, err
	}
	ctx1, cf := context.WithCancel(ctx)
	return metadata.AppendToOutgoingContext(ctx1, "x-ydb-auth-ticket", token), cf, nil
}

// ContextWithoutAuth implements Provider.
func (m *metadataProvider) ContextWithoutAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}

// GetToken implements Provider.
func (m *metadataProvider) GetToken(ctx context.Context) (string, error) {
	return m.creds.Token(ctx)
}

// Init implements Provider.
func (m *metadataProvider) Init() error {
	m.once.Do(func() {
		m.creds = yc.NewInstanceServiceAccount(yc.WithURL("http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"))
	})
	return nil
}

func NewMetadata(logger *zap.SugaredLogger) Provider {
	return &metadataProvider{}
}
