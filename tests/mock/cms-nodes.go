package mock

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
)

func makePointer[T any](arg T) *T {
	return &arg
}

func makeLocation(nodeId uint32) *Ydb_Discovery.NodeLocation {
	return &Ydb_Discovery.NodeLocation{
		DataCenter: makePointer("DC-1"),
		Module:     makePointer("DC-1-MODULE-1"),
		Rack:       makePointer("DC-1-MODULE-1-RACK-1"),
		Unit:       makePointer(fmt.Sprintf("DC-1-MODULE-1-RACK-1-UNIT-%v", nodeId)),
	}
}

func makeNode(nodeId uint32) *Ydb_Maintenance.Node {
	return &Ydb_Maintenance.Node{
		NodeId:   nodeId,
		Host:     fmt.Sprintf("storage-%v.ydb.tech", nodeId),
		Port:     19000,
		Location: makeLocation(nodeId),
		State:    Ydb_Maintenance.ItemState_ITEM_STATE_UP,
		Type: &Ydb_Maintenance.Node_Storage{
			Storage: &Ydb_Maintenance.Node_StorageNode{},
		},
	}
}

func (s *YdbMock) SetNodeConfiguration(groupDistribution [][]uint32) {
  s.isNodeCurrentlyPermitted = make(map[uint32]bool)
	s.nodeGroups = groupDistribution

	for _, group := range s.nodeGroups {
		for _, nodeId := range group {
			s.isNodeCurrentlyPermitted[nodeId] = false
			s.nodes = append(s.nodes, makeNode(nodeId))
		}
	}
}
