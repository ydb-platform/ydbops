package mock

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	. "github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func deleteFromSlice[T any](s []T, i int) []T {
  return append(s[:i], s[i+1:]...)
}

func (s *YdbMock) setPendingOrPerformed(
	currentNodeId uint32,
	availabilityMode AvailabilityMode,
) ActionState_ActionStatus {
	for _, nodeGroup := range s.nodeGroups {
		permittedOutOfGroup := 0
		for _, nodeId := range nodeGroup {
			if s.isNodeCurrentlyPermitted[nodeId] {
				permittedOutOfGroup += 1
			}
		}

		for _, nodeId := range nodeGroup {
			if nodeId == currentNodeId {
        if s.isNodeCurrentlyPermitted[currentNodeId] {
					return ActionState_ACTION_STATUS_PERFORMED
        }

				if permittedOutOfGroup == 0 {
          s.isNodeCurrentlyPermitted[currentNodeId] = true
					return ActionState_ACTION_STATUS_PERFORMED
				}

				if permittedOutOfGroup == 1 && availabilityMode != AvailabilityMode_AVAILABILITY_MODE_STRONG {
          s.isNodeCurrentlyPermitted[currentNodeId] = true
					return ActionState_ACTION_STATUS_PERFORMED
				}

				if availabilityMode == AvailabilityMode_AVAILABILITY_MODE_FORCE {
          s.isNodeCurrentlyPermitted[currentNodeId] = true
					return ActionState_ACTION_STATUS_PERFORMED
				}
			}
		}
	}

	return ActionState_ACTION_STATUS_PENDING
}

func (s *YdbMock) givePerformedOrPendingStatus(taskOptions *MaintenanceTaskOptions, action *Action) *ActionState {
	currentNodeId := action.GetLockAction().Scope.GetNodeId()
	status := s.setPendingOrPerformed(currentNodeId, taskOptions.AvailabilityMode)

	return &ActionState{
		Action:    action,
		Status:    status,
		Reason:    ActionState_ACTION_REASON_UNSPECIFIED,
		Deadline:  timestamppb.New(time.Now().Add(time.Minute * 3)),
		ActionUid: s.actionToActionUid[action],
	}
}

func (s *YdbMock) makeGroupStatesFor(taskOptions *MaintenanceTaskOptions, actionGroups []*ActionGroup) []*ActionGroupStates {
	result := []*ActionGroupStates{}
	for _, ag := range actionGroups {
		ags := &ActionGroupStates{
			ActionStates: []*ActionState{},
		}

		groupId := uuid.New().String()
		for _, action := range ag.Actions {
			if _, present := s.actionToActionUid[action]; present {
				fmt.Printf("The actionState for this action should not have existed yet")
				os.Exit(1)
			}

			s.actionToActionUid[action] = &ActionUid{
				TaskUid:  taskOptions.TaskUid,
				GroupId:  groupId,
				ActionId: uuid.New().String(),
			}

			ags.ActionStates = append(ags.ActionStates, s.givePerformedOrPendingStatus(taskOptions, action))
		}

		result = append(result, ags)
	}
	return result
}

func (s *YdbMock) cleanupActionGroupState(task *fakeMaintenanceTask, actionId string) {
	for k, actionGroupState := range task.actionGroupStates {
		for i, actionState := range actionGroupState.ActionStates {
			if actionState.ActionUid.ActionId == actionId {
        actionGroupState.ActionStates = deleteFromSlice(actionGroupState.ActionStates, i)
				break
			}
		}
    if len(actionGroupState.ActionStates) == 0 {
      task.actionGroupStates = deleteFromSlice(task.actionGroupStates, k)
      return
    }
	}
}

func (s *YdbMock) cleanupActionById(actionId string) {
	for _, task := range s.tasks {
		for k, actionGroup := range task.actionGroups {
			for i, action := range actionGroup.Actions {
				if s.actionToActionUid[action].ActionId == actionId {
					s.isNodeCurrentlyPermitted[action.GetLockAction().Scope.GetNodeId()] = false
					s.actionToActionUid[action] = nil
					s.cleanupActionGroupState(task, actionId)
          actionGroup.Actions = deleteFromSlice(actionGroup.Actions, i)
					break
				}
			}

			if len(actionGroup.Actions) == 0 {
        task.actionGroups = deleteFromSlice(task.actionGroups, k)

        if len(task.actionGroups) == 0 {
          s.tasks[task.options.TaskUid] = nil
        }

        return
			}
		}
	}
}

func (s *YdbMock) refreshStatesForTask(taskUid string) {
  task := s.tasks[taskUid]
  for _, ags := range task.actionGroupStates {
    for _, as := range ags.ActionStates {
      nodeId := as.Action.GetLockAction().Scope.GetNodeId()
      as.Status = s.setPendingOrPerformed(nodeId, task.options.AvailabilityMode)
    }
  }
}
