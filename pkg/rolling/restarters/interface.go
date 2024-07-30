package restarters

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydbops/pkg/options"
)

type Restarter interface {
	Filter(
		spec FilterNodeParams,
		cluster ClusterNodesInfo,
	) []*Ydb_Maintenance.Node
	RestartNode(node *Ydb_Maintenance.Node) error
}

type ClusterNodesInfo struct {
	AllNodes        []*Ydb_Maintenance.Node
	TenantToNodeIds map[string][]uint32
}

type FilterNodeParams struct {
	Version         options.VersionSpec
	StartedTime     *options.StartedTime
	ExcludeHosts    []string
	SelectedTenants []string
	SelectedNodeIds []uint32
	SelectedHosts   []string
	MaxStaticNodeId uint32
}
