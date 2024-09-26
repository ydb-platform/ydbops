package restarters

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/tests/mock"
)

var _ = Describe("Test storage k8s Filter", func() {
	var (
		now                     = time.Now()
		tenMinutesAgoTimestamp  = now.Add(-10 * time.Minute)
		fiveMinutesAgoTimestamp = now.Add(-5 * time.Minute)
	)

	It("k8s restarter filtering by --started>timestamp", func() {
		filterSpec := FilterNodeParams{
			MaxStaticNodeId: DefaultMaxStaticNodeId,
			StartedTime: &options.StartedTime{
				Direction: '<',
				Timestamp: fiveMinutesAgoTimestamp,
			},
		}

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

		clusterInfo := ClusterNodesInfo{
			AllNodes:        nodes,
			TenantToNodeIds: map[string][]uint32{},
		}

		filteredNodes := applyStorageK8sFilteringRules(filterSpec, clusterInfo, map[string]string{})

		Expect(len(filteredNodes)).To(Equal(3))

		filteredNodeIds := make(map[uint32]bool)
		for _, node := range filteredNodes {
			filteredNodeIds[node.NodeId] = true
		}

		Expect(filteredNodeIds).Should(HaveKey(uint32(1)))
		Expect(filteredNodeIds).Should(HaveKey(uint32(2)))
		Expect(filteredNodeIds).Should(HaveKey(uint32(3)))
	})

	It("k8s restarter filtering by --dc", func() {
		firstDCName := "ru-central1-a"
		secondDCName := "ru-central1-b"
		filterSpec := FilterNodeParams{
			MaxStaticNodeId:     DefaultMaxStaticNodeId,
			SelectedDatacenters: []string{secondDCName},
		}

		nodeGroups := [][]uint32{
			{1, 2, 3, 4, 5, 6, 7, 8},
		}
		nodeInfoMap := map[uint32]mock.TestNodeInfo{
			1: {
				Datacenter: firstDCName,
			},
			2: {
				Datacenter: firstDCName,
			},
			3: {
				Datacenter: firstDCName,
			},
			4: {
				Datacenter: secondDCName,
			},
			5: {
				Datacenter: secondDCName,
			},
			6: {
				Datacenter: secondDCName,
			},
			7: {
				Datacenter: secondDCName,
			},
			8: {
				Datacenter: secondDCName,
			},
		}

		nodes := mock.CreateNodesFromShortConfig(nodeGroups, nodeInfoMap)

		clusterInfo := ClusterNodesInfo{
			AllNodes:        nodes,
			TenantToNodeIds: map[string][]uint32{},
		}

		filteredNodes := applyStorageK8sFilteringRules(filterSpec, clusterInfo, map[string]string{})

		Expect(len(filteredNodes)).To(Equal(5))

		filteredNodeIds := make(map[uint32]bool)
		for _, node := range filteredNodes {
			filteredNodeIds[node.NodeId] = true
		}

		Expect(filteredNodeIds).Should(HaveKey(uint32(4)))
		Expect(filteredNodeIds).Should(HaveKey(uint32(5)))
		Expect(filteredNodeIds).Should(HaveKey(uint32(6)))
		Expect(filteredNodeIds).Should(HaveKey(uint32(7)))
		Expect(filteredNodeIds).Should(HaveKey(uint32(8)))
	})
})
