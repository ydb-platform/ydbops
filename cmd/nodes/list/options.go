package list

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/ydb-platform/ydbops/pkg/cmdutil"
)

type Options struct {
}

const (
	DefaultMaintenanceDurationSeconds = 3600
)

func (o *Options) DefineFlags(fs *pflag.FlagSet) {
}

func (o *Options) Validate() error {
	return nil
}

func (o *Options) Run(f cmdutil.Factory) error {
	nodes, err := f.GetCMSClient().Nodes()
	if err != nil {
		return err
	}

	fmt.Printf("--- begin nodes ---\n")
	for _, node := range nodes {
		nodeType := "UNKNOWN"
		if node.GetStorage() != nil {
			nodeType = "STORAGE"
		} else if node.GetDynamic() != nil {
			nodeType = "DATABASE"
		}
		fmt.Printf("%d\t%s:%d\t%s\n", node.GetNodeId(), node.GetHost(), node.GetPort(), nodeType)
	}
	fmt.Printf("--- end nodes ---\n")

	return nil
}
