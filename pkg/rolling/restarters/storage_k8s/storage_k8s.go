package storage_k8s

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"
	"go.uber.org/zap"
)

type Restarter struct {

}

func (r Restarter) Filter(_ *zap.SugaredLogger, spec restarters.FilterNodeParams) []*Ydb_Maintenance.Node {
  return spec.AllNodes
}

func (r Restarter) RestartNode(logger *zap.SugaredLogger, node *Ydb_Maintenance.Node) error {
  logger.Info(fmt.Sprintf("Restarting baremetal %s with k8s specifics", node.Host))
	return nil
}
