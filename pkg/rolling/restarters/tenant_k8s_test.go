package restarters

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/tests/mock"
)

var _ = Describe("Test tenant k8s Filter", func() {
	var (
		now                     = time.Now()
		tenMinutesAgoTimestamp  = now.Add(-10 * time.Minute)
		fiveMinutesAgoTimestamp = now.Add(-5 * time.Minute)
	)

	It("k8s restarter filtering by --started>timestamp", func() {
		filterSpec := FilterNodeParams{
			MaxStaticNodeID: DefaultMaxStaticNodeID,
			StartedTime: &options.StartedTime{
				Direction: '<',
				Timestamp: fiveMinutesAgoTimestamp,
			},
		}

		nodeGroups := [][]uint32{
			{1, 2, 3, 4, 5, 6, 7, 8},
			{9, 10, 11},
			{12, 13, 14},
		}

		tenant1Name := "tenant1"
		tenant2Name := "tenant2"

		nodeInfoMap := map[uint32]mock.TestNodeInfo{
			1: {
				StartTime: tenMinutesAgoTimestamp,
			},
			9: {
				StartTime:  tenMinutesAgoTimestamp,
				IsDynnode:  true,
				TenantName: tenant1Name,
			},
			10: {
				IsDynnode:  true,
				TenantName: tenant1Name,
			},
			11: {
				IsDynnode:  true,
				TenantName: tenant1Name,
			},
			12: {
				StartTime:  tenMinutesAgoTimestamp,
				IsDynnode:  true,
				TenantName: tenant2Name,
			},
			13: {
				IsDynnode:  true,
				TenantName: tenant2Name,
			},
			14: {
				StartTime:  tenMinutesAgoTimestamp,
				IsDynnode:  true,
				TenantName: tenant2Name,
			},
		}

		nodes := mock.CreateNodesFromShortConfig(nodeGroups, nodeInfoMap)

		clusterInfo := ClusterNodesInfo{
			AllNodes: nodes,
			TenantToNodeIds: map[string][]uint32{
				tenant1Name: {9, 10, 11},
				tenant2Name: {12, 13, 14},
			},
		}

		filteredNodes := applyTenantK8sFilteringRules(filterSpec, clusterInfo, map[string]string{})

		Expect(len(filteredNodes)).To(Equal(3))

		filteredNodeIds := make(map[uint32]bool)
		for _, node := range filteredNodes {
			filteredNodeIds[node.NodeId] = true
		}

		Expect(filteredNodeIds).Should(HaveKey(uint32(9)))
		Expect(filteredNodeIds).Should(HaveKey(uint32(12)))
		Expect(filteredNodeIds).Should(HaveKey(uint32(14)))
	})
})
