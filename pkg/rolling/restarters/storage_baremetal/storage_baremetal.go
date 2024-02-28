package storage_baremetal

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"
)

type Restarter struct {
	opts *opts
}

func (r Restarter) RestartNode(node *Ydb_Maintenance.Node) error {
  fmt.Println("Restarting baremetal", node.Host)
	fmt.Println("By the way, ssh args are", r.opts.SSHArgs)
	return nil
}

// TODO After the refactoring, this `New` function got a bit outdated
// func New(opts *opts) restarters.RestarterInterface {
// 	return Restarter{ opts: opts }
// }

func (r Restarter) Filter(spec restarters.FilterNodeParams) []*Ydb_Maintenance.Node {
	nodes := util.FilterBy(spec.AllNodes,
		func(node *Ydb_Maintenance.Node) bool {
			return node.GetStorage() != nil
		},
	)

	if len(spec.SelectedNodeIds) > 0 {
		nodes = util.FilterBy(nodes,
			func(node *Ydb_Maintenance.Node) bool {
				return util.Contains(spec.SelectedNodeIds, node.NodeId)
			},
		)
	}

	return nodes
}
