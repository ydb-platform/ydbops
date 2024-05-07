package restarters

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type TenantK8sRestarter struct {
	Opts *TenantK8sOpts

	k8sRestarter
}

func NewTenantK8sRestarter(logger *zap.SugaredLogger, kubeconfigPath, namespace string) *TenantK8sRestarter {
	return &TenantK8sRestarter{
		Opts: &TenantK8sOpts{
			k8sOpts: k8sOpts{
				kubeconfigPath: kubeconfigPath,
				namespace:      namespace,
			},
		},
		k8sRestarter: newK8sRestarter(logger),
	}
}

func (r TenantK8sRestarter) RestartNode(node *Ydb_Maintenance.Node) error {
	return r.restartNodeByRestartingPod(node.Host, r.Opts.namespace)
}

func (r *TenantK8sRestarter) Filter(spec FilterNodeParams, cluster ClusterNodesInfo) []*Ydb_Maintenance.Node {
	databaseLabelSelector := "app.kubernetes.io/component=dynamic-node"

	r.prepareK8sState(r.Opts.kubeconfigPath, databaseLabelSelector, r.Opts.namespace)

	tenantNodes := FilterTenantNodes(cluster.AllNodes)

	selectedNodes := populateWithK8sRules(tenantNodes, spec, r.FQDNToPodName)

	selectedNodes = ExcludeByTenantNames(selectedNodes, spec.SelectedTenants, cluster.TenantToNodeIds)

	filteredNodes := ExcludeByCommonFields(selectedNodes, spec)

	r.logger.Debugf("Tenant K8s restarter selected following nodes for restart: %v", filteredNodes)

	return selectedNodes
}
