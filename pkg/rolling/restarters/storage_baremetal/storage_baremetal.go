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

	sshCommand, remainingSshArgs := stripCommandFromArgs(r.Opts.SSHArgs)

// YDBD_PID=$(sudo lsof -i :2135 | grep LISTEN | awk '{print $2}' | head -n 1)
// YDBD_UNIT=$(sudo ps -A -o'pid,unit' | grep $YDBD_PID | awk '{print $2}')
// sudo systemctl restart $YDBD_UNIT

// YDBD_PID=$(sudo lsof -t -a -i :2135 -u ydb)
// YDBD_UNIT=$(sudo ps -A -o'pid,unit' | grep $YDBD_PID | awk '{print $2}')
// sudo systemctl restart $YDBD_UNIT

	remoteRestartCommand := `(test -x /bin/systemctl && sudo systemctl restart ydb-server-storage.service)`

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
		fmt.Println("Error on cmd.Start():", err)
		return err
	}

	go streamPipeIntoLogger(stdout, logger)
	go streamPipeIntoLogger(stderr, logger)

	if err := cmd.Wait(); err != nil {
		fmt.Println("Error on cmd.Wait():", err)
		return err
	}

	return nil
}

func New() *Restarter {
	return &Restarter{Opts: &Opts{}}
}

func (r Restarter) Filter(_ *zap.SugaredLogger, spec restarters.FilterNodeParams) []*Ydb_Maintenance.Node {
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

	return selectedNodes
}
