package credentials

import (
	"context"
	"fmt"
	"os"
	"strings"

	sdkcredentials "github.com/ydb-platform/ydb-go-sdk/v3/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/ydb-platform/ydbops/pkg/options"
)

type fileTokenSource struct {
	path      string
	tokenType string
}

func (s fileTokenSource) Token() (sdkcredentials.Token, error) {
	// Open the path on every exchange so externally rotated files and symlinks are followed.
	data, err := os.ReadFile(s.path)
	if err != nil {
		return sdkcredentials.Token{}, fmt.Errorf("read OAuth2 token file %q: %w", s.path, err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return sdkcredentials.Token{}, fmt.Errorf("OAuth2 token file %q is empty", s.path)
	}
	return sdkcredentials.Token{Token: token, TokenType: s.tokenType}, nil
}

type oauth2Provider struct {
	creds sdkcredentials.Credentials
}

func NewOAuth2TokenExchange(config options.AuthOAuth2TokenExchange) (Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	opts := []sdkcredentials.Oauth2TokenExchangeCredentialsOption{
		sdkcredentials.WithTokenEndpoint(config.TokenEndpoint),
	}
	if config.SubjectTokenFile != "" {
		opts = append(opts, sdkcredentials.WithSubjectToken(fileTokenSource{config.SubjectTokenFile, config.SubjectTokenType}))
	} else {
		opts = append(opts, sdkcredentials.WithFixedSubjectToken(config.SubjectToken, config.SubjectTokenType))
	}
	if config.ActorTokenFile != "" {
		opts = append(opts, sdkcredentials.WithActorToken(fileTokenSource{config.ActorTokenFile, config.ActorTokenType}))
	}
	creds, err := sdkcredentials.NewOauth2TokenExchangeCredentials(opts...)
	if err != nil {
		return nil, err
	}
	return &oauth2Provider{creds: creds}, nil
}

func (p *oauth2Provider) Init() error { return nil }

func (p *oauth2Provider) GetToken(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return p.creds.Token(ctx)
}

func (p *oauth2Provider) ContextWithAuth(ctx context.Context) (context.Context, context.CancelFunc, error) {
	token, err := p.GetToken(ctx)
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	return metadata.AppendToOutgoingContext(ctx, "x-ydb-auth-ticket", token), cancel, nil
}

func (p *oauth2Provider) ContextWithoutAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}
