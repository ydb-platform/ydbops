package credentials

import "context"

type iamTokenCredentialsProvider struct {
	token string
}

// ContextWithAuth implements Provider.
func (i *iamTokenCredentialsProvider) ContextWithAuth(context.Context) (context.Context, context.CancelFunc) {
	panic("unimplemented")
}

// ContextWithoutAuth implements Provider.
func (i *iamTokenCredentialsProvider) ContextWithoutAuth(context.Context) (context.Context, context.CancelFunc) {
	panic("unimplemented")
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
