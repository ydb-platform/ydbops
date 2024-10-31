package restarters

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type TenantSSHRestarter struct {
	sshRestarter

	Opts *TenantSSHOpts
}

const (
	defaultTenantSystemdUnit = "ydb-server-mt-starter"
)

func NewTenantSSHRestarter(logger *zap.SugaredLogger, sshArgs []string, systemdUnit string) *TenantSSHRestarter {
	return &TenantSSHRestarter{
		Opts: &TenantSSHOpts{
			sshOpts: sshOpts{
				sshArgs: sshArgs,
			},
			tenantUnit: systemdUnit,
		},
		sshRestarter: newSSHRestarter(logger),
	}
}

func (r TenantSSHRestarter) RestartNode(node *Ydb_Maintenance.Node) error {
	systemdUnitName := defaultTenantSystemdUnit
	if r.Opts.tenantUnit != "" {
		systemdUnitName = r.Opts.tenantUnit
	}

	return r.restartNodeBySystemdUnit(node, systemdUnitName, r.Opts.sshArgs)
}

func (r TenantSSHRestarter) Filter(spec FilterNodeParams, cluster ClusterNodesInfo) []*Ydb_Maintenance.Node {
	tenantNodes := FilterTenantNodes(cluster.AllNodes)

	preSelectedNodes := PopulateByCommonFields(tenantNodes, spec)

	preSelectedNodes = ExcludeByTenantNames(preSelectedNodes, spec.SelectedTenants, cluster.TenantToNodeIds)

	filteredNodes := ExcludeByCommonFields(preSelectedNodes, spec)

	fmt.Printf("%v\n", r)
	r.logger.Debugf("Tenant SSH Restarter selected following nodes for restart: %+v", filteredNodes)

	return filteredNodes
}
