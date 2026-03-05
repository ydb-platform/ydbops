package prettyprint

import (
	"fmt"
	"strings"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"

	"github.com/ydb-platform/ydbops/pkg/client/cms"
)

func TaskToString(task cms.MaintenanceTask) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Uid: %s\n", task.GetTaskUid()))

	if task.GetRetryAfter() != nil {
		sb.WriteString(fmt.Sprintf("Retry after: %s\n", task.GetRetryAfter().AsTime().Format(time.DateTime)))
	}

	for _, gs := range task.GetActionGroupStates() {
		as := gs.ActionStates[0]

		lock := as.Action.GetLockAction()
		if lock == nil {
			sb.WriteString("  Non-lock action ")
		} else if nodeId := lock.Scope.GetNodeId(); nodeId != 0 {
			sb.WriteString(fmt.Sprintf("  Lock on node %d ", nodeId))
		} else {
			sb.WriteString(fmt.Sprintf("  Lock on host %s ", lock.Scope.GetHost()))
		}

		if as.Status == Ydb_Maintenance.ActionState_ACTION_STATUS_PERFORMED {
			sb.WriteString(fmt.Sprintf("PERFORMED, until: %s", as.Deadline.AsTime().Format(time.DateTime)))
		} else {
			sb.WriteString(fmt.Sprintf("%s, %s", as.Status.String(), as.GetReason().String()))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func ResultToString(result *Ydb_Maintenance.ManageActionResult) string {
	sb := strings.Builder{}

	for _, status := range result.ActionStatuses {
		sb.WriteString(fmt.Sprintf("  Completed action id: %s, status: %s", status.ActionUid.ActionId, status.Status))
	}

	return sb.String()
}
