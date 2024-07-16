package credentials

import (
	"context"
	"strings"
	"sync"

	"github.com/ydb-platform/ydb-go-sdk/v3/credentials"
	yc "github.com/ydb-platform/ydb-go-yc"
	"google.golang.org/grpc/metadata"
)

type iamCredsProvider struct {
	keyfileName string
	iamEndpoint string

	once    sync.Once
	creds   credentials.Credentials
	initErr error
}

// ContextWithAuth implements Provider.
func (i *iamCredsProvider) ContextWithAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	// TODO(shmel1k@): add error handling
	tok, _ := i.GetToken()
	ctx, cf := context.WithCancel(ctx)
	return metadata.AppendToOutgoingContext(ctx,
		"x-ydb-auth-ticket", tok), cf
}

// ContextWithoutAuth implements Provider.
func (i *iamCredsProvider) ContextWithoutAuth(ctx context.Context) (context.Context, context.CancelFunc) {
	// TODO(shmel1k@): remove ContextWithoutAuth method
	return context.WithCancel(ctx)
}

// GetToken implements Provider.
func (i *iamCredsProvider) GetToken() (string, error) {
	return i.creds.Token(context.TODO())
}

// Init implements Provider.
func (i *iamCredsProvider) Init() error {
	i.once.Do(func() {
		sp := strings.Split(i.iamEndpoint, ":")
		if len(sp) == 1 {
			i.iamEndpoint += ":443"
		}
		i.creds, i.initErr = yc.NewClient(
			yc.WithServiceFile(i.keyfileName),
			yc.WithEndpoint(i.iamEndpoint),
		)
	})
	return i.initErr
}

func NewIamCreds(keyfileName, iamEndpoint string) Provider {
	return &iamCredsProvider{
		keyfileName: keyfileName,
		iamEndpoint: iamEndpoint,
	}
}
