package watch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"watchpid/internal/config"
	"watchpid/internal/daemon"
	"watchpid/internal/model"
	"watchpid/internal/notify"
	"watchpid/internal/process"
	"watchpid/internal/store"
)

type Service struct {
	Store *store.Store
}

func NewService(s *store.Store) *Service {
	return &Service{Store: s}
}

func (s *Service) RegisterExisting(pid int, name string, detach bool) (*model.Watch, error) {
	snap, err := process.Inspect(pid)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		name = snap.Name
		if name == "" {
			name = fmt.Sprintf("pid-%d", pid)
		}
	}

	now := time.Now()
	w := &model.Watch{
		ID:                   newWatchID(),
		Name:                 name,
		Mode:                 model.ModeWatch,
		Status:               model.StatusPending,
		TargetPID:            pid,
		TargetCreateToken:    snap.CreateToken,
		TargetStartedAt:      snap.StartedAt,
		CreatedAt:            now,
		StartedAt:            &now,
		LogPath:              s.Store.LogPath(newWatchID()),
		NotificationEnabled:  true,
		NotificationChannels: []string{"pushplus"},
	}
	w.LogPath = s.Store.LogPath(w.ID)

	if err := s.Store.SaveWatch(w); err != nil {
		return nil, err
	}

	if detach {
		if err := s.spawnDaemon("watch", w.ID, w.LogPath, w); err != nil {
			return nil, err
		}
		return w, nil
	}

	if err := s.MonitorExisting(w.ID); err != nil {
		return nil, err
	}
	return s.Store.LoadWatch(w.ID)
}

func (s *Service) RegisterRun(name string, command []string, detach bool, foregroundStdout io.Writer, foregroundStderr io.Writer) (*model.Watch, error) {
	if len(command) == 0 {
		return nil, errors.New("empty command")
	}
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(command[0])
	}

	now := time.Now()
	w := &model.Watch{
		ID:                   newWatchID(),
		Name:                 name,
		Mode:                 model.ModeRun,
		Status:               model.StatusPending,
		Command:              command,
		CommandString:        strings.Join(command, " "),
		CreatedAt:            now,
		LogPath:              s.Store.LogPath(newWatchID()),
		NotificationEnabled:  true,
		NotificationChannels: []string{"pushplus"},
	}
	w.LogPath = s.Store.LogPath(w.ID)

	if err := s.Store.SaveWatch(w); err != nil {
		return nil, err
	}

	if detach {
		if err := s.spawnDaemon("run", w.ID, w.LogPath, w); err != nil {
			return nil, err
		}
		return w, nil
	}

	if err := s.RunRegistered(w.ID, foregroundStdout, foregroundStderr); err != nil {
		return nil, err
	}
	return s.Store.LoadWatch(w.ID)
}

func (s *Service) spawnDaemon(mode, id, logPath string, w *model.Watch) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{"daemon", mode, "--id", id}
	pid, err := daemon.StartDetached(executable, args, logPath)
	if err != nil {
		return err
	}
	w.WatcherPID = pid
	w.Status = model.StatusWatching
	now := time.Now()
	w.LastCheckedAt = &now
	return s.Store.SaveWatch(w)
}

func (s *Service) MonitorExisting(id string) error {
	w, err := s.Store.LoadWatch(id)
	if err != nil {
		return err
	}

	w.Status = model.StatusWatching
	if w.StartedAt == nil {
		now := time.Now()
		w.StartedAt = &now
	}
	if err := s.Store.SaveWatch(w); err != nil {
		return err
	}

	for {
		active, err := s.handleCancel(id)
		if err != nil || !active {
			return err
		}

		same, _, err := process.SameProcess(w.TargetPID, w.TargetCreateToken)
		now := time.Now()
		w.LastCheckedAt = &now
		if err != nil {
			w.LastError = err.Error()
			_ = s.Store.SaveWatch(w)
			return err
		}
		if !same {
			w.Status = model.StatusFinished
			w.CompletedAt = &now
			w.WatcherPID = 0
			_ = s.dispatchNotification(w, "Process finished", s.renderFinishMessage(*w))
			_ = s.Store.SaveWatch(w)
			_ = s.Store.AppendEvent(model.Event{
				WatchID:      w.ID,
				Name:         w.Name,
				Mode:         w.Mode,
				Status:       w.Status,
				TargetPID:    w.TargetPID,
				OccurredAt:   now,
				Message:      "observed target process exit",
				Notification: notificationState(w),
			})
			return nil
		}
		_ = s.Store.SaveWatch(w)
		time.Sleep(5 * time.Second)
	}
}

