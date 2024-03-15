package restarters

import (
	"io"
	"strconv"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydbops/internal/collections"
	"github.com/ydb-platform/ydbops/pkg/options"
	"go.uber.org/zap"
)

func FilterStorageNodes(nodes []*Ydb_Maintenance.Node) []*Ydb_Maintenance.Node {
	return collections.FilterBy(nodes,
		func(node *Ydb_Maintenance.Node) bool {
			return node.GetStorage() != nil
		},
	)
}

func FilterTenantNodes(nodes []*Ydb_Maintenance.Node) []*Ydb_Maintenance.Node {
	return collections.FilterBy(nodes,
		func(node *Ydb_Maintenance.Node) bool {
			return node.GetDynamic() != nil
		},
	)
}

func FilterByNodeIds(nodes []*Ydb_Maintenance.Node, nodeIds []uint32) []*Ydb_Maintenance.Node {
	return collections.FilterBy(nodes,
		func(node *Ydb_Maintenance.Node) bool {
			return collections.Contains(nodeIds, node.NodeId)
		},
	)
}

func FilterByHostFQDN(nodes []*Ydb_Maintenance.Node, hostFQDNs []string) []*Ydb_Maintenance.Node {
	return collections.FilterBy(nodes,
		func(node *Ydb_Maintenance.Node) bool {
			return collections.Contains(hostFQDNs, node.Host)
		},
	)
}

func StreamPipeIntoLogger(p io.ReadCloser, logger *zap.SugaredLogger) {
	buf := make([]byte, 1024)
	for {
		n, err := p.Read(buf)
		if n > 0 {
			logger.Info(string(buf[:n]))
		}
		if err != nil {
			if err != io.EOF {
				logger.Error("Error reading from pipe", zap.Error(err))
			}
			break
		}
	}
}

func SatisfiesStartingTime(node *Ydb_Maintenance.Node, startedTime *options.StartedTime) bool {
	if startedTime == nil {
		return true
	}

	nodeStartTime := node.GetStartTime().AsTime()

	if startedTime.Direction == '<' {
		return startedTime.Timestamp.After(nodeStartTime)
	} else {
		return startedTime.Timestamp.Before(nodeStartTime)
	}
}

func isInclusiveFilteringUnspecified(spec FilterNodeParams) bool {
	return len(spec.SelectedHostFQDNs) == 0 && len(spec.SelectedNodeIds) == 0
}

func includeByHostIdOrFQDN(nodes []*Ydb_Maintenance.Node, spec FilterNodeParams) []*Ydb_Maintenance.Node {
	selected := []*Ydb_Maintenance.Node{}

	selected = append(
		selected, FilterByHostFQDN(nodes, spec.SelectedHostFQDNs)...,
	)

	selected = append(
		selected, FilterByNodeIds(nodes, spec.SelectedNodeIds)...,
	)

	selected = MergeAndUnique(selected)

	return selected
}

func PopulateByCommonFields(nodes []*Ydb_Maintenance.Node, spec FilterNodeParams) []*Ydb_Maintenance.Node {
	if isInclusiveFilteringUnspecified(spec) {
		return nodes
	} else {
		return includeByHostIdOrFQDN(nodes, spec)
	}
}

func ExcludeByCommonFields(nodes []*Ydb_Maintenance.Node, spec FilterNodeParams) []*Ydb_Maintenance.Node {
	filtered := []*Ydb_Maintenance.Node{}
	for _, node := range nodes {
		if collections.Contains(spec.ExcludeHosts, strconv.Itoa(int(node.NodeId))) {
			continue
		}

		if collections.Contains(spec.ExcludeHosts, node.Host) {
			continue
		}

		if !SatisfiesStartingTime(node, spec.StartedTime) {
			continue
		}

		filtered = append(filtered, node)
	}
	return filtered
}

func PopulateByTenantNames(
	tenantNodes []*Ydb_Maintenance.Node,
	selectedTenants []string,
	tenantToNodeIds map[string][]uint32,
) []*Ydb_Maintenance.Node {
	return collections.FilterBy(tenantNodes, func(node *Ydb_Maintenance.Node) bool {
		for _, tenant := range selectedTenants {
			if collections.Contains(tenantToNodeIds[tenant], node.NodeId) {
				return true
			}
		}
		return false
	})
}

func MergeAndUnique(nodeSlices ...[]*Ydb_Maintenance.Node) []*Ydb_Maintenance.Node {
	presentNodes := make(map[uint32]bool)
	result := []*Ydb_Maintenance.Node{}
	for _, arg := range nodeSlices {
		for _, node := range arg {
			if _, present := presentNodes[node.NodeId]; !present {
				result = append(result, node)
				presentNodes[node.NodeId] = true
			}
		}
	}

	return result
}
