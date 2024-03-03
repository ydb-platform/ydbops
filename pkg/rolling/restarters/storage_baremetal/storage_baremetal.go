package storage_baremetal

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"
	"go.uber.org/zap"
)

type Restarter struct {
	Opts *Opts
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

func streamPipeIntoLogger(p io.ReadCloser, logger *zap.SugaredLogger) {
	buf := make([]byte, 1024)
	for {
		n, err := p.Read(buf)
		if n > 0 {
			logger.Info(string(buf[:n]))
		}
		if err != nil {
			if err != io.EOF {
				logger.Error("Error reading from pipe", zap.Error(err))
			}
			break
		}
	}
}

func (r Restarter) RestartNode(logger *zap.SugaredLogger, node *Ydb_Maintenance.Node) error {
	logger.Info(fmt.Sprintf("Restarting %s with ssh-args %v", node.Host, r.Opts.SSHArgs))

	// It is theoretically possible to guess the systemd-unit, but it is a fragile
	// solution. tarasov-egor@ will keep it here during development time for reference:
	//
	// YDBD_PORT=2135
	// YDBD_PID=$(sudo lsof -i :$YDBD_PORT | grep LISTEN | awk '{print $2}' | head -n 1)
	// YDBD_UNIT=$(sudo ps -A -o'pid,unit' | grep $YDBD_PID | awk '{print $2}')
	// sudo systemctl restart $YDBD_UNIT

	systemdUnitName := DefaultSystemdUnit
	if r.Opts.IsOldSystemdKikimr {
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

	go streamPipeIntoLogger(stdout, logger)
	go streamPipeIntoLogger(stderr, logger)

	if err := cmd.Wait(); err != nil {
		fmt.Println("TODO Error on cmd.Wait():", err)
		return err
	}

	return nil
}

func New() *Restarter {
	return &Restarter{Opts: &Opts{}}
}

func (r Restarter) Filter(logger *zap.SugaredLogger, spec *restarters.FilterNodeParams) []*Ydb_Maintenance.Node {
	allStorageNodes := util.FilterBy(spec.AllNodes,
		func(node *Ydb_Maintenance.Node) bool {
			return node.GetStorage() != nil
		},
	)

	selectedNodes := []*Ydb_Maintenance.Node{}

	if len(spec.SelectedNodeIds) > 0 {
		selectedNodes = append(selectedNodes, util.FilterBy(allStorageNodes,
			func(node *Ydb_Maintenance.Node) bool {
				return util.Contains(spec.SelectedNodeIds, node.NodeId)
			},
		)...)
	}

	if len(spec.SelectedHostFQDNs) > 0 {
		selectedNodes = append(selectedNodes, util.FilterBy(allStorageNodes,
			func(node *Ydb_Maintenance.Node) bool {
				return util.Contains(spec.SelectedHostFQDNs, node.Host)
			},
		)...)
	}

	logger.Debugf("storage_baremetal.Restarter selected following nodes for restart: %v", selectedNodes)

	return selectedNodes
}
