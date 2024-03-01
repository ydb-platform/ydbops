package storage_baremetal

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"
)

type Restarter struct {
	Opts *Opts
}

func (r Restarter) RestartNode(node *Ydb_Maintenance.Node) error {
  fmt.Println("Restarting baremetal", node.Host)
	fmt.Println("By the way, ssh args are", r.Opts.SSHArgs)
	return nil
}

func New() *Restarter {
	return &Restarter{ Opts: &Opts{} }
}

func (r Restarter) Filter(spec restarters.FilterNodeParams) []*Ydb_Maintenance.Node {
	allStorageNodes := util.FilterBy(spec.AllNodes,
		func(node *Ydb_Maintenance.Node) bool {
			return node.GetStorage() != nil
		},
	)

	selectedNodes := []*Ydb_Maintenance.Node{}

	if len(spec.SelectedNodeIds) > 0 {
		selectedNodes = append(selectedNodes, util.FilterBy(allStorageNodes,
			func(node *Ydb_Maintenance.Node) bool {
				return util.Contains(spec.SelectedNodeIds, node.NodeId)
			},
		)...)
	}

	if len(spec.SelectedHostFQDNs) > 0 {
		selectedNodes = append(selectedNodes, util.FilterBy(allStorageNodes,
			func(node *Ydb_Maintenance.Node) bool {
				return util.Contains(spec.SelectedHostFQDNs, node.Host)
			},
		)...)
	}

	return selectedNodes
}
