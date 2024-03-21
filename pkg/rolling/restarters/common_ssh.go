package restarters

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
)

type sshRestarter struct {
	logger *zap.SugaredLogger
}

const (
	defaultStorageSystemdUnit = "ydb-server-storage.service"
	sshBin                    = "ssh"
	psshBin                   = "pssh"
	nsshBin                   = "nssh"
)

func (r sshRestarter) stripCommandFromArgs(args []string) (string, []string) {
	remainingSSHArgs := []string{}
	command := sshBin
	for _, arg := range args {
		if arg == sshBin || arg == psshBin || arg == nsshBin {
			command = arg
		} else {
			remainingSSHArgs = append(remainingSSHArgs, arg)
		}
	}

	return command, remainingSSHArgs
}

func (r sshRestarter) restartNodeBySystemdUnit(
	node *Ydb_Maintenance.Node,
	unitName string,
	sshArgs []string,
) error {
	r.logger.Debugf("Restarting %s systemd unit", unitName)

	remoteRestartCommand := fmt.Sprintf(
		`(test -x /bin/systemctl && sudo systemctl restart %s)`,
		unitName,
	)

	sshCommand, remainingSSHArgs := r.stripCommandFromArgs(sshArgs)

	fullSSHArgs := []string{}
	fullSSHArgs = append(fullSSHArgs, remainingSSHArgs...)
	switch sshCommand {
	case sshBin:
		fullSSHArgs = append(fullSSHArgs, node.Host, remoteRestartCommand)
	case nsshBin, psshBin:
		fullSSHArgs = append(fullSSHArgs, "run", remoteRestartCommand, node.Host)
	default:
		return fmt.Errorf("supported ssh commands: ssh, pssh, nssh. Specified: %s", sshCommand)
	}

	cmd := exec.Command(sshCommand, fullSSHArgs...)

	r.logger.Debugf("Full ssh command: `%s %v`", sshCommand, strings.Join(fullSSHArgs, " "))

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	warningTime := 5 * time.Second
	ticker := time.NewTicker(warningTime)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			r.logger.Warnf("Waiting to connect to the node by SSH...")
		}
	}()

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

func newSSHRestarter(logger *zap.SugaredLogger) sshRestarter {
	return sshRestarter{
		logger: logger,
	}
}
