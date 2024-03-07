package restarters

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
)

type Restarter interface {
	Filter(spec FilterNodeParams, cluster ClusterNodesInfo) []*Ydb_Maintenance.Node
	RestartNode(node *Ydb_Maintenance.Node) error
}

type ClusterNodesInfo struct {
	AllTenants []string
	AllNodes   []*Ydb_Maintenance.Node
}

type FilterNodeParams struct {
	Version           string
	ExcludeHosts      []string
	SelectedTenants   []string
	SelectedNodeIds   []uint32
	SelectedHostFQDNs []string
}
