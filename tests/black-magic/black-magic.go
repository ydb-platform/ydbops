package blackmagic

import (
	"fmt"
	"regexp"

	"github.com/google/go-cmp/cmp"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"google.golang.org/protobuf/testing/protocmp"
)

var uuidRegex = regexp.MustCompile(`^(rolling-restart-|maintenance-)?[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Here is some black magic. Problem is: `ydbops` produces random UUIDs during execution. This
// is a stateful checker that checks all the string fields in test scenario, and if you specified equal string labels
// in test scenario for uuids (e.g. test-uuid-1 and test-uuid-1 in some subsequent object), then this comparer makes
// sure that two ACTUALLY passed uuids in these exact locations are ALSO the same.
//
// TODO jorres@: Maybe think about adding a hidden `--static-uuids` purely for e2e testing, not inventing
// this complicated crutch here...
func UUIDComparer(expectedPlaceholders, actualPlaceholders map[string]int) cmp.Option {
	return cmp.Comparer(func(expected, actual string) bool {
		if expected == actual {
			return true
		}

		if !uuidRegex.MatchString(expected) && !uuidRegex.MatchString(actual) {
			return expected == actual
		}

		expectedValue, expectedOk := expectedPlaceholders[expected]
		actualValue, actualOk := actualPlaceholders[actual]

		if !expectedOk && !actualOk {
			placeholder := len(expectedPlaceholders) + 1
			expectedPlaceholders[expected] = placeholder
			actualPlaceholders[actual] = placeholder
			return true
		}

		if expectedOk && actualOk {
			if expectedValue != actualValue {
				fmt.Printf(
					"UuidComparer: failed comparing %s and %s. Previously seen one of those matching with a different uuid.\n",
					expected, actual,
				)
			}
			return expectedValue == actualValue
		}

		fmt.Printf(
			"UuidComparer: when comparing %s and %s, I have previously seen only one of those."+
				" This means that actual message contained uuid different from what was expected.\n",
			expected, actual,
		)

		return false
	})
}

func ActionGroupSorter() cmp.Option {
	return protocmp.SortRepeated(func(a, b *Ydb_Maintenance.ActionGroup) bool {
		aNode := a.Actions[0].GetLockAction().GetScope().GetNodeId()
		bNode := b.Actions[0].GetLockAction().GetScope().GetNodeId()
		return aNode < bNode
	})
}

// Here is some more black magic.
//
// When user specifies what CMS requests are expected, within one CompleteActionRequest
// ActionUids can shuffle if we restart more than one tenant in parallel.
//
// This small helper relaxes this behaviour - we only expect in each CompleteActionRequest
// the dynnodes of DIFFERENT tenants, but not the order.
func ActionUidSorter() cmp.Option {
	return protocmp.SortRepeated(func(a, b *Ydb_Maintenance.ActionUid) bool {
		if a.GroupId != b.GroupId {
			return a.GroupId < b.GroupId
		}
		return a.ActionId < b.ActionId
	})
}
