package restarters

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type TenantBaremetalRestarter struct {
	Opts   *TenantBaremetalOpts
	logger *zap.SugaredLogger
}

const (
	defaultTenantSystemdUnit        = "ydb-server-mt-starter"
	internalTenantSystemdUnitPrefix = "kikimr-multi"
)

func NewTenantBaremetalRestarter(logger *zap.SugaredLogger) *TenantBaremetalRestarter {
	return &TenantBaremetalRestarter{
		Opts: &TenantBaremetalOpts{
			baremetalOpts: baremetalOpts{},
		},
		logger: logger,
	}
}

func (r TenantBaremetalRestarter) RestartNode(node *Ydb_Maintenance.Node) error {
	systemdUnitName := defaultTenantSystemdUnit
	if r.Opts.kikimrTenantUnit {
		systemdUnitName = fmt.Sprintf("%s@{%v}", internalTenantSystemdUnitPrefix, node.Port)
	}

	return restartNodeBySystemdUnit(r.logger, node, systemdUnitName, r.Opts.sshArgs)
}

func (r TenantBaremetalRestarter) Filter(spec FilterNodeParams, cluster ClusterNodesInfo) []*Ydb_Maintenance.Node {
	tenantNodes := FilterTenantNodes(cluster.AllNodes)

	preSelectedNodes := DoDefaultPopulate(tenantNodes, spec)

	filteredNodes := DoDefaultExclude(preSelectedNodes, spec)

	r.logger.Debugf("Tenant Baremetal Restarter selected following nodes for restart: %v", filteredNodes)

	return filteredNodes
}
