package maintenance

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydbops/pkg/cms"
	"github.com/ydb-platform/ydbops/pkg/connection"
	"github.com/ydb-platform/ydbops/pkg/options"
	"go.uber.org/zap"
)

const (
	MaintenanceTaskPrefix = "maintenance-"
)

func getNodesOnHost(cms *cms.Client, hostFQDN string) ([]*Ydb_Maintenance.Node, error) {
	nodes, err := cms.Nodes()
	if err != nil {
		return nil, err
	}

  res := []*Ydb_Maintenance.Node{}

  for _, node := range(nodes) {
    // TODO here is the non-trivial part with Kubernetes, surgically create a shared logic 
    // with Kubernetes restarters
    if node.Host == hostFQDN {
      res = append(res, node)
    }
  }

	return res, nil
}

func RequestHost(
	rootOpts options.RootOptions,
	logger *zap.SugaredLogger,
	opts *options.MaintenanceHostOpts,
) (string, error) {
  // TODO this `PrepareClients` is weird. I'm forcing the user to always create two clients. 
  // The usage in the next line does not need discovery client at all
	cmsClient, _, err := connection.PrepareClients(rootOpts, options.DefaultRetryCount, logger)

	taskUID := MaintenanceTaskPrefix + uuid.New().String()

	nodes, err := getNodesOnHost(cmsClient, opts.HostFQDN)

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
