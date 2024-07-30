package maintenance

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydbops/pkg/client/cms"
	"google.golang.org/protobuf/types/known/durationpb"
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

type RequestHostParams struct {
	AvailabilityMode    Ydb_Maintenance.AvailabilityMode
	HostFQDN            string
	MaintenanceDuration *durationpb.Duration
}

func RequestHost(cmsClient cms.Client, params *RequestHostParams) (string, error) {
	taskUID := MaintenanceTaskPrefix + uuid.New().String()

	nodes, err := getNodesOnHost(cmsClient, params.HostFQDN)
	if err != nil {
		return "", err
	}

	taskParams := cms.MaintenanceTaskParams{
		TaskUID:          taskUID,
		AvailabilityMode: params.AvailabilityMode,
		Duration:         params.MaintenanceDuration,
		Nodes:            nodes,
	}

	task, err := cmsClient.CreateMaintenanceTask(taskParams)
	if err != nil {
		return "", fmt.Errorf("failed to create maintenance task: %w", err)
	}

	return task.GetTaskUid(), nil
}
