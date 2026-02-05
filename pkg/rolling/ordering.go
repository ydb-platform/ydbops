package rolling

import (
	"math/rand/v2"
	"slices"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydbops/internal/collections"
)

const (
	RandomOrderingKey  = "random"
	ClusterOrderingKey = "cluster"
	TenantOrderingKey  = "tenant"
)

var OrderingKeyChoices = []string{
	RandomOrderingKey,
	ClusterOrderingKey,
	TenantOrderingKey,
}

func reOrderNodesByOrderingKey(orderingKey string, nodes []*Ydb_Maintenance.Node) []*Ydb_Maintenance.Node {
	// Avoids Re-ordering nodes if a storage node exists in the list.
	// Ordering logic is specifically for a list of nodes that just consist of tenant nodes.
	if slices.ContainsFunc(nodes, func(node *Ydb_Maintenance.Node) bool {
		return node.GetDynamic() == nil
	}) {
		return nodes
	}

	switch orderingKey {
	case ClusterOrderingKey:
		return nodes
	case RandomOrderingKey:
		rand.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
		return nodes
	case TenantOrderingKey:
		return reOrderNodesByTenantOrderingKey(nodes)

	default:
		return nodes
	}
}

func reOrderNodesByTenantOrderingKey(nodes []*Ydb_Maintenance.Node) []*Ydb_Maintenance.Node {
	groups := make(map[string][]*Ydb_Maintenance.Node)
	for _, node := range nodes {
		// Will not panic caused by nil pointer, because we don't pass storage nodes to this function
		tenant := node.GetDynamic().Tenant
		groups[tenant] = append(groups[tenant], node)
	}

	tenants := collections.Keys(groups)
	positions := make(map[string]int, len(tenants))
	result := make([]*Ydb_Maintenance.Node, 0, len(nodes))

	for appended := 0; appended < len(nodes); {
		for _, tenant := range tenants {
			pos := positions[tenant]
			if pos >= len(groups[tenant]) {
				continue
			}
			result = append(result, groups[tenant][pos])
			positions[tenant]++
			appended++
		}
	}

	return result
}
