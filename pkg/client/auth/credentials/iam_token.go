package credentials

import (
	"context"

	"google.golang.org/grpc/metadata"
)

type iamTokenCredentialsProvider struct {
	token string
}

// ContextWithAuth implements Provider.
func (i *iamTokenCredentialsProvider) ContextWithAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	tok, _ := i.GetToken() // TODO(shmel1k@): return err as params
	ctx, cf := context.WithCancel(ctx)
	return metadata.AppendToOutgoingContext(ctx,
		"x-ydb-auth-ticket", tok), cf
}

// ContextWithoutAuth implements Provider.
func (i *iamTokenCredentialsProvider) ContextWithoutAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}

// GetToken implements Provider.
func (i *iamTokenCredentialsProvider) GetToken() (string, error) {
	return i.token, nil
}

// Init implements Provider.
func (i *iamTokenCredentialsProvider) Init() error {
	return nil
}

func NewIamToken(token string) Provider {
	return &iamTokenCredentialsProvider{
		token: token,
	}
}
