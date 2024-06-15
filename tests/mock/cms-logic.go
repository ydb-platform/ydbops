package mock

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	. "github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func deleteFromSlice[T any](s []T, i int) []T {
	return append(s[:i], s[i+1:]...)
}

func (s *YdbMock) setPendingOrPerformed(
	currentNodeID uint32,
	availabilityMode AvailabilityMode,
) ActionState_ActionStatus {
	for _, nodeGroup := range s.nodeGroups {
		alreadyReleased := 0
		for _, nodeID := range nodeGroup {
			if s.isNodeCurrentlyReleased[nodeID] {
				alreadyReleased++
			}
		}

		for _, nodeID := range nodeGroup {
			if nodeID != currentNodeID {
				continue
			}

			if s.isNodeCurrentlyReleased[currentNodeID] {
				return ActionState_ACTION_STATUS_PERFORMED
			}

			if alreadyReleased == 0 {
				s.isNodeCurrentlyReleased[currentNodeID] = true
				return ActionState_ACTION_STATUS_PERFORMED
			}

			if alreadyReleased == 1 &&
				(availabilityMode == AvailabilityMode_AVAILABILITY_MODE_WEAK ||
					availabilityMode == AvailabilityMode_AVAILABILITY_MODE_FORCE) {
				s.isNodeCurrentlyReleased[currentNodeID] = true
				return ActionState_ACTION_STATUS_PERFORMED
			}

			if availabilityMode == AvailabilityMode_AVAILABILITY_MODE_FORCE {
				s.isNodeCurrentlyReleased[currentNodeID] = true
				return ActionState_ACTION_STATUS_PERFORMED
			}
		}
	}

	return ActionState_ACTION_STATUS_PENDING
}

func whichStorageNodeIs(host string) uint32 {
	// fake host fqdns look like this: ydb-%d.ydb.tech
	parts := strings.Split(host, "-")
	numStr := strings.Split(parts[1], ".")[0]
	num, err := strconv.ParseUint(numStr, 10, 32)
	if err != nil {
		panic(err)
	}

	return uint32(num)
}

func nodeIdFromAction(action *Action) uint32 {
	nodeId := action.GetLockAction().Scope.GetNodeId()
	if nodeId == 0 { // Scope is Host
		host := action.GetLockAction().Scope.GetHost()
		nodeId = whichStorageNodeIs(host)
	}

	return nodeId
}

func (s *YdbMock) givePerformedOrPendingStatus(taskOptions *MaintenanceTaskOptions, action *Action) *ActionState {
	currentNodeID := nodeIdFromAction(action)

	status := s.setPendingOrPerformed(currentNodeID, taskOptions.AvailabilityMode)

	return &ActionState{
		Action:    action,
		Status:    status,
		Reason:    ActionState_ACTION_REASON_UNSPECIFIED,
		Deadline:  timestamppb.New(time.Now().Add(time.Minute * 3)),
		ActionUid: s.actionToActionUID[action],
	}
}

func (s *YdbMock) makeGroupStatesFor(taskOptions *MaintenanceTaskOptions, actionGroups []*ActionGroup) []*ActionGroupStates {
	result := []*ActionGroupStates{}
	for _, ag := range actionGroups {
		ags := &ActionGroupStates{
			ActionStates: []*ActionState{},
		}

		groupID := uuid.New().String()
		for _, action := range ag.Actions {
			if _, present := s.actionToActionUID[action]; present {
				fmt.Printf("The actionState for this action should not have existed yet")
				os.Exit(1)
			}

			s.actionToActionUID[action] = &ActionUid{
				TaskUid:  taskOptions.TaskUid,
				GroupId:  groupID,
				ActionId: uuid.New().String(),
			}

			ags.ActionStates = append(ags.ActionStates, s.givePerformedOrPendingStatus(taskOptions, action))
		}

		result = append(result, ags)
	}
	return result
}

func (s *YdbMock) cleanupActionGroupState(task *fakeMaintenanceTask, actionID string) {
	for k, actionGroupState := range task.actionGroupStates {
		for i, actionState := range actionGroupState.ActionStates {
			if actionState.ActionUid.ActionId == actionID {
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

func (s *YdbMock) cleanupActionByID(actionID string) {
	for _, task := range s.tasks {
		for k, actionGroup := range task.actionGroups {
			for i, action := range actionGroup.Actions {
				if s.actionToActionUID[action].ActionId != actionID {
					continue
				}

				nodeId := nodeIdFromAction(action)

				s.isNodeCurrentlyReleased[nodeId] = false
				delete(s.actionToActionUID, action)
				s.cleanupActionGroupState(task, actionID)
				actionGroup.Actions = deleteFromSlice(actionGroup.Actions, i)
				break
			}

			if len(actionGroup.Actions) == 0 {
				task.actionGroups = deleteFromSlice(task.actionGroups, k)

				if len(task.actionGroups) == 0 {
					delete(s.tasks, task.options.TaskUid)
				}

				return
			}
		}
	}
}

func (s *YdbMock) refreshStatesForTask(taskUID string) {
	task := s.tasks[taskUID]
	for _, ags := range task.actionGroupStates {
		for _, as := range ags.ActionStates {
			nodeID := nodeIdFromAction(as.Action)
			as.Status = s.setPendingOrPerformed(nodeID, task.options.AvailabilityMode)
		}
	}
}
