package watch

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Polaris-F/watchpid/internal/model"
	"github.com/Polaris-F/watchpid/internal/store"
)

func TestRegisterRunCompletesAndPersistsState(t *testing.T) {
	root, err := ioutil.TempDir("", "watchpid-watch-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(s)
	w, err := service.RegisterRun("smoke", helperCommand("stdout=hello", "stderr=world", "exit=0"), false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if w.Status != model.StatusCompleted {
		t.Fatalf("unexpected status: %s", w.Status)
	}
	if w.ExitCode == nil || *w.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %#v", w.ExitCode)
	}
	if w.TargetPID == 0 {
		t.Fatal("expected target pid")
	}
	if w.CompletedAt == nil {
		t.Fatal("expected completion time")
	}
	if w.NotificationEnabled {
		t.Fatalf("expected notifications disabled without config: %#v", w)
	}
	if !strings.Contains(w.NotificationError, "PushPlus token not configured") {
		t.Fatalf("unexpected notification error: %s", w.NotificationError)
	}

	logData, err := ioutil.ReadFile(w.LogPath)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(logData)
	if !strings.Contains(logText, "hello") || !strings.Contains(logText, "world") {
		t.Fatalf("unexpected log contents: %s", logText)
	}

	eventData, err := ioutil.ReadFile(s.EventsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(eventData), "registered command finished") {
		t.Fatalf("unexpected events: %s", string(eventData))
	}
}

func TestRegisterRunCapturesFailureExitCode(t *testing.T) {
	root, err := ioutil.TempDir("", "watchpid-watch-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(s)
	w, err := service.RegisterRun("failure", helperCommand("exit=7"), false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if w.Status != model.StatusFailed {
		t.Fatalf("unexpected status: %s", w.Status)
	}
	if w.ExitCode == nil || *w.ExitCode != 7 {
		t.Fatalf("unexpected exit code: %#v", w.ExitCode)
	}
}

func TestHelperProcess(t *testing.T) {
	args := os.Args
	sep := -1
	for i, arg := range args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep == -1 {
		return
	}

	exitCode := 0
	for _, arg := range args[sep+1:] {
		if strings.HasPrefix(arg, "stdout=") {
			fmt.Fprint(os.Stdout, strings.TrimPrefix(arg, "stdout="))
			continue
		}
		if strings.HasPrefix(arg, "stderr=") {
			fmt.Fprint(os.Stderr, strings.TrimPrefix(arg, "stderr="))
			continue
		}
		if strings.HasPrefix(arg, "exit=") {
			value := strings.TrimPrefix(arg, "exit=")
			parsed, err := strconv.Atoi(value)
			if err != nil {
				os.Exit(2)
			}
			exitCode = parsed
		}
	}

	os.Exit(exitCode)
}

func helperCommand(args ...string) []string {
	command := []string{os.Args[0], "-test.run=TestHelperProcess", "--"}
	command = append(command, args...)
	return command
}
