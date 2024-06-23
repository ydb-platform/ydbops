package cmdutil

import "github.com/ydb-platform/ydbops/pkg/client/cms"

type Factory interface {
	CreateCMSClient() (cms.Client, error)
	CreateDiscoveryClient() error
}

func NewFactory() Factory {
	return nil
}
