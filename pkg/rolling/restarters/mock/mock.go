package mock

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"

	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"
)

const (
	name = "mock"
)

type mock struct{}

func (m *mock) Filter(spec restarters.FilterNodeParams) []*Ydb_Maintenance.Node {
	// take only storage nodes
	return util.FilterBy(spec.AllNodes,
		func(node *Ydb_Maintenance.Node) bool {
			return node.GetStorage() != nil
		},
	)
}
func (m *mock) RestartNode(*Ydb_Maintenance.Node) error { return nil }
