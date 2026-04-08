package store

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Polaris-F/watchpid/internal/model"
)

type Store struct {
	Root       string
	WatchesDir string
	LogsDir    string
	EventsPath string
	ConfigPath string
}

func New(root string) (*Store, error) {
	if root == "" {
		if envRoot := strings.TrimSpace(os.Getenv("WATCHPID_HOME")); envRoot != "" {
			root = envRoot
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			root = filepath.Join(home, ".watchpid")
		}
	}

	s := &Store{
		Root:       root,
		WatchesDir: filepath.Join(root, "watches"),
		LogsDir:    filepath.Join(root, "logs"),
		EventsPath: filepath.Join(root, "events.jsonl"),
		ConfigPath: filepath.Join(root, "config.env"),
	}

	if err := s.EnsureLayout(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) EnsureLayout() error {
	dirs := []string{s.Root, s.WatchesDir, s.LogsDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	if _, err := os.Stat(s.EventsPath); os.IsNotExist(err) {
		if err := ioutil.WriteFile(s.EventsPath, []byte(""), 0644); err != nil {
			return err
		}
	}

	if _, err := os.Stat(s.ConfigPath); os.IsNotExist(err) {
		template := strings.Join([]string{
			"# watchpid configuration",
			"# Environment variables override values in this file.",
			"# WATCHPID_NOTIFY_CHANNELS=pushplus",
			"# WATCHPID_PUSHPLUS_TOKEN=",
			"",
		}, "\n")
		if err := ioutil.WriteFile(s.ConfigPath, []byte(template), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) WatchPath(id string) string {
	return filepath.Join(s.WatchesDir, id+".json")
}

func (s *Store) LogPath(id string) string {
	return filepath.Join(s.LogsDir, id+".log")
}

func (s *Store) SaveWatch(w *model.Watch) error {
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	path := s.WatchPath(w.ID)
	tmp := path + ".tmp"
	if err := ioutil.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) LoadWatch(id string) (*model.Watch, error) {
	data, err := ioutil.ReadFile(s.WatchPath(id))
	if err != nil {
		return nil, err
	}
	var w model.Watch
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *Store) ListWatches() ([]model.Watch, error) {
	entries, err := ioutil.ReadDir(s.WatchesDir)
	if err != nil {
		return nil, err
	}

	items := make([]model.Watch, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := ioutil.ReadFile(filepath.Join(s.WatchesDir, entry.Name()))
		if err != nil {
			continue
		}
		var w model.Watch
		if err := json.Unmarshal(data, &w); err != nil {
			continue
		}
		items = append(items, w)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) AppendEvent(event model.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.EventsPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}
