package store

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Polaris-F/watchpid/internal/model"
)

func TestNewCreatesLayout(t *testing.T) {
	root, err := ioutil.TempDir("", "watchpid-store-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	s, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	paths := []string{s.Root, s.WatchesDir, s.LogsDir, s.EventsPath, s.ConfigPath}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestListWatchesSortsNewestFirst(t *testing.T) {
	root, err := ioutil.TempDir("", "watchpid-store-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	s, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	older := &model.Watch{
		ID:        "watch-old",
		Name:      "old",
		Mode:      model.ModeRun,
		Status:    model.StatusCompleted,
		CreatedAt: time.Unix(100, 0),
	}
	newer := &model.Watch{
		ID:        "watch-new",
		Name:      "new",
		Mode:      model.ModeRun,
		Status:    model.StatusCompleted,
		CreatedAt: time.Unix(200, 0),
	}

	if err := s.SaveWatch(older); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveWatch(newer); err != nil {
		t.Fatal(err)
	}

	items, err := s.ListWatches()
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 {
		t.Fatalf("unexpected watch count: %d", len(items))
	}
	if items[0].ID != newer.ID || items[1].ID != older.ID {
		t.Fatalf("unexpected order: %#v", items)
	}
}

func TestAppendEventWritesJSONLine(t *testing.T) {
	root, err := ioutil.TempDir("", "watchpid-store-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	s, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	err = s.AppendEvent(model.Event{
		WatchID:    "watch-1",
		Name:       "demo",
		Mode:       model.ModeRun,
		Status:     model.StatusCompleted,
		OccurredAt: time.Unix(123, 0),
		Message:    "registered command finished",
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadFile(s.EventsPath)
	if err != nil {
		t.Fatal(err)
	}

	text := string(data)
	if !strings.Contains(text, "\"watch_id\":\"watch-1\"") {
		t.Fatalf("events missing watch id: %s", text)
	}
	if !strings.HasSuffix(text, "\n") {
		t.Fatalf("events file should end with newline: %q", text)
	}
}
