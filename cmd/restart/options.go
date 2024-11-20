package restart

import (
	"time"

	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

type Options struct {
	*rolling.RestartOptions
}

func (o *Options) DefineFlags(fs *pflag.FlagSet) {
	o.RestartOptions.DefineFlags(fs)
}

func PrepareRestarters(
	opts *options.TargetingOptions,
	sshArgs []string,
	customSystemdUnitName string,
	restartDuration int,
) (storage, tenant restarters.Restarter) {
	if opts.KubeconfigPath != "" {
		storage = restarters.NewStorageK8sRestarter(
			options.Logger,
			&restarters.StorageK8sRestarterOptions{
				K8sRestarterOptions: &restarters.K8sRestarterOptions{
					KubeconfigPath:  opts.KubeconfigPath,
					Namespace:       opts.K8sNamespace,
					RestartDuration: time.Duration(restartDuration) * time.Second,
				},
			},
		)
		tenant = restarters.NewTenantK8sRestarter(
			options.Logger,
			&restarters.TenantK8sRestarterOptions{
				K8sRestarterOptions: &restarters.K8sRestarterOptions{
					KubeconfigPath:  opts.KubeconfigPath,
					Namespace:       opts.K8sNamespace,
					RestartDuration: time.Duration(restartDuration) * time.Second,
				},
			},
		)
		return storage, tenant
	}

	storage = restarters.NewStorageSSHRestarter(
		options.Logger,
		sshArgs,
		customSystemdUnitName,
	)
	tenant = restarters.NewTenantSSHRestarter(
		options.Logger,
		sshArgs,
		customSystemdUnitName,
	)
	return storage, tenant
}

func (o *Options) Run(f cmdutil.Factory) error {
	storageRestarter, tenantRestarter := PrepareRestarters(
		&o.TargetingOptions,
		o.SSHArgs,
		o.CustomSystemdUnitName,
		o.RestartDuration,
	)

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
