package restarters

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type TenantSSHRestarter struct {
	sshRestarter
	Opts   *TenantSSHOpts
	logger *zap.SugaredLogger
}

const (
	defaultTenantSystemdUnit        = "ydb-server-mt-starter"
	internalTenantSystemdUnitPrefix = "kikimr-multi"
)

func NewTenantSSHRestarter(logger *zap.SugaredLogger) *TenantSSHRestarter {
	return &TenantSSHRestarter{
		Opts: &TenantSSHOpts{
			sshOpts: sshOpts{},
		},
		sshRestarter: newSSHRestarter(logger),
	}
}

func (r TenantSSHRestarter) RestartNode(node *Ydb_Maintenance.Node) error {
	systemdUnitName := defaultTenantSystemdUnit
	if r.Opts.kikimrTenantUnit {
		systemdUnitName = fmt.Sprintf("%s@{%v}", internalTenantSystemdUnitPrefix, node.Port)
	}

	return r.restartNodeBySystemdUnit(node, systemdUnitName, r.Opts.sshArgs)
}

func (r TenantSSHRestarter) Filter(spec FilterNodeParams, cluster ClusterNodesInfo) []*Ydb_Maintenance.Node {
	tenantNodes := FilterTenantNodes(cluster.AllNodes)

	preSelectedNodes := PopulateByCommonFields(tenantNodes, spec)

	selectedByTenantName := PopulateByTenantNames(tenantNodes, spec.SelectedTenants, cluster.TenantToNodeIds)

	preSelectedNodes = MergeAndUnique(preSelectedNodes, selectedByTenantName)

	filteredNodes := ExcludeByCommonFields(preSelectedNodes, spec)

	r.logger.Debugf("Tenant SSH Restarter selected following nodes for restart: %v", filteredNodes)

	return filteredNodes
}
