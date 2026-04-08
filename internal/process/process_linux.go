// +build linux

package process

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	linuxClockTicksOnce sync.Once
	linuxClockTicks     uint64 = 100
	linuxBootTimeOnce   sync.Once
	linuxBootTime       time.Time
	linuxBootTimeErr    error
)

func Inspect(pid int) (Snapshot, error) {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := ioutil.ReadFile(statPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Snapshot{}, ErrNotFound
		}
		return Snapshot{}, err
	}

	comm, startTicks, err := parseProcStat(data)
	if err != nil {
		return Snapshot{}, err
	}

	cmdline, _ := readProcCmdline(pid)
	name := comm
	if len(cmdline) > 0 {
		name = filepath.Base(cmdline[0])
	}

	startedAt := linuxStartedAt(startTicks)

	return Snapshot{
		PID:         pid,
		Name:        name,
		Command:     cmdline,
		CreateToken: strconv.FormatUint(startTicks, 10),
		StartedAt:   startedAt,
	}, nil
}

func parseProcStat(data []byte) (string, uint64, error) {
	text := string(bytes.TrimSpace(data))
	start := strings.Index(text, "(")
	end := strings.LastIndex(text, ")")
	if start == -1 || end == -1 || end <= start {
		return "", 0, errors.New("invalid /proc stat format")
	}

	comm := text[start+1 : end]
	fields := strings.Fields(text[end+1:])
	if len(fields) < 20 {
		return "", 0, errors.New("unexpected /proc stat field count")
	}

	startTicks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return "", 0, err
	}

	return comm, startTicks, nil
}

func readProcCmdline(pid int) ([]string, error) {
	data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return nil, err
	}
	parts := bytes.Split(bytes.TrimRight(data, "\x00"), []byte{0})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(string(part))
		if item != "" {
			out = append(out, item)
		}
	}
	return out, nil
}

func linuxStartedAt(startTicks uint64) *time.Time {
	boot, err := linuxGetBootTime()
	if err != nil {
		return nil
	}
	hz := linuxGetClockTicks()
	if hz == 0 {
		return nil
	}
	offset := time.Duration((float64(startTicks) / float64(hz)) * float64(time.Second))
	t := boot.Add(offset)
	return &t
}

func linuxGetClockTicks() uint64 {
	linuxClockTicksOnce.Do(func() {
		out, err := exec.Command("getconf", "CLK_TCK").Output()
		if err != nil {
			return
		}
		value, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
		if err == nil && value > 0 {
			linuxClockTicks = value
		}
	})
	return linuxClockTicks
}

func linuxGetBootTime() (time.Time, error) {
	linuxBootTimeOnce.Do(func() {
		data, err := ioutil.ReadFile("/proc/stat")
		if err != nil {
			linuxBootTimeErr = err
			return
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "btime ") {
				continue
			}
			value, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "btime")), 10, 64)
			if err != nil {
				linuxBootTimeErr = err
				return
			}
			linuxBootTime = time.Unix(value, 0)
			return
		}
		linuxBootTimeErr = errors.New("btime not found in /proc/stat")
	})
	return linuxBootTime, linuxBootTimeErr
}
