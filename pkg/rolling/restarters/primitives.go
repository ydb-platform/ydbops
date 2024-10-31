package restarters

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/internal/collections"
	"github.com/ydb-platform/ydbops/pkg/options"
)

const (
	DefaultMaxStaticNodeId = 50000
)

func FilterStorageNodes(nodes []*Ydb_Maintenance.Node, maxStaticNodeId uint32) []*Ydb_Maintenance.Node {
	return collections.FilterBy(nodes,
		func(node *Ydb_Maintenance.Node) bool {
			return node.GetStorage() != nil && node.NodeId < maxStaticNodeId
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

func FilterByDatacenters(nodes []*Ydb_Maintenance.Node, datacenters []string) []*Ydb_Maintenance.Node {
	return collections.FilterBy(nodes,
		func(node *Ydb_Maintenance.Node) bool {
			return collections.Contains(datacenters, node.GetLocation().GetDataCenter())
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
			if errors.Is(err, io.EOF) {
				logger.Info("Finished reading from remote command pipe")
			} else {
				logger.Errorf("Unknown error while reading from remote command pipe: %w", err)
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

	if nodeStartTime.IsZero() {
		zap.S().Warnf(
			"Node %s did not have startTime specified by CMS (possibly an old YDB version). Avoid using --started filter on current YDB cluster",
			node.Host,
		)
		return false
	}

	if startedTime.Direction == '<' {
		return startedTime.Timestamp.After(nodeStartTime)
	}

	return startedTime.Timestamp.Before(nodeStartTime)
}

func isInclusiveFilteringUnspecified(spec FilterNodeParams) bool {
	return len(spec.SelectedDatacenters) == 0 && len(spec.SelectedHosts) == 0 && len(spec.SelectedNodeIds) == 0
}

func includeByFilterNodeParams(nodes []*Ydb_Maintenance.Node, spec FilterNodeParams) []*Ydb_Maintenance.Node {
	selected := []*Ydb_Maintenance.Node{}

	selected = append(
		selected, FilterByDatacenters(nodes, spec.SelectedDatacenters)...,
	)

	selected = append(
		selected, FilterByHostFQDN(nodes, spec.SelectedHosts)...,
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
	}

	return includeByFilterNodeParams(nodes, spec)
}

func ExcludeByCommonFields(nodes []*Ydb_Maintenance.Node, spec FilterNodeParams) []*Ydb_Maintenance.Node {
	filtered := []*Ydb_Maintenance.Node{}

	unknownVersions := make(map[string][]string)
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

		if spec.Version != nil {
			if node.Version == "" {
				zap.S().Warnf(
					"Node %s did not have version field specified by CMS (possibly an old YDB version). Avoid using --version filter on current YDB cluster",
					node.Host,
				)
				continue
			}

			satisfiesVersion, err := spec.Version.Satisfies(node.Version)
			if err != nil {
				unknownVersions[node.Version] = append(unknownVersions[node.Version], node.Host)
			}

			if !satisfiesVersion {
				continue
			}
		}

		filtered = append(filtered, node)
	}

	if spec.Version != nil && len(unknownVersions) > 0 {
		prettyUnknownVersions, _ := json.MarshalIndent(unknownVersions, "", "  ")
		zap.S().Warnf(`Failed to extract major.minor.patch when filtering by %s from some nodes.
Here is a map from node version to node FQDNs with this version: 
%s`,
			spec.Version.String(),
			prettyUnknownVersions,
		)
	}

	return filtered
}

func ExcludeByTenantNames(
	tenantNodes []*Ydb_Maintenance.Node,
	selectedTenants []string,
	tenantToNodeIds map[string][]uint32,
) []*Ydb_Maintenance.Node {
	if len(selectedTenants) == 0 {
		return tenantNodes
	}

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
