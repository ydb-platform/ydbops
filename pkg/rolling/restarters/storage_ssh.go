package restarters

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type StorageSSHRestarter struct {
	sshRestarter

	Opts *StorageSSHOpts
}

func (r StorageSSHRestarter) RestartNode(node *Ydb_Maintenance.Node) error {
	r.logger.Infof("Restarting storage node %s", node.Host)

	systemdUnitName := defaultStorageSystemdUnit
	if r.Opts.storageUnit != "" {
		systemdUnitName = r.Opts.storageUnit
	}

	return r.restartNodeBySystemdUnit(node, systemdUnitName, r.Opts.sshArgs)
}

func NewStorageSSHRestarter(logger *zap.SugaredLogger, sshArgs []string, systemdUnit string) *StorageSSHRestarter {
	return &StorageSSHRestarter{
		Opts: &StorageSSHOpts{
			sshOpts: sshOpts{
				sshArgs: sshArgs,
			},
			storageUnit: systemdUnit,
		},
		sshRestarter: newSSHRestarter(logger),
	}
}

func (r StorageSSHRestarter) Filter(
	spec FilterNodeParams,
	cluster ClusterNodesInfo,
) []*Ydb_Maintenance.Node {
	storageNodes := FilterStorageNodes(cluster.AllNodes, spec.MaxStaticNodeId)

	preSelectedNodes := PopulateByCommonFields(storageNodes, spec)

	filteredNodes := ExcludeByCommonFields(preSelectedNodes, spec)

	r.logger.Debugf("Storage SSH Restarter selected following nodes for restart: %v", filteredNodes)

	return filteredNodes
}
