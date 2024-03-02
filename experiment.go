package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func streamPipe(p io.ReadCloser, logger *zap.Logger) {
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

func initLogger(level string) (zap.AtomicLevel, *zap.Logger) {
	atom, _ := zap.ParseAtomicLevel(level)
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderCfg),
			zapcore.Lock(os.Stdout),
			atom,
		),
	)

	_ = zap.ReplaceGlobals(logger)
	return atom, logger
}

func main2() {
	// nssh run -A -J lb.bastion.nemax.nebiuscloud.net --ycp-profile nemax --no-yubikey  "(echo 'hello world')" eu-north1-a-ct4-28a.infra.nemax.nebiuscloud.net
  host := "eu-north1-a-ct4-28a.infra.nemax.nebiuscloud.net"

	sshCommand := "nssh"
	restartCommand := `for i in {1..4}; do echo $i; sleep 1; done`
  fullSSHArgs := []string{
    "run", "-A", "-J", "lb.bastion.nemax.nebiuscloud.net", "--ycp-profile", "nemax", "--no-yubikey", restartCommand, host,
  }

  fmt.Println(fullSSHArgs)

	cmd := exec.Command(sshCommand, fullSSHArgs...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		fmt.Println("Error on cmd.Start():", err)
	}

	_, logger := initLogger("info")

	go streamPipe(stdout, logger)
	go streamPipe(stderr, logger)

	if err := cmd.Wait(); err != nil {
		fmt.Println("Error on cmd.Wait():", err)
	}

	return
}
