package provider

import (
	"context"

	"google.golang.org/grpc/metadata"
)

type Provider interface {
	ContextWithAuth(context.Context) (context.Context, context.CancelFunc, error)
	ContextWithoutAuth(context.Context) (context.Context, context.CancelFunc)

	GetToken(context.Context) (string, error)
	Init() error
}

type accessTokenProvider struct {
	token string
}

func (p *accessTokenProvider) ContextWithAuth(ctx context.Context) (context.Context, context.CancelFunc, error) {
	tok, err := p.GetToken(ctx)
	if err != nil {
		return nil, nil, err
	}
	ctx, cf := context.WithCancel(ctx)
	return metadata.AppendToOutgoingContext(ctx, "x-ydb-auth-ticket", tok), cf, nil
}

func (p *accessTokenProvider) ContextWithoutAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}

func (p *accessTokenProvider) GetToken(ctx context.Context) (string, error) {
	return p.token, ctx.Err()
}

func (p *accessTokenProvider) Init() error {
	return nil
}

func NewProviderFromToken(token string) Provider {
	return &accessTokenProvider{token: token}
}
