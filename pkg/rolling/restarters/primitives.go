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

func FilterByStartedTime(nodes []*Ydb_Maintenance.Node, startedTime options.StartedOptions) []*Ydb_Maintenance.Node {
	return collections.FilterBy(nodes,
		func(node *Ydb_Maintenance.Node) bool {
			nodeStartTime := node.GetStartTime().AsTime()
			if startedTime.Direction == '>' {
				return startedTime.Timestamp.Before(nodeStartTime)
			} else {
				return startedTime.Timestamp.After(nodeStartTime)
			}
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

func FilterByNodeIdOrFQDN(nodes []*Ydb_Maintenance.Node, spec FilterNodeParams) []*Ydb_Maintenance.Node {
	preSelected := []*Ydb_Maintenance.Node{}

	preSelected = append(
		preSelected,
		FilterByNodeIds(nodes, spec.SelectedNodeIds)...,
	)

	preSelected = append(
		preSelected, FilterByHostFQDN(nodes, spec.SelectedHostFQDNs)...,
	)

	selected := []*Ydb_Maintenance.Node{}
	for _, node := range preSelected {
		if collections.Contains(spec.ExcludeHosts, strconv.Itoa(int(node.NodeId))) {
			continue
		}

		if collections.Contains(spec.ExcludeHosts, node.Host) {
			continue
		}
		selected = append(selected, node)
	}

	return selected
}
