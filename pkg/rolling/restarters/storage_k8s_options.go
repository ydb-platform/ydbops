package restarters

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/pflag"
)

type StorageK8sOpts struct {
	KubeconfigPath string
	Storage        string
	Namespace      string
}

func (o *StorageK8sOpts) DefineFlags(fs *pflag.FlagSet) {
	defaultKubeconfigPath := filepath.Join("~", ".kube", "config")
	fs.StringVar(&o.KubeconfigPath, "kubeconfig", defaultKubeconfigPath, fmt.Sprintf(
		"Path to kubeconfig file (default: %s)",
		defaultKubeconfigPath,
	))
	fs.StringVar(&o.Storage, "storage", "", "Storage resource name to restart")
	fs.StringVar(&o.Namespace, "namespace", "", "Namespace of the Storage object to discover pods")
}

func (o *StorageK8sOpts) Validate() error {
	if o.Namespace == "" {
		return fmt.Errorf("Please specify a non-empty --namespace.")
	}
	if o.Storage == "" {
		return fmt.Errorf("Please specify a non-empty --storage.")
	}
	return nil
}
