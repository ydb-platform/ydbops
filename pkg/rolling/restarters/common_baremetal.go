package restarters

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type baremetalRestarter struct {
	logger *zap.SugaredLogger
}

const (
	defaultStorageSystemdUnit  = "ydb-server-storage.service"
	internalStorageSystemdUnit = "kikimr"
)

func (r baremetalRestarter) stripCommandFromArgs(args []string) (string, []string) {
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

func (r baremetalRestarter) restartNodeBySystemdUnit(node *Ydb_Maintenance.Node, unitName string, sshArgs []string) error {
	r.logger.Debugf("Restarting %s systemd unit", unitName)

	remoteRestartCommand := fmt.Sprintf(
		`(test -x /bin/systemctl && sudo systemctl restart %s)`,
		unitName,
	)

	sshCommand, remainingSshArgs := r.stripCommandFromArgs(sshArgs)

	fullSSHArgs := []string{}
	fullSSHArgs = append(fullSSHArgs, remainingSshArgs...)
	switch sshCommand {
	case "ssh":
		fullSSHArgs = append(fullSSHArgs, node.Host, remoteRestartCommand)
	case "nssh", "pssh":
		fullSSHArgs = append(fullSSHArgs, "run", remoteRestartCommand, node.Host)
	default:
		return fmt.Errorf("Supported ssh commands: ssh, pssh, nssh. Specified: %s", sshCommand)
	}

	r.logger.Debugf("Full ssh command: ", strings.Join(fullSSHArgs, " "))

	cmd := exec.Command(sshCommand, fullSSHArgs...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		r.logger.Errorf("Failed to start remote command: ", err)
		return err
	}

	go StreamPipeIntoLogger(stdout, r.logger)
	go StreamPipeIntoLogger(stderr, r.logger)

	if err := cmd.Wait(); err != nil {
		r.logger.Errorf("Remote command finished with an error:", err)
		return err
	}

	return nil
}

func newBaremetalRestarter(logger *zap.SugaredLogger) baremetalRestarter {
	return baremetalRestarter{
		logger: logger,
	}
}
