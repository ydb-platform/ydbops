package restarters

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

type k8sOpts struct {
	kubeconfigPath string
	namespace      string
}

var defaultKubeconfigPath = filepath.Join("~", ".kube", "config")

func (o *k8sOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.kubeconfigPath, "kubeconfig", "", fmt.Sprintf(
		"Path to kubeconfig file (default: %s)",
		defaultKubeconfigPath,
	))
	fs.StringVar(&o.namespace, "namespace", "", "Namespace of the Storage object to discover pods")
}

func (o *k8sOpts) Validate() error {
	if o.kubeconfigPath == "" {
		zap.S().Infof("--kubeconfig not specified, assuming default %s", defaultKubeconfigPath)
		o.kubeconfigPath = defaultKubeconfigPath
	}

	if o.namespace == "" {
		return fmt.Errorf("please specify a non-empty --namespace")
	}
	return nil
}
