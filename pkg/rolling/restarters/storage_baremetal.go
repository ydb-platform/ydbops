package restarters

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type StorageBaremetalRestarter struct {
	baremetalRestarter

	Opts   *StorageBaremetalOpts
}

func (r StorageBaremetalRestarter) RestartNode(node *Ydb_Maintenance.Node) error {
	r.logger.Infof("Restarting storage node %s with ssh-args %v", node.Host, r.Opts.sshArgs)

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

func NewStorageBaremetalRestarter(logger *zap.SugaredLogger) *StorageBaremetalRestarter {
	return &StorageBaremetalRestarter{
		Opts: &StorageBaremetalOpts{
			baremetalOpts: baremetalOpts{},
		},
		baremetalRestarter: newBaremetalRestarter(logger),
	}
}

func (r StorageBaremetalRestarter) Filter(
	spec FilterNodeParams,
	cluster ClusterNodesInfo,
) []*Ydb_Maintenance.Node {
	storageNodes := FilterStorageNodes(cluster.AllNodes)

	preSelectedNodes := PopulateByCommonFields(storageNodes, spec)

	filteredNodes := ExcludeByCommonFields(preSelectedNodes, spec)

	r.logger.Debugf("Storage Baremetal Restarter selected following nodes for restart: %v", filteredNodes)

	return filteredNodes
}
