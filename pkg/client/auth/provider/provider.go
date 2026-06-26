package provider

import (
	"context"

	"google.golang.org/grpc/metadata"
)

type Provider interface {
	ContextWithAuth(context.Context) (context.Context, context.CancelFunc) // TODO(shmel1k@): think about compatibility
	// with ydb-go-sdk
	ContextWithoutAuth(context.Context) (context.Context, context.CancelFunc)

	GetToken() (string, error)
	Init() error
}

type accessTokenProvider struct {
	token string
}

func (p *accessTokenProvider) ContextWithAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	tok, _ := p.GetToken() // TODO(shmel1k@): return err as params
	ctx, cf := context.WithCancel(ctx)
	return metadata.AppendToOutgoingContext(ctx, "x-ydb-auth-ticket", tok), cf
}

func (p *accessTokenProvider) ContextWithoutAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}

func (p *accessTokenProvider) GetToken() (string, error) {
	return p.token, nil
}

func (p *accessTokenProvider) Init() error {
	return nil
}

func NewProviderFromToken(token string) Provider {
	return &accessTokenProvider{token: token}
}
