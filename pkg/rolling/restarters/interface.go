package restarters

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type Restarter interface {
	Filter(logger *zap.SugaredLogger, spec FilterNodeParams, cluster ClusterNodesInfo) []*Ydb_Maintenance.Node
	RestartNode(logger *zap.SugaredLogger, node *Ydb_Maintenance.Node) error
}

type ClusterNodesInfo struct {
	AllTenants []string
	AllNodes   []*Ydb_Maintenance.Node
}

type FilterNodeParams struct {
	Version           string
	SelectedTenants   []string
	SelectedNodeIds   []uint32
	SelectedHostFQDNs []string
}
