package rolling

import (
	"reflect"
	"testing"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

func testTenantNode(nodeID uint32, tenant string) *Ydb_Maintenance.Node {
	return &Ydb_Maintenance.Node{
		NodeId: nodeID,
		Type: &Ydb_Maintenance.Node_Dynamic{
			Dynamic: &Ydb_Maintenance.Node_DynamicNode{
				Tenant: tenant,
			},
		},
	}
}

func nodeIDs(nodes []*Ydb_Maintenance.Node) []uint32 {
	ids := make([]uint32, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.GetNodeId())
	}
	return ids
}

func TestOrderTenantNodesRoundRobin(t *testing.T) {
	nodes := []*Ydb_Maintenance.Node{
		testTenantNode(1, "tenant-a"),
		testTenantNode(2, "tenant-a"),
		testTenantNode(3, "tenant-a"),
		testTenantNode(10, "tenant-b"),
		testTenantNode(11, "tenant-b"),
		testTenantNode(20, "tenant-c"),
	}

	ordered := orderTenantNodesRoundRobin(nodes)
	expected := []uint32{1, 10, 20, 2, 11, 3}

	if !reflect.DeepEqual(nodeIDs(ordered), expected) {
		t.Fatalf("unexpected tenant round-robin order: got %v, want %v", nodeIDs(ordered), expected)
	}
}

func TestOrderNodesForRestartUsesTenantRoundRobinWithTenantsInflight(t *testing.T) {
	rolling := &Rolling{
		logger: zap.NewNop().Sugar(),
		opts: &RestartOptions{
			TenantsInflight: 10,
		},
	}
	nodes := []*Ydb_Maintenance.Node{
		testTenantNode(1, "tenant-a"),
		testTenantNode(2, "tenant-a"),
		testTenantNode(3, "tenant-a"),
		testTenantNode(10, "tenant-b"),
		testTenantNode(11, "tenant-b"),
		testTenantNode(20, "tenant-c"),
	}

	ordered := rolling.orderNodesForRestart(nodes)
	expected := []uint32{1, 10, 20, 2, 11, 3}

	if !reflect.DeepEqual(nodeIDs(ordered), expected) {
		t.Fatalf("unexpected restart node order: got %v, want %v", nodeIDs(ordered), expected)
	}
}