func (s *Service) RunRegistered(id string, foregroundStdout io.Writer, foregroundStderr io.Writer) error {
	w, err := s.Store.LoadWatch(id)
	if err != nil {
		return err
	}
	if len(w.Command) == 0 {
		return errors.New("watch has no command")
	}

	cmd := exec.Command(w.Command[0], w.Command[1:]...)
	if foregroundStdout != nil {
		cmd.Stdout = foregroundStdout
	} else {
		logFile, err := os.OpenFile(w.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer logFile.Close()
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}
	if foregroundStderr != nil {
		cmd.Stderr = foregroundStderr
	}

	if err := cmd.Start(); err != nil {
		w.Status = model.StatusFailed
		now := time.Now()
		w.CompletedAt = &now
		w.LastError = err.Error()
		_ = s.Store.SaveWatch(w)
		return err
	}

	now := time.Now()
	w.Status = model.StatusWatching
	w.TargetPID = cmd.Process.Pid
	w.WatcherPID = os.Getpid()
	w.StartedAt = &now

	snap, err := process.Inspect(cmd.Process.Pid)
	if err == nil {
		w.TargetCreateToken = snap.CreateToken
		w.TargetStartedAt = snap.StartedAt
	}
	if err := s.Store.SaveWatch(w); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	for {
		select {
		case waitErr := <-done:
			return s.finalizeRun(w, waitErr)
		case <-time.After(2 * time.Second):
			active, err := s.handleCancel(w.ID)
			if err != nil {
				return err
			}
			if !active {
				return nil
			}
			now := time.Now()
			w.LastCheckedAt = &now
			_ = s.Store.SaveWatch(w)
		}
	}
}

func (s *Service) finalizeRun(w *model.Watch, waitErr error) error {
	now := time.Now()
	w.CompletedAt = &now
	w.WatcherPID = 0

	exitCode := 0
	if waitErr != nil {
		exitCode = extractExitCode(waitErr)
		if exitCode == 0 {
			exitCode = 1
		}
	}
	w.ExitCode = &exitCode
	if exitCode == 0 {
		w.Status = model.StatusCompleted
	} else {
		w.Status = model.StatusFailed
	}

	_ = s.dispatchNotification(w, "Command finished", s.renderFinishMessage(*w))
	if err := s.Store.SaveWatch(w); err != nil {
		return err
	}
	return s.Store.AppendEvent(model.Event{
		WatchID:      w.ID,
		Name:         w.Name,
		Mode:         w.Mode,
		Status:       w.Status,
		TargetPID:    w.TargetPID,
		ExitCode:     w.ExitCode,
		OccurredAt:   now,
		Message:      "registered command finished",
		Notification: notificationState(w),
	})
}

func (s *Service) Cancel(id string) (*model.Watch, error) {
	w, err := s.Store.LoadWatch(id)
	if err != nil {
		return nil, err
	}
	if !w.IsActive() {
		return w, nil
	}
	w.CancelRequested = true
	if err := s.Store.SaveWatch(w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) handleCancel(id string) (bool, error) {
	w, err := s.Store.LoadWatch(id)
	if err != nil {
		return false, err
	}
	if !w.CancelRequested {
		return true, nil
	}
	now := time.Now()
	w.Status = model.StatusCanceled
	w.CompletedAt = &now
	w.WatcherPID = 0
	if err := s.Store.SaveWatch(w); err != nil {
		return false, err
	}
	if err := s.Store.AppendEvent(model.Event{
		WatchID:    w.ID,
		Name:       w.Name,
		Mode:       w.Mode,
		Status:     w.Status,
		TargetPID:  w.TargetPID,
		OccurredAt: now,
		Message:    "watch canceled",
	}); err != nil {
		return false, err
	}
	return false, nil
}

func (s *Service) dispatchNotification(w *model.Watch, title, body string) error {
	cfg, err := config.Load(s.Store)
	if err != nil {
		w.NotificationError = err.Error()
		return err
	}

	channels := cfg.Notify.Channels
	if len(channels) == 0 {
		w.NotificationEnabled = false
		w.NotificationError = config.MissingTokenHint(s.Store)
		return nil
	}

	message := notify.Message{Title: title + ": " + w.Name, Body: body}
	for _, channel := range channels {
		switch channel {
		case "pushplus":
			notifier := notify.PushPlus{Token: cfg.Notify.PushPlus.Token}
			if err := notifier.Send(context.Background(), message); err != nil {
				w.NotificationError = err.Error()
				return err
			}
			w.NotificationSent = true
			w.NotificationError = ""
		default:
			w.NotificationError = "unsupported notifier: " + channel
		}
	}
	return nil
}

func (s *Service) renderFinishMessage(w model.Watch) string {
	parts := []string{
		"watchpid notification",
		"",
		"Name: " + w.Name,
		"Mode: " + string(w.Mode),
		"Status: " + string(w.Status),
	}

	if w.TargetPID > 0 {
		parts = append(parts, fmt.Sprintf("PID: %d", w.TargetPID))
	}
	if w.ExitCode != nil {
		parts = append(parts, fmt.Sprintf("ExitCode: %d", *w.ExitCode))
	}
	if w.TargetStartedAt != nil {
		parts = append(parts, "StartedAt: "+w.TargetStartedAt.Format(time.RFC3339))
	}
	if w.CompletedAt != nil {
		parts = append(parts, "CompletedAt: "+w.CompletedAt.Format(time.RFC3339))
	}
	if w.LogPath != "" {
		parts = append(parts, "LogPath: "+w.LogPath)
	}
	return strings.Join(parts, "\n")
}

func newWatchID() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		now := time.Now().UnixNano()
		return fmt.Sprintf("watch-%d", now)
	}
	return "watch-" + hex.EncodeToString(buf)
}

func extractExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return 1
}

func notificationState(w *model.Watch) string {
	if w.NotificationSent {
		return "sent"
	}
	if w.NotificationError != "" {
		return "error: " + w.NotificationError
	}
	return "skipped"
}
