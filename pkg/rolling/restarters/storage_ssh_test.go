package restarters

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/tests/mock"
)

var _ = Describe("Test storage ssh Filter", func() {
	var (
		now                     = time.Now()
		tenMinutesAgoTimestamp  = now.Add(-10 * time.Minute)
		fiveMinutesAgoTimestamp = now.Add(-5 * time.Minute)
	)

	It("ssh restarter filtering by --started>timestamp", func() {
		restarter := NewStorageSSHRestarter(zap.S(), []string{}, "")

		nodeGroups := [][]uint32{
			{1, 2, 3, 4, 5, 6, 7, 8},
		}
		nodeInfoMap := map[uint32]mock.TestNodeInfo{
			1: {
				StartTime: tenMinutesAgoTimestamp,
			},
			2: {
				StartTime: tenMinutesAgoTimestamp,
			},
			3: {
				StartTime: tenMinutesAgoTimestamp,
			},
		}

		nodes := mock.CreateNodesFromShortConfig(nodeGroups, nodeInfoMap)

		filterSpec := FilterNodeParams{
			MaxStaticNodeID: DefaultMaxStaticNodeID,
			StartedTime: &options.StartedTime{
				Direction: '<',
				Timestamp: fiveMinutesAgoTimestamp,
			},
		}

		clusterInfo := ClusterNodesInfo{
			AllNodes:        nodes,
			TenantToNodeIds: map[string][]uint32{},
		}

		filteredNodes := restarter.Filter(filterSpec, clusterInfo)

		Expect(len(filteredNodes)).To(Equal(3))

		filteredNodeIds := make(map[uint32]bool)
		for _, node := range filteredNodes {
			filteredNodeIds[node.NodeId] = true
		}

		Expect(filteredNodeIds).Should(HaveKey(uint32(1)))
		Expect(filteredNodeIds).Should(HaveKey(uint32(2)))
		Expect(filteredNodeIds).Should(HaveKey(uint32(3)))
	})

	It("storage restarter without arguments takes all storage nodes and no dynnodes", func() {
		restarter := NewStorageSSHRestarter(zap.S(), []string{}, "")

		nodeGroups := [][]uint32{
			{1, 2, 3, 4, 5, 6, 7, 8},
			{9, 10, 11},
		}
		nodeInfoMap := map[uint32]mock.TestNodeInfo{
			9: {
				IsDynnode:  true,
				TenantName: "fakeTenant",
			},
			10: {
				IsDynnode:  true,
				TenantName: "fakeTenant",
			},
			11: {
				IsDynnode:  true,
				TenantName: "fakeTenant",
			},
		}

		nodes := mock.CreateNodesFromShortConfig(nodeGroups, nodeInfoMap)

		filterSpec := FilterNodeParams{
			MaxStaticNodeID: DefaultMaxStaticNodeID,
		}

		clusterInfo := ClusterNodesInfo{
			AllNodes: nodes,
			TenantToNodeIds: map[string][]uint32{
				"fakeTenant": {9, 10, 11},
			},
		}

		filteredNodes := restarter.Filter(filterSpec, clusterInfo)

		Expect(len(filteredNodes)).To(Equal(8))

		filteredNodeIds := make(map[uint32]bool)
		for _, node := range filteredNodes {
			filteredNodeIds[node.NodeId] = true
		}

		for i := 1; i <= 8; i++ {
			Expect(filteredNodeIds).Should(HaveKey(uint32(i)))
		}
	})
})
