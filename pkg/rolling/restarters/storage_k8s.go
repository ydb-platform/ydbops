package restarters

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type StorageK8sRestarter struct {
}

func (r StorageK8sRestarter) Filter(
	logger *zap.SugaredLogger,
	spec FilterNodeParams,
	cluster ClusterNodesInfo,
) []*Ydb_Maintenance.Node {
	return cluster.AllNodes
}

func (r StorageK8sRestarter) RestartNode(logger *zap.SugaredLogger, node *Ydb_Maintenance.Node) error {
	logger.Info(fmt.Sprintf("Restarting baremetal %s with k8s specifics", node.Host))
	return nil
}
