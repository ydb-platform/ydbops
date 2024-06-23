package credentials

import "context"

type Provider interface {
	ContextWithAuth(context.Context) (context.Context, context.CancelFunc) // TODO(shmel1k@): think about compatibility
	// with ydb-go-sdk
	ContextWithoutAuth(context.Context) (context.Context, context.CancelFunc)

	GetToken() string
}
