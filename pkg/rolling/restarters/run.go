package restarters

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/pkg/options"
)

const (
	HostnameEnvVar = "HOSTNAME"
)

type RunRestarter struct {
	Opts   *RunOpts
	logger *zap.SugaredLogger
}

func (r RunRestarter) RestartNode(node *Ydb_Maintenance.Node) error {
	//nolint:gosec
	cmd := exec.Command(r.Opts.PayloadFilepath)

	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", HostnameEnvVar, node.Host))

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error running payload file: %w", err)
	}

	go StreamPipeIntoLogger(stdout, r.logger)
	go StreamPipeIntoLogger(stderr, r.logger)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("payload command finished with an error: %w", err)
	}
	return nil
}

func NewRunRestarter(logger *zap.SugaredLogger) *RunRestarter {
	return &RunRestarter{
		Opts:   &RunOpts{},
		logger: logger,
	}
}

func determineRunScope(spec FilterNodeParams, cluster ClusterNodesInfo) []*Ydb_Maintenance.Node {
	// Abstraction leaks here:
	restartOpts := options.RestartOptionsInstance

	if !restartOpts.Tenant && !restartOpts.Storage {
		return cluster.AllNodes
	}

	result := []*Ydb_Maintenance.Node{}
	if restartOpts.Storage {
		storageNodes := FilterStorageNodes(cluster.AllNodes)
		result = append(result, storageNodes...)
	}

	if restartOpts.Tenant {
		tenantNodes := FilterTenantNodes(cluster.AllNodes)

		selectedByTenantName := PopulateByTenantNames(tenantNodes, spec.SelectedTenants, cluster.TenantToNodeIds)

		result = append(result, selectedByTenantName...)
	}

	return result
}

func (r RunRestarter) Filter(spec FilterNodeParams, cluster ClusterNodesInfo) []*Ydb_Maintenance.Node {
	runScopeNodes := determineRunScope(spec, cluster)

	preSelectedNodes := PopulateByCommonFields(runScopeNodes, spec)

	filteredNodes := ExcludeByCommonFields(preSelectedNodes, spec)

	r.logger.Debugf("Run Restarter selected following nodes for restart: %v", filteredNodes)

	return filteredNodes
}
