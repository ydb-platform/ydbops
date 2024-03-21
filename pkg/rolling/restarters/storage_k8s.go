package restarters

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/internal/collections"
)

type StorageK8sRestarter struct {
	k8sRestarter

	Opts *StorageK8sOpts
}

func NewStorageK8sRestarter(logger *zap.SugaredLogger, kubeconfigPath, namespace string) *StorageK8sRestarter {
	return &StorageK8sRestarter{
		Opts: &StorageK8sOpts{
			k8sOpts: k8sOpts{
				kubeconfigPath: kubeconfigPath,
				namespace:      namespace,
			},
		},
		k8sRestarter: newK8sRestarter(logger),
	}
}

func (r StorageK8sRestarter) RestartNode(node *Ydb_Maintenance.Node) error {
	return r.restartNodeByRestartingPod(node.Host, r.Opts.namespace)
}

func populateWithK8sRules(nodes []*Ydb_Maintenance.Node, spec FilterNodeParams, hostnameToPod map[string]string) []*Ydb_Maintenance.Node {
	selectedNodes := []*Ydb_Maintenance.Node{}

	if len(spec.SelectedNodeIds) == 0 && len(spec.SelectedHostFQDNs) == 0 {
		selectedNodes = nodes
	} else {
		selectedNodes = append(
			selectedNodes,
			FilterByNodeIds(nodes, spec.SelectedNodeIds)...,
		)

		for _, node := range nodes {
			selectedHostFQDNsMap := collections.ToIndexMap(spec.SelectedHostFQDNs)

			if _, present := selectedHostFQDNsMap[node.Host]; present {
				selectedNodes = append(selectedNodes, node)
				continue
			}

			if _, present := selectedHostFQDNsMap[hostnameToPod[node.Host]]; present {
				selectedNodes = append(selectedNodes, node)
				continue
			}
		}
	}

	return selectedNodes
}

func (r *StorageK8sRestarter) Filter(
	spec FilterNodeParams,
	cluster ClusterNodesInfo,
) []*Ydb_Maintenance.Node {
	storageLabelSelector := "app.kubernetes.io/instance=storage"

	r.prepareK8sState(r.Opts.kubeconfigPath, storageLabelSelector, r.Opts.namespace)

	allStorageNodes := FilterStorageNodes(cluster.AllNodes)

	selectedNodes := populateWithK8sRules(allStorageNodes, spec, r.hostnameToPod)

	filteredNodes := ExcludeByCommonFields(selectedNodes, spec)

	r.logger.Debugf("Storage K8s restarter selected following nodes for restart: %v", filteredNodes)
	return selectedNodes
}
