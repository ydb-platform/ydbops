package restart

import (
	"time"

	"github.com/spf13/pflag"

	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
	"go.uber.org/zap"
)

type Options struct {
	*rolling.RestartOptions
}

func (o *Options) DefineFlags(fs *pflag.FlagSet) {
	o.RestartOptions.DefineFlags(fs)
}

func (o *Options) Run(f cmdutil.Factory) error {
	var storageRestarter restarters.Restarter
	var tenantRestarter restarters.Restarter

	if o.RestartOptions.KubeconfigPath != "" {
		storageRestarter = restarters.NewStorageK8sRestarter(
			options.Logger,
			&restarters.StorageK8sRestarterOptions{
				K8sRestarterOptions: &restarters.K8sRestarterOptions{
					KubeconfigPath:  o.RestartOptions.KubeconfigPath,
					Namespace:       o.RestartOptions.K8sNamespace,
					RestartDuration: time.Duration(o.RestartOptions.RestartDuration) * time.Second,
				},
			},
		)
		tenantRestarter = restarters.NewTenantK8sRestarter(
			options.Logger,
			&restarters.TenantK8sRestarterOptions{
				K8sRestarterOptions: &restarters.K8sRestarterOptions{
					KubeconfigPath:  o.RestartOptions.KubeconfigPath,
					Namespace:       o.RestartOptions.K8sNamespace,
					RestartDuration: time.Duration(o.RestartOptions.RestartDuration) * time.Second,
				},
			},
		)
	} else {
		storageRestarter = restarters.NewStorageSSHRestarter(
			options.Logger,
			o.RestartOptions.SSHArgs,
			o.RestartOptions.CustomSystemdUnitName,
		)
		tenantRestarter = restarters.NewTenantSSHRestarter(
			options.Logger,
			o.RestartOptions.SSHArgs,
			o.RestartOptions.CustomSystemdUnitName,
		)
	}

	bothUnspecified := !o.RestartOptions.Storage && !o.RestartOptions.Tenant

	var executer rolling.Executer
	var err error
	if o.RestartOptions.Storage || bothUnspecified {
		// TODO(shmel1k@): add logger to NewExecuter parameters
		executer = rolling.NewExecuter(o.RestartOptions, zap.S(), f.GetCMSClient(), f.GetDiscoveryClient(), storageRestarter)
		err = executer.Execute()
	}

	if err != nil {
		return err
	}

	if o.RestartOptions.Tenant || bothUnspecified {
		executer = rolling.NewExecuter(o.RestartOptions, zap.S(), f.GetCMSClient(), f.GetDiscoveryClient(), tenantRestarter)
		err = executer.Execute()
	}

	return err
}
