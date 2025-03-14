package restarters

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type StorageK8sRestarter struct {
	k8sRestarter

	Opts *StorageK8sOpts
}

type StorageK8sRestarterOptions struct {
	*K8sRestarterOptions
}

func NewStorageK8sRestarter(logger *zap.SugaredLogger, params *StorageK8sRestarterOptions) *StorageK8sRestarter {
	return &StorageK8sRestarter{
		Opts: &StorageK8sOpts{
			k8sOpts: k8sOpts{
				kubeconfigPath: params.KubeconfigPath,
				namespace:      params.Namespace,
			},
		},
		k8sRestarter: newK8sRestarter(logger, &k8sRestarterOptions{
			restartDuration: params.RestartDuration,
		}),
	}
}

func (r StorageK8sRestarter) RestartNode(node *Ydb_Maintenance.Node) error {
	return r.restartNodeByRestartingPod(node.Host, node.Port, r.Opts.namespace)
}

func populateWithK8sRules(
	nodes []*Ydb_Maintenance.Node,
	spec FilterNodeParams,
	fqdnToPodName map[string]string,
) []*Ydb_Maintenance.Node {
	if isInclusiveFilteringUnspecified(spec) {
		return nodes
	}

	selectedNodes := []*Ydb_Maintenance.Node{}

	// TODO make this linear
	for _, node := range nodes {
		for _, selectedHostFQDN := range spec.SelectedHosts {
			if selectedHostFQDN == fqdnToPodName[node.Host] {
				selectedNodes = append(selectedNodes, node)
				continue
			}
		}
	}

	return selectedNodes
}

func applyStorageK8sFilteringRules(
	spec FilterNodeParams,
	cluster ClusterNodesInfo,
	fqdnToPodName map[string]string,
) []*Ydb_Maintenance.Node {
	allStorageNodes := FilterStorageNodes(cluster.AllNodes, spec.MaxStaticNodeID)

	selectedByCMSNodes := PopulateByCommonFields(allStorageNodes, spec)
	selectedByK8sNodes := populateWithK8sRules(allStorageNodes, spec, fqdnToPodName)
	selectedNodes := MergeAndUnique(selectedByCMSNodes, selectedByK8sNodes)

	filteredNodes := ExcludeByCommonFields(selectedNodes, spec)

	return filteredNodes
}

func (r *StorageK8sRestarter) Filter(
	spec FilterNodeParams,
	cluster ClusterNodesInfo,
) []*Ydb_Maintenance.Node {
	storageLabelSelector := "app.kubernetes.io/component=storage-node"
	r.prepareK8sState(r.Opts.kubeconfigPath, storageLabelSelector, r.Opts.namespace)

	filteredNodes := applyStorageK8sFilteringRules(spec, cluster, r.FQDNToPodName)

	r.logger.Debugf("Storage K8s restarter selected following nodes for restart: %+v", filteredNodes)
	return filteredNodes
}
