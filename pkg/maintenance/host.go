package maintenance

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"

	"github.com/ydb-platform/ydbops/pkg/client/cms"
	"github.com/ydb-platform/ydbops/pkg/options"
)

const (
	MaintenanceTaskPrefix = "maintenance-"
)

func getNodesOnHost(cmsClient cms.Client, hostFQDN string) ([]*Ydb_Maintenance.Node, error) {
	nodes, err := cmsClient.Nodes()
	if err != nil {
		return nil, err
	}

	res := []*Ydb_Maintenance.Node{}

	for _, node := range nodes {
		// TODO here is the non-trivial part with Kubernetes, surgically create a shared logic
		// with Kubernetes restarters
		if node.Host == hostFQDN {
			res = append(res, node)
		}
	}

	return res, nil
}

func RequestHost(cmsClient cms.Client, opts *options.MaintenanceHostOpts) (string, error) {
	taskUID := MaintenanceTaskPrefix + uuid.New().String()

	nodes, err := getNodesOnHost(cmsClient, opts.HostFQDN)
	if err != nil {
		return "", err
	}

	taskParams := cms.MaintenanceTaskParams{
		TaskUID:          taskUID,
		AvailabilityMode: opts.GetAvailabilityMode(),
		Duration:         opts.GetMaintenanceDuration(),
		Nodes:            nodes,
	}

	task, err := cmsClient.CreateMaintenanceTask(taskParams)
	if err != nil {
		return "", fmt.Errorf("failed to create maintenance task: %w", err)
	}

	return task.GetTaskUid(), nil
}
