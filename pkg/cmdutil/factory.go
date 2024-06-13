package cmdutil

import (
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"

	"github.com/ydb-platform/ydbops/pkg/client/cms"
	"github.com/ydb-platform/ydbops/pkg/client/discovery"
	"github.com/ydb-platform/ydbops/pkg/command"
)

type Factory interface {
	GetCMSClient() cms.Client
	GetDiscoveryClient() discovery.Client
	GetBaseOptions() *command.BaseOptions
	// GetAuthClient() auth.Client
}

type OperationResponse interface {
	GetOperation() *Ydb_Operations.Operation
}

type factory struct {
	cmsClient       cms.Client
	discoveryClient discovery.Client
	opts            *command.BaseOptions
}

func New(
	opts *command.BaseOptions,
	cmsClient cms.Client,
	discoveryClient discovery.Client,
) Factory {
	return &factory{
		opts:            opts,
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

func (f *factory) GetBaseOptions() *command.BaseOptions {
	return f.opts
}
