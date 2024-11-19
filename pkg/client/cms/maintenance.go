package cms

import (
	"context"
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/draft/Ydb_Maintenance_V1"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"

	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/utils"
)

const (
	TaskUuidPrefix = "maintenance-"
)

type CreateTaskParams struct {
	HostFQDNs                  []string
	MaintenanceDurationSeconds int
	AvailabilityMode           string
}

type Maintenance interface {
	CompleteAction([]*Ydb_Maintenance.ActionUid) (*Ydb_Maintenance.ManageActionResult, error)
	CompleteActions(string, []string) (*Ydb_Maintenance.ManageActionResult, error)
	CreateMaintenanceTask(MaintenanceTaskParams) (MaintenanceTask, error)
	DropMaintenanceTask(string) (string, error)
	DropTask(string) error
	GetMaintenanceTask(string) (MaintenanceTask, error)
	ListTasksForUser(string) ([]MaintenanceTask, error)
	MaintenanceTasks(string) ([]MaintenanceTask, error)
	RefreshMaintenanceTask(string) (MaintenanceTask, error)
	RefreshTask(string) (MaintenanceTask, error)
}

// CompleteActions implements Client.
func (d *defaultCMSClient) CompleteActions(taskID string, hosts []string) (*Ydb_Maintenance.ManageActionResult, error) {
	task, err := d.GetMaintenanceTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get maintenance task %v: %w", taskID, err)
	}

	nodeIds, errIds := utils.GetNodeIds(hosts)
	hostFQDNs, errFqdns := utils.GetNodeFQDNs(hosts)

	if errIds != nil && errFqdns != nil {
		return nil, fmt.Errorf(
			"failed to parse --hosts argument as node ids (%w) or host fqdns (%w)",
			errIds,
			errFqdns,
		)
	}

	hostFQDNToActionUID := make(map[string]*Ydb_Maintenance.ActionUid)
	nodeIdToActionUID := make(map[uint32]*Ydb_Maintenance.ActionUid)
	for _, gs := range task.GetActionGroupStates() {
		as := gs.ActionStates[0]
		scope := as.Action.GetLockAction().Scope

		hostFqdn := scope.GetHost()
		nodeId := scope.GetNodeId()
		switch {
		case hostFqdn != "":
			hostFQDNToActionUID[hostFqdn] = as.ActionUid
		case nodeId != 0:
			nodeIdToActionUID[nodeId] = as.ActionUid
		default:
			return nil, fmt.Errorf(
				"failed to complete action. An action's scope didn't contain host or nodeId: %+v. Contact the developers",
				scope,
			)
		}
	}

	var finishedActions []*Ydb_Maintenance.ActionUid

	if errIds == nil {
		finishedActions, err = getFinishedActions(nodeIds, nodeIdToActionUID)
	} else {
		finishedActions, err = getFinishedActions(hostFQDNs, hostFQDNToActionUID)
	}

	if err != nil {
		return nil, err
	}

	return d.CompleteAction(finishedActions)
}

func getFinishedActions[T uint32 | string](nodes []T, nodeToActionUID map[T]*Ydb_Maintenance.ActionUid) ([]*Ydb_Maintenance.ActionUid, error) {
	finishedActions := []*Ydb_Maintenance.ActionUid{}
	for _, host := range nodes {
		actionUid, present := nodeToActionUID[host]
		if !present {
			return nil, fmt.Errorf("Failed to complete host %v, corresponding CMS action not found.\n"+
				"This host either was never requested or already completed", host)
		}
		finishedActions = append(finishedActions, actionUid)
	}

	return finishedActions, nil
}

func (d *defaultCMSClient) queryEachTaskForActions(taskIds []string) ([]MaintenanceTask, error) {
	tasks := []MaintenanceTask{}
	for _, taskId := range taskIds {
		task, err := d.GetMaintenanceTask(taskId)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to list all maintenance tasks, failure to obtain detailed info about task %v: %w",
				taskId,
				err,
			)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// DropTask implements Client.
func (d *defaultCMSClient) DropTask(taskID string) error {
	// TODO(shmel1k@): add status
	_, err := d.DropMaintenanceTask(taskID)
	if err != nil {
		return err
	}

	// TODO(shmel1k@): return back commentaries.
	// fmt.Printf("Drop task %v status: %s\n", taskID, status)

	return nil
}

// ListTasks implements Client.
func (d *defaultCMSClient) ListTasksForUser(userSID string) ([]MaintenanceTask, error) {
	return d.MaintenanceTasks(userSID)
}

// RefreshTask implements Client.
func (d *defaultCMSClient) RefreshTask(taskID string) (MaintenanceTask, error) {
	var result Ydb_Maintenance.MaintenanceTaskResult
	_, err := d.executeMaintenanceOperation(&result, func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (client.OperationResponse, error) {
		return cl.RefreshMaintenanceTask(ctx, &Ydb_Maintenance.RefreshMaintenanceTaskRequest{
			OperationParams: d.connectionsFactory.OperationParams(),
			TaskUid:         taskID,
		})
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}
