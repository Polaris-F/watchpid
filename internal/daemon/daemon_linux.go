// +build linux

package daemon

import (
	"os"
	"os/exec"
	"syscall"
)

func StartDetached(executable string, args []string, logPath string) (int, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer logFile.Close()

	devNull, err := os.OpenFile("/dev/null", os.O_RDONLY, 0)
	if err != nil {
		return 0, err
	}
	defer devNull.Close()

	cmd := exec.Command(executable, args...)
	cmd.Stdin = devNull
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return pid, nil
}
