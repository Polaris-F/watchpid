package model

import "time"

type WatchMode string

const (
	ModeWatch WatchMode = "watch"
	ModeRun   WatchMode = "run"
)

type WatchStatus string

const (
	StatusPending   WatchStatus = "pending"
	StatusWatching  WatchStatus = "watching"
	StatusCompleted WatchStatus = "completed"
	StatusFailed    WatchStatus = "failed"
	StatusFinished  WatchStatus = "finished"
	StatusCanceled  WatchStatus = "canceled"
)

type Watch struct {
	ID                   string      `json:"id"`
	Name                 string      `json:"name"`
	Mode                 WatchMode   `json:"mode"`
	Status               WatchStatus `json:"status"`
	Command              []string    `json:"command,omitempty"`
	CommandString        string      `json:"command_string,omitempty"`
	TargetPID            int         `json:"target_pid,omitempty"`
	TargetCreateToken    string      `json:"target_create_token,omitempty"`
	TargetStartedAt      *time.Time  `json:"target_started_at,omitempty"`
	WatcherPID           int         `json:"watcher_pid,omitempty"`
	LogPath              string      `json:"log_path,omitempty"`
	ExitCode             *int        `json:"exit_code,omitempty"`
	CreatedAt            time.Time   `json:"created_at"`
	StartedAt            *time.Time  `json:"started_at,omitempty"`
	CompletedAt          *time.Time  `json:"completed_at,omitempty"`
	LastCheckedAt        *time.Time  `json:"last_checked_at,omitempty"`
	CancelRequested      bool        `json:"cancel_requested,omitempty"`
	NotificationEnabled  bool        `json:"notification_enabled"`
	NotificationChannels []string    `json:"notification_channels,omitempty"`
	NotificationSent     bool        `json:"notification_sent,omitempty"`
	NotificationError    string      `json:"notification_error,omitempty"`
	LastError            string      `json:"last_error,omitempty"`
}

func (w Watch) IsActive() bool {
	return w.Status == StatusPending || w.Status == StatusWatching
}

type Event struct {
	WatchID      string      `json:"watch_id"`
	Name         string      `json:"name"`
	Mode         WatchMode   `json:"mode"`
	Status       WatchStatus `json:"status"`
	TargetPID    int         `json:"target_pid,omitempty"`
	ExitCode     *int        `json:"exit_code,omitempty"`
	OccurredAt   time.Time   `json:"occurred_at"`
	Message      string      `json:"message,omitempty"`
	Notification string      `json:"notification,omitempty"`
}
