package restarters

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type TenantBaremetalRestarter struct {
	Opts *TenantBaremetalOpts
}

const (
	defaultTenantSystemdUnit        = "ydb-server-mt-starter"
	internalTenantSystemdUnitPrefix = "kikimr-multi"
)

func NewTenantBaremetalRestarter() *TenantBaremetalRestarter {
	return &TenantBaremetalRestarter{
		Opts: &TenantBaremetalOpts{
			baremetalOpts: baremetalOpts{},
		},
	}
}

func (r TenantBaremetalRestarter) RestartNode(logger *zap.SugaredLogger, node *Ydb_Maintenance.Node) error {
	systemdUnitName := defaultTenantSystemdUnit
	if r.Opts.kikimrTenantUnit {
		systemdUnitName = fmt.Sprintf("%s@{%v}", internalTenantSystemdUnitPrefix, node.Port)
	}

	return restartNodeBySystemdUnit(logger, node, systemdUnitName, r.Opts.sshArgs)
}

func (r TenantBaremetalRestarter) Filter(
	logger *zap.SugaredLogger,
	spec FilterNodeParams,
	cluster ClusterNodesInfo,
) []*Ydb_Maintenance.Node {
	allTenantNodes := FilterTenantNodes(cluster.AllNodes)

	selectedNodes := FilterByNodeIdOrFQDN(allTenantNodes, spec)

	logger.Debugf("Tenant Baremetal Restarter selected following nodes for restart: %v", selectedNodes)

	return selectedNodes
}
