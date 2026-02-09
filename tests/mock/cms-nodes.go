package mock

import (
	"fmt"
	"math"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func makePointer[T any](arg T) *T {
	return &arg
}

func makeLocation(nodeID uint32) *Ydb_Discovery.NodeLocation {
	return &Ydb_Discovery.NodeLocation{
		DataCenter: makePointer("DC-1"),
		Module:     makePointer("DC-1-MODULE-1"),
		Rack:       makePointer("DC-1-MODULE-1-RACK-1"),
		Unit:       makePointer(fmt.Sprintf("DC-1-MODULE-1-RACK-1-UNIT-%v", nodeID)),
	}
}

func makeNode(nodeID uint32) *Ydb_Maintenance.Node {
	return &Ydb_Maintenance.Node{
		NodeId:   nodeID,
		Host:     fmt.Sprintf("ydb-%v.ydb.tech", nodeID),
		Port:     19000,
		Location: makeLocation(nodeID),
		State:    Ydb_Maintenance.ItemState_ITEM_STATE_UP,
		Type: &Ydb_Maintenance.Node_Storage{
			Storage: &Ydb_Maintenance.Node_StorageNode{},
		},
	}
}

func determineRestartDuration(nNodes, nodesInflight int) time.Duration {
	defaultRestartDuration := time.Second * 60
	singleBatchRestartTime := defaultRestartDuration * 3

	singleBatchWithWait := singleBatchRestartTime + 1*time.Second
	maximumTotalBatches := int(math.Ceil(float64(nNodes) / float64(nodesInflight)))

	return time.Duration(maximumTotalBatches) * singleBatchWithWait
}

func MakeActionGroupsFromNodesIdsFixedDuration(duration time.Duration, nodeIDs ...uint32) []*Ydb_Maintenance.ActionGroup {
	result := make([]*Ydb_Maintenance.ActionGroup, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		result = append(result,
			&Ydb_Maintenance.ActionGroup{
				Actions: []*Ydb_Maintenance.Action{
					{
						Action: &Ydb_Maintenance.Action_LockAction{
							LockAction: &Ydb_Maintenance.LockAction{
								Scope: &Ydb_Maintenance.ActionScope{
									Scope: &Ydb_Maintenance.ActionScope_NodeId{
										NodeId: nodeID,
									},
								},
								Duration: durationpb.New(duration),
							},
						},
					},
				},
			},
		)
	}
	return result
}

func MakeActionGroupsFromNodeIds(nodeIDs ...uint32) []*Ydb_Maintenance.ActionGroup {
	return MakeActionGroupsFromNodeIdsWithInflight(1, nodeIDs...)
}

func MakeActionGroupsFromNodeIdsWithInflight(nodesInflight int, nodeIDs ...uint32) []*Ydb_Maintenance.ActionGroup {
	result := make([]*Ydb_Maintenance.ActionGroup, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		result = append(result,
			&Ydb_Maintenance.ActionGroup{
				Actions: []*Ydb_Maintenance.Action{
					{
						Action: &Ydb_Maintenance.Action_LockAction{
							LockAction: &Ydb_Maintenance.LockAction{
								Scope: &Ydb_Maintenance.ActionScope{
									Scope: &Ydb_Maintenance.ActionScope_NodeId{
										NodeId: nodeID,
									},
								},
								Duration: durationpb.New(determineRestartDuration(len(nodeIDs), nodesInflight)),
							},
						},
					},
				},
			},
		)
	}
	return result
}

func MakeActionGroupsFromHostFQDNsFixedDuration(duration time.Duration, hostFQDNs ...string) []*Ydb_Maintenance.ActionGroup {
	result := make([]*Ydb_Maintenance.ActionGroup, 0, len(hostFQDNs))
	for _, hostFQDN := range hostFQDNs {
		result = append(result,
			&Ydb_Maintenance.ActionGroup{
				Actions: []*Ydb_Maintenance.Action{
					{
						Action: &Ydb_Maintenance.Action_LockAction{
							LockAction: &Ydb_Maintenance.LockAction{
								Scope: &Ydb_Maintenance.ActionScope{
									Scope: &Ydb_Maintenance.ActionScope_Host{
										Host: hostFQDN,
									},
								},
								Duration: durationpb.New(duration),
							},
						},
					},
				},
			},
		)
	}
	return result
}

func MakeActionGroupsFromHostFQDNs(hostFQDNs ...string) []*Ydb_Maintenance.ActionGroup {
	result := make([]*Ydb_Maintenance.ActionGroup, 0, len(hostFQDNs))
	for _, hostFQDN := range hostFQDNs {
		result = append(result,
			&Ydb_Maintenance.ActionGroup{
				Actions: []*Ydb_Maintenance.Action{
					{
						Action: &Ydb_Maintenance.Action_LockAction{
							LockAction: &Ydb_Maintenance.LockAction{
								Scope: &Ydb_Maintenance.ActionScope{
									Scope: &Ydb_Maintenance.ActionScope_Host{
										Host: hostFQDN,
									},
								},
								Duration: durationpb.New(determineRestartDuration(len(hostFQDNs), 1)),
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
	Datacenter string
	State      Ydb_Maintenance.ItemState
}

func CreateNodesFromShortConfig(nodeGroups [][]uint32, nodeInfo map[uint32]TestNodeInfo) []*Ydb_Maintenance.Node {
	nodes := []*Ydb_Maintenance.Node{}
	for _, group := range nodeGroups {
		for _, nodeID := range group {
			testNodeInfo, moreInfoPresent := nodeInfo[nodeID]
			node := makeNode(nodeID)

			// Put default values:
			node.Type = &Ydb_Maintenance.Node_Storage{
				Storage: &Ydb_Maintenance.Node_StorageNode{},
			}
			node.State = Ydb_Maintenance.ItemState_ITEM_STATE_UP
			node.StartTime = timestamppb.New(time.Now())
			node.Version = "some-fake-version-will-fail-when-parsing"

			if !moreInfoPresent {
				nodes = append(nodes, node)
				continue
			}

			if testNodeInfo.IsDynnode {
				node.Type = &Ydb_Maintenance.Node_Dynamic{
					Dynamic: &Ydb_Maintenance.Node_DynamicNode{
						Tenant: testNodeInfo.TenantName,
					},
				}
			}

			if !testNodeInfo.StartTime.IsZero() {
				node.StartTime = timestamppb.New(testNodeInfo.StartTime)
			}

			if len(testNodeInfo.Version) > 0 {
				node.Version = testNodeInfo.Version
			}

			if testNodeInfo.State == Ydb_Maintenance.ItemState_ITEM_STATE_MAINTENANCE ||
				testNodeInfo.State == Ydb_Maintenance.ItemState_ITEM_STATE_DOWN {
				node.State = testNodeInfo.State
			}

			if len(testNodeInfo.Datacenter) > 0 {
				datacenter := &testNodeInfo.Datacenter
				node.Location.DataCenter = datacenter
			}

			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (s *YdbMock) SetNodeConfiguration(nodeGroups [][]uint32, nodeInfo map[uint32]TestNodeInfo) {
	s.isNodeCurrentlyReleased = make(map[uint32]bool)
	s.nodeGroups = nodeGroups

	for _, group := range s.nodeGroups {
		for _, nodeID := range group {
			s.isNodeCurrentlyReleased[nodeID] = false
		}
	}

	s.nodes = CreateNodesFromShortConfig(nodeGroups, nodeInfo)
}
