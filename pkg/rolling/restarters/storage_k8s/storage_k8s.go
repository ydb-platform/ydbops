package storage_k8s

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"
)

type Restarter struct {

}

func (r Restarter) Filter(spec restarters.FilterNodeParams) []*Ydb_Maintenance.Node {
  return spec.AllNodes
}

func (r Restarter) RestartNode(node *Ydb_Maintenance.Node) error {
  fmt.Println("Restarting baremetal", node.Host)
	return nil
}
