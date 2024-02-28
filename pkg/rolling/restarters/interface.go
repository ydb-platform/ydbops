package restarters

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
)

type RestarterInterface interface {
	Filter (spec FilterNodeParams) []*Ydb_Maintenance.Node
	RestartNode(node *Ydb_Maintenance.Node) error
}

type FilterNodeParams struct {
	AllTenants      []string
	AllNodes        []*Ydb_Maintenance.Node
	SelectedTenants []string
	SelectedNodeIds []uint32
}
