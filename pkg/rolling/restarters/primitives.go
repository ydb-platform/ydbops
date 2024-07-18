package restarters

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
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

	if startedTime.Direction == '<' {
		return startedTime.Timestamp.After(nodeStartTime)
	}

	return startedTime.Timestamp.Before(nodeStartTime)
}

func compareMajorMinorPatch(sign string, nodeVersion, userVersion [3]int) bool {
	res := 0
	for i := 0; i < 3; i++ {
		if nodeVersion[i] < userVersion[i] {
			res = -1
			break
		} else if nodeVersion[i] > userVersion[i] {
			res = 1
			break
		}
	}

	switch sign {
	case "==":
		return res == 0
	case "<":
		return res == -1
	case ">":
		return res == 1
	case "!=":
		return res != 0
	}
	return false
}

func tryParseWith(reString, version string) (int, int, int, bool) {
	re := regexp.MustCompile(reString)
	matches := re.FindStringSubmatch(version)
	if len(matches) == 4 {
		num1, _ := strconv.Atoi(matches[1])
		num2, _ := strconv.Atoi(matches[2])
		num3, _ := strconv.Atoi(matches[3])
		return num1, num2, num3, true
	}
	return 0, 0, 0, false
}

func parseNodeVersion(version string) (int, int, int, error) {
	pattern1 := `^ydb-stable-(\d+)-(\d+)-(\d+).*$`
	major, minor, patch, parsed := tryParseWith(pattern1, version)
	if parsed {
		return major, minor, patch, nil
	}

	pattern2 := `^(\d+)\.(\d+)\.(\d+).*$`
	major, minor, patch, parsed = tryParseWith(pattern2, version)
	if parsed {
		return major, minor, patch, nil
	}

	return 0, 0, 0, fmt.Errorf("failed to parse the version number in any of the known patterns")
}

func SatisfiedVersion(node *Ydb_Maintenance.Node, version *options.VersionSpec) (bool, error) {
	if version == nil {
		return true, nil
	}

	major, minor, patch, err := parseNodeVersion(node.Version)
	if err != nil {
		return false, fmt.Errorf("Failed to extract major.minor.patch from version %s", node.Version)
	}

	return compareMajorMinorPatch(
		version.Sign,
		[3]int{major, minor, patch},
		[3]int{version.Major, version.Minor, version.Patch},
	), nil
}

func isInclusiveFilteringUnspecified(spec FilterNodeParams) bool {
	return len(spec.SelectedHosts) == 0 && len(spec.SelectedNodeIds) == 0
}

func includeByHostIDOrFQDN(nodes []*Ydb_Maintenance.Node, spec FilterNodeParams) []*Ydb_Maintenance.Node {
	selected := []*Ydb_Maintenance.Node{}

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

	return includeByHostIDOrFQDN(nodes, spec)
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

		satisfiesVersion, err := SatisfiedVersion(node, spec.Version)
		if err != nil {
			unknownVersions[node.Version] = append(unknownVersions[node.Version], node.Host)
		}

		if !satisfiesVersion {
			continue
		}

		filtered = append(filtered, node)
	}

	prettyUnknownVersions, _ := json.MarshalIndent(unknownVersions, "", "  ")
	zap.S().Warnf(`Failed to extract major.minor.patch when filtering by %s from some nodes.
Here is a map from node version to node FQDNs with this version: 
%s`,
		spec.Version.String(),
		prettyUnknownVersions,
	)

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
