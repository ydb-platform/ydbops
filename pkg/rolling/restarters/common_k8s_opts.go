package restarters

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/pflag"
)

type k8sOpts struct {
	kubeconfigPath string
	namespace      string
}

func (o *k8sOpts) DefineFlags(fs *pflag.FlagSet) {
	defaultKubeconfigPath := filepath.Join("~", ".kube", "config")
	fs.StringVar(&o.kubeconfigPath, "kubeconfig", defaultKubeconfigPath, fmt.Sprintf(
		"Path to kubeconfig file (default: %s)",
		defaultKubeconfigPath,
	))
	fs.StringVar(&o.namespace, "namespace", "", "Namespace of the Storage object to discover pods")
}

func (o *k8sOpts) Validate() error {
	if o.namespace == "" {
		return fmt.Errorf("Please specify a non-empty --namespace.")
	}
  return nil
}
