package mock

import (
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
)

func makePointer[T any](arg T) *T {
	return &arg
}

func makeLocation(nodeId int) *Ydb_Discovery.NodeLocation {
	return &Ydb_Discovery.NodeLocation{
		DataCenter: makePointer("DC-1"),
		Module:     makePointer("DC-1-MODULE-1"),
		Rack:       makePointer("DC-1-MODULE-1-RACK-1"),
		Unit:       makePointer(fmt.Sprintf("DC-1-MODULE-1-RACK-1-UNIT-%v", nodeId)),
	}
}

func makeNode(nodeId int) *Ydb_Maintenance.Node {
	return &Ydb_Maintenance.Node{
		NodeId:   uint32(nodeId),
		Host:     fmt.Sprintf("storage-%v.ydb.tech", nodeId),
		Port:     19000,
		Location: makeLocation(nodeId),
		State:    Ydb_Maintenance.ItemState_ITEM_STATE_UP,
		Type: &Ydb_Maintenance.Node_Storage{
			Storage: &Ydb_Maintenance.Node_StorageNode{},
		},
	}
}

func (s *YdbMock) initNodes() {
	s.nodes = []*Ydb_Maintenance.Node{}
	for i := 0; i < 8; i++ {
		s.nodes = append(s.nodes, makeNode(i))
	}

	s.nodeGroups = [][]uint32{{1, 2, 3, 4, 5, 6, 7, 8}}

  s.isNodeCurrentlyPermitted = make(map[uint32]bool)
	for i := 0; i < 8; i++ {
		s.isNodeCurrentlyPermitted[uint32(i)] = false
	}
}
