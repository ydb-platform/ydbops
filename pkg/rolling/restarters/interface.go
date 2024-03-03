package restarters

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type RestarterInterface interface {
	Filter(logger *zap.SugaredLogger, spec *FilterNodeParams) []*Ydb_Maintenance.Node
	RestartNode(logger *zap.SugaredLogger, node *Ydb_Maintenance.Node) error
}

type FilterNodeParams struct {
	AllTenants        []string
	AllNodes          []*Ydb_Maintenance.Node
	SelectedTenants   []string
	SelectedNodeIds   []uint32
	SelectedHostFQDNs []string
}
