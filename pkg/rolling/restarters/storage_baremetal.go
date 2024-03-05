package restarters

import (
	"fmt"
	"os/exec"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type StorageBaremetalRestarter struct {
	Opts *StorageBaremetalOpts
}

const (
	DefaultSystemdUnit  = "ydb-server-storage.service"
	InternalSystemdUnit = "kikimr"
)

func stripCommandFromArgs(args []string) (string, []string) {
	remainingSshArgs := []string{}
	command := "ssh"
	for _, arg := range args {
		if arg == "ssh" || arg == "pssh" || arg == "nssh" {
			command = arg
		} else {
			remainingSshArgs = append(remainingSshArgs, arg)
		}
	}

	return command, remainingSshArgs
}

func (r StorageBaremetalRestarter) RestartNode(logger *zap.SugaredLogger, node *Ydb_Maintenance.Node) error {
	logger.Info(fmt.Sprintf("Restarting %s with ssh-args %v", node.Host, r.Opts.SSHArgs))

	// It is theoretically possible to guess the systemd-unit, but it is a fragile
	// solution. tarasov-egor@ will keep it here during development time for reference:
	//
	// YDBD_PORT=2135
	// YDBD_PID=$(sudo lsof -i :$YDBD_PORT | grep LISTEN | awk '{print $2}' | head -n 1)
	// YDBD_UNIT=$(sudo ps -A -o'pid,unit' | grep $YDBD_PID | awk '{print $2}')
	// sudo systemctl restart $YDBD_UNIT

	systemdUnitName := DefaultSystemdUnit
	if r.Opts.IsSystemdInternalKikimr {
		systemdUnitName = InternalSystemdUnit
	}

	logger.Debug(fmt.Sprintf("Restarting %s systemd unit", systemdUnitName))

	remoteRestartCommand := fmt.Sprintf(
		`(test -x /bin/systemctl && sudo systemctl restart %s)`,
		systemdUnitName,
	)

	sshCommand, remainingSshArgs := stripCommandFromArgs(r.Opts.SSHArgs)

	fullSSHArgs := []string{"run"}
	fullSSHArgs = append(fullSSHArgs, remainingSshArgs...)
	switch sshCommand {
	case "ssh":
		fullSSHArgs = append(fullSSHArgs, node.Host, remoteRestartCommand)
	case "nssh", "pssh":
		fullSSHArgs = append(fullSSHArgs, remoteRestartCommand, node.Host)
	default:
		return fmt.Errorf("Supported ssh commands: ssh, pssh, nssh. Specified: %s", sshCommand)
	}

	cmd := exec.Command(sshCommand, fullSSHArgs...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		fmt.Println("TODO Error on cmd.Start():", err)
		return err
	}

	go StreamPipeIntoLogger(stdout, logger)
	go StreamPipeIntoLogger(stderr, logger)

	if err := cmd.Wait(); err != nil {
		fmt.Println("TODO Error on cmd.Wait():", err)
		return err
	}

	return nil
}

func NewBaremetalRestarter() *StorageBaremetalRestarter {
	return &StorageBaremetalRestarter{Opts: &StorageBaremetalOpts{}}
}

func (r StorageBaremetalRestarter) Filter(
	logger *zap.SugaredLogger,
	spec FilterNodeParams,
	cluster ClusterNodesInfo,
) []*Ydb_Maintenance.Node {
	allStorageNodes := FilterStorageNodes(cluster.AllNodes)

	selectedNodes := []*Ydb_Maintenance.Node{}

	selectedNodes = append(
		selectedNodes,
		FilterByNodeIds(allStorageNodes, spec.SelectedNodeIds)...,
	)

	selectedNodes = append(
		selectedNodes, FilterByHostFQDN(allStorageNodes, spec.SelectedHostFQDNs)...,
	)

	logger.Debugf("storage_baremetal.Restarter selected following nodes for restart: %v", selectedNodes)

	return selectedNodes
}
