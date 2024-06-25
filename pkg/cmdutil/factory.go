package cmdutil

import (
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"

	"github.com/ydb-platform/ydbops/pkg/client/cms"
	"github.com/ydb-platform/ydbops/pkg/client/discovery"
)

type Factory interface {
	GetCMSClient() cms.Client
	GetDiscoveryClient() discovery.Client
	// GetAuthClient() auth.Client
}

type OperationResponse interface {
	GetOperation() *Ydb_Operations.Operation
}

type factory struct {
	cmsClient       cms.Client
	discoveryClient discovery.Client
}

func New(
	cmsClient cms.Client,
	discoveryClient discovery.Client,
) Factory {
	return &factory{
		cmsClient:       cmsClient,
		discoveryClient: discoveryClient,
	}
}

func (f *factory) GetCMSClient() cms.Client {
	return f.cmsClient
}

func (f *factory) GetDiscoveryClient() discovery.Client {
	return f.discoveryClient
}
