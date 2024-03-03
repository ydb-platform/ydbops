package restarters

import (
	"io"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-ops/internal/util"
	"go.uber.org/zap"
)

func FilterStorageNodes(nodes []*Ydb_Maintenance.Node) []*Ydb_Maintenance.Node {
	return util.FilterBy(nodes,
		func(node *Ydb_Maintenance.Node) bool {
			return node.GetStorage() != nil
		},
	)
}

func FilterDynnodes(nodes []*Ydb_Maintenance.Node) []*Ydb_Maintenance.Node {
	return util.FilterBy(nodes,
		func(node *Ydb_Maintenance.Node) bool {
			return node.GetDynamic() != nil
		},
	)
}

func FilterByNodeIds(nodes []*Ydb_Maintenance.Node, nodeIds []uint32) []*Ydb_Maintenance.Node {
	return util.FilterBy(nodes,
		func(node *Ydb_Maintenance.Node) bool {
			return util.Contains(nodeIds, node.NodeId)
		},
	)
}

func FilterByHostFQDN(nodes []*Ydb_Maintenance.Node, hostFQDNs []string) []*Ydb_Maintenance.Node {
	return util.FilterBy(nodes,
		func(node *Ydb_Maintenance.Node) bool {
			return util.Contains(hostFQDNs, node.Host)
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

