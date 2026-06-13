package rolling

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

func TestRolling(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rolling Suite")
}

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

var _ = Describe("Tenant restart node ordering", func() {
	It("round-robins tenant nodes after sorting by node id", func() {
		nodes := []*Ydb_Maintenance.Node{
			testTenantNode(1, "tenant-a"),
			testTenantNode(2, "tenant-a"),
			testTenantNode(3, "tenant-a"),
			testTenantNode(10, "tenant-b"),
			testTenantNode(11, "tenant-b"),
			testTenantNode(20, "tenant-c"),
		}

		ordered := orderTenantNodesRoundRobin(nodes)

		Expect(nodeIDs(ordered)).To(Equal([]uint32{1, 10, 20, 2, 11, 3}))
	})

	It("uses tenant round-robin for restart ordering when tenants-inflight is set", func() {
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

		Expect(nodeIDs(ordered)).To(Equal([]uint32{1, 10, 20, 2, 11, 3}))
	})
})
