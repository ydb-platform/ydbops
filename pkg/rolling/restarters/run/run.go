package run

import (
	"fmt"
	"os/exec"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"
	"go.uber.org/zap"
)

type Restarter struct {
	Opts *Opts
}

func (r Restarter) RestartNode(logger *zap.SugaredLogger, node *Ydb_Maintenance.Node) error {
	cmd := exec.Command(r.Opts.PayloadFilepath)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
    return fmt.Errorf("Error running payload file: %w", err)
	}

	go restarters.StreamPipeIntoLogger(stdout, logger)
	go restarters.StreamPipeIntoLogger(stderr, logger)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Payload command finished with an error: %w", err)
	}
  return nil
}

func New() *Restarter {
	return &Restarter{Opts: &Opts{}}
}

func (r Restarter) Filter(logger *zap.SugaredLogger, spec *restarters.FilterNodeParams) []*Ydb_Maintenance.Node {
	selectedNodes := []*Ydb_Maintenance.Node{}

	selectedNodes = append(
		selectedNodes,
		restarters.FilterByNodeIds(spec.AllNodes, spec.SelectedNodeIds)...,
	)

	selectedNodes = append(
		selectedNodes, restarters.FilterByHostFQDN(spec.AllNodes, spec.SelectedHostFQDNs)...,
	)

	logger.Debugf("storage_baremetal.Restarter selected following nodes for restart: %v", selectedNodes)

	return selectedNodes
}
