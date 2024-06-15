package maintenance

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"

	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/options"
)

const (
	TaskUuidPrefix = "maintenance-"
)

func CreateTask(opts *options.MaintenanceCreateOpts) (string, error) {
	cms := client.GetCmsClient()

	taskUID := TaskUuidPrefix + uuid.New().String()

	taskParams := client.MaintenanceTaskParams{
		TaskUID:          taskUID,
		AvailabilityMode: opts.GetAvailabilityMode(),
		Duration:         opts.GetMaintenanceDuration(),
		ScopeType:        client.HostScope,
		Hosts:            opts.HostFQDNs,
	}

	task, err := cms.CreateMaintenanceTask(taskParams)
	if err != nil {
		return "", fmt.Errorf("failed to create maintenance task: %w", err)
	}

	return task.GetTaskUid(), nil
}

func DropTask(opts *options.TaskIdOpts) error {
	cmsClient := client.GetCmsClient()
	status, err := cmsClient.DropMaintenanceTask(opts.TaskID)
	if err != nil {
		return err
	}

	fmt.Printf("Drop task %v status: %s\n", opts.TaskID, status)

	return nil
}

func queryEachTaskForActions(cmsClient *client.Cms, taskIds []string) ([]client.MaintenanceTask, error) {
	tasks := []client.MaintenanceTask{}
	for _, taskId := range taskIds {
		task, err := cmsClient.GetMaintenanceTask(taskId)
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

func ListTasks() ([]client.MaintenanceTask, error) {
	discoveryClient := client.GetDiscoveryClient()
	userSID, err := discoveryClient.WhoAmI()
	if err != nil {
		return nil, fmt.Errorf("failed to determine the user SID: %w", err)
	}

	cmsClient := client.GetCmsClient()
	taskIds, err := cmsClient.MaintenanceTasks(userSID)
	if err != nil {
		return nil, fmt.Errorf("failed to list all maintenance tasks: %w", err)
	}

	tasks, err := queryEachTaskForActions(cmsClient, taskIds)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func CompleteActions(
	taskIdOpts *options.TaskIdOpts,
	completeOpts *options.CompleteOpts,
) (*Ydb_Maintenance.ManageActionResult, error) {
	cmsClient := client.GetCmsClient()
	task, err := cmsClient.GetMaintenanceTask(taskIdOpts.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get maintenance task %v: %w", taskIdOpts.TaskID, err)
	}

	hostToActionUID := make(map[string]*Ydb_Maintenance.ActionUid)
	for _, gs := range task.GetActionGroupStates() {
		as := gs.ActionStates[0]
		scope := as.Action.GetLockAction().Scope
		host := scope.GetHost()
		if host == "" {
			return nil, fmt.Errorf("Trying to complete an action with nodeId scope, currently unimplemented")
		}

		hostToActionUID[host] = as.ActionUid
	}

	completedActions := []*Ydb_Maintenance.ActionUid{}
	for _, host := range completeOpts.HostFQDNs {
		actionUid, present := hostToActionUID[host]
		if !present {
			return nil, fmt.Errorf("Failed to complete host %s, corresponding CMS action not found.\n"+
				"This host either was never requested or already completed", host)
		}
		completedActions = append(completedActions, actionUid)
	}

	result, err := cmsClient.CompleteAction(completedActions)
	if err != nil {
		return nil, err
	}

	return result, err
}

func RefreshTask(opts *options.TaskIdOpts) (client.MaintenanceTask, error) {
	cmsClient := client.GetCmsClient()
	task, err := cmsClient.RefreshMaintenanceTask(opts.TaskID)
	if err != nil {
		return nil, err
	}

	return task, nil
}
