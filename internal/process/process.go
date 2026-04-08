package process

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("process not found")

type Snapshot struct {
	PID         int
	Name        string
	Command     []string
	CreateToken string
	StartedAt   *time.Time
}

func SameProcess(pid int, createToken string) (bool, *Snapshot, error) {
	snap, err := Inspect(pid)
	if err != nil {
		if err == ErrNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}
	return snap.CreateToken == createToken, &snap, nil
}
