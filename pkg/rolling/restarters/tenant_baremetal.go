package restarters

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type TenantBaremetalRestarter struct {
	baremetalRestarter
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
		baremetalRestarter: newBaremetalRestarter(logger),
	}
}

func (r TenantBaremetalRestarter) RestartNode(node *Ydb_Maintenance.Node) error {
	systemdUnitName := defaultTenantSystemdUnit
	if r.Opts.kikimrTenantUnit {
		systemdUnitName = fmt.Sprintf("%s@{%v}", internalTenantSystemdUnitPrefix, node.Port)
	}

	return r.restartNodeBySystemdUnit(node, systemdUnitName, r.Opts.sshArgs)
}

func (r TenantBaremetalRestarter) Filter(spec FilterNodeParams, cluster ClusterNodesInfo) []*Ydb_Maintenance.Node {
	tenantNodes := FilterTenantNodes(cluster.AllNodes)

	preSelectedNodes := PopulateByCommonFields(tenantNodes, spec)

	selectedByTenantName := PopulateByTenantNames(tenantNodes, spec.SelectedTenants, cluster.TenantToNodeIds)

	preSelectedNodes = MergeAndUnique(preSelectedNodes, selectedByTenantName)

	filteredNodes := ExcludeByCommonFields(preSelectedNodes, spec)

	r.logger.Debugf("Tenant Baremetal Restarter selected following nodes for restart: %v", filteredNodes)

	return filteredNodes
}
