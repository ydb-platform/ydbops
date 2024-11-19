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
	opts *options.FilteringOptions,
	sshArgs []string,
	customSystemdUnitName string,
	restartDuration int,
) (restarters.Restarter, restarters.Restarter) {
	if opts.KubeconfigPath != "" {
		storageRestarter := restarters.NewStorageK8sRestarter(
			options.Logger,
			&restarters.StorageK8sRestarterOptions{
				K8sRestarterOptions: &restarters.K8sRestarterOptions{
					KubeconfigPath:  opts.KubeconfigPath,
					Namespace:       opts.K8sNamespace,
					RestartDuration: time.Duration(restartDuration) * time.Second,
				},
			},
		)
		tenantRestarter := restarters.NewTenantK8sRestarter(
			options.Logger,
			&restarters.TenantK8sRestarterOptions{
				K8sRestarterOptions: &restarters.K8sRestarterOptions{
					KubeconfigPath:  opts.KubeconfigPath,
					Namespace:       opts.K8sNamespace,
					RestartDuration: time.Duration(restartDuration) * time.Second,
				},
			},
		)
		return storageRestarter, tenantRestarter
	}

	storageRestarter := restarters.NewStorageSSHRestarter(
		options.Logger,
		sshArgs,
		customSystemdUnitName,
	)
	tenantRestarter := restarters.NewTenantSSHRestarter(
		options.Logger,
		sshArgs,
		customSystemdUnitName,
	)
	return storageRestarter, tenantRestarter
}

func (o *Options) Run(f cmdutil.Factory) error {
	storageRestarter, tenantRestarter := PrepareRestarters(
		&o.FilteringOptions,
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
