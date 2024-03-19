package mock

import (
	"fmt"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		Host:     fmt.Sprintf("ydb-%v.ydb.tech", nodeId),
		Port:     19000,
		Location: makeLocation(nodeId),
		State:    Ydb_Maintenance.ItemState_ITEM_STATE_UP,
		Type: &Ydb_Maintenance.Node_Storage{
			Storage: &Ydb_Maintenance.Node_StorageNode{},
		},
	}
}

func MakeActionGroups(nodeIds ...uint32) []*Ydb_Maintenance.ActionGroup {
	result := []*Ydb_Maintenance.ActionGroup{}
	for _, nodeId := range nodeIds {
		result = append(result,
			&Ydb_Maintenance.ActionGroup{
				Actions: []*Ydb_Maintenance.Action{
					{
						Action: &Ydb_Maintenance.Action_LockAction{
							LockAction: &Ydb_Maintenance.LockAction{
								Scope: &Ydb_Maintenance.ActionScope{
									Scope: &Ydb_Maintenance.ActionScope_NodeId{
										NodeId: nodeId,
									},
								},
								Duration: durationpb.New(time.Duration(180) * time.Second),
							},
						},
					},
				},
			},
		)
	}
	return result
}

type TestNodeInfo struct {
	StartTime  time.Time
	IsDynnode  bool
	TenantName string
	Version    string
}

func CreateNodesFromShortConfig(nodeGroups [][]uint32, nodeInfo map[uint32]TestNodeInfo) []*Ydb_Maintenance.Node {
	nodes := []*Ydb_Maintenance.Node{}
	for _, group := range nodeGroups {
		for _, nodeId := range group {
			testNodeInfo, ok := nodeInfo[nodeId]
			node := makeNode(nodeId)

			if ok && testNodeInfo.IsDynnode {
				node.Type = &Ydb_Maintenance.Node_Dynamic{
					Dynamic: &Ydb_Maintenance.Node_DynamicNode{
						Tenant: testNodeInfo.TenantName,
					},
				}
			} else {
				node.Type = &Ydb_Maintenance.Node_Storage{
					Storage: &Ydb_Maintenance.Node_StorageNode{},
				}
			}

			if ok && !testNodeInfo.StartTime.IsZero() {
				node.StartTime = timestamppb.New(testNodeInfo.StartTime)
			} else {
				node.StartTime = timestamppb.New(time.Now())
			}

			if ok && len(testNodeInfo.Version) > 0 {
				node.Version = testNodeInfo.Version
			} else {
				node.Version = "some-fake-version-will-fail-when-parsing"
			}

			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (s *YdbMock) SetNodeConfiguration(nodeGroups [][]uint32, nodeInfo map[uint32]TestNodeInfo) {
	s.isNodeCurrentlyPermitted = make(map[uint32]bool)
	s.nodeGroups = nodeGroups

	for _, group := range s.nodeGroups {
		for _, nodeId := range group {
			s.isNodeCurrentlyPermitted[nodeId] = false
		}
	}

	s.nodes = CreateNodesFromShortConfig(nodeGroups, nodeInfo)
}
