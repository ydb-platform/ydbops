package restarters

import (
	"fmt"
	"os/exec"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type RunRestarter struct {
	Opts *RunOpts
}

func (r RunRestarter) RestartNode(logger *zap.SugaredLogger, node *Ydb_Maintenance.Node) error {
	cmd := exec.Command(r.Opts.PayloadFilepath)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
    return fmt.Errorf("Error running payload file: %w", err)
	}

	go StreamPipeIntoLogger(stdout, logger)
	go StreamPipeIntoLogger(stderr, logger)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Payload command finished with an error: %w", err)
	}
  return nil
}

func NewRunRestarter() *RunRestarter {
	return &RunRestarter{Opts: &RunOpts{}}
}

func (r RunRestarter) Filter(logger *zap.SugaredLogger, spec FilterNodeParams, cluster ClusterNodesInfo) []*Ydb_Maintenance.Node {
	selectedNodes := []*Ydb_Maintenance.Node{}

	selectedNodes = append(
		selectedNodes,
		FilterByNodeIds(cluster.AllNodes, spec.SelectedNodeIds)...,
	)

	selectedNodes = append(
		selectedNodes, FilterByHostFQDN(cluster.AllNodes, spec.SelectedHostFQDNs)...,
	)

	logger.Debugf("storage_baremetal.Restarter selected following nodes for restart: %v", selectedNodes)

	return selectedNodes
}
