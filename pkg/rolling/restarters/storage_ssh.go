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

	// It is theoretically possible to guess the systemd-unit, but it is a fragile
	// solution. tarasov-egor@ will keep it here during development time for reference:
	//
	// YDBD_PORT=2135
	// YDBD_PID=$(sudo lsof -i :$YDBD_PORT | grep LISTEN | awk '{print $2}' | head -n 1)
	// YDBD_UNIT=$(sudo ps -A -o'pid,unit' | grep $YDBD_PID | awk '{print $2}')
	// sudo systemctl restart $YDBD_UNIT

	systemdUnitName := defaultStorageSystemdUnit
	if r.Opts.kikimrStorageUnit {
		systemdUnitName = internalStorageSystemdUnit
	}

	return r.restartNodeBySystemdUnit(node, systemdUnitName, r.Opts.sshArgs)
}

func NewStorageSSHRestarter(logger *zap.SugaredLogger) *StorageSSHRestarter {
	return &StorageSSHRestarter{
		Opts: &StorageSSHOpts{
			sshOpts: sshOpts{},
		},
		sshRestarter: newSSHRestarter(logger),
	}
}

func (r StorageSSHRestarter) Filter(
	spec FilterNodeParams,
	cluster ClusterNodesInfo,
) []*Ydb_Maintenance.Node {
	storageNodes := FilterStorageNodes(cluster.AllNodes)

	preSelectedNodes := PopulateByCommonFields(storageNodes, spec)

	filteredNodes := ExcludeByCommonFields(preSelectedNodes, spec)

	r.logger.Debugf("Storage SSH Restarter selected following nodes for restart: %v", filteredNodes)

	return filteredNodes
}
