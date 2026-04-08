package config

import (
	"bufio"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/Polaris-F/watchpid/internal/store"
)

const (
	envNotifyChannels = "WATCHPID_NOTIFY_CHANNELS"
	envPushPlusToken  = "WATCHPID_PUSHPLUS_TOKEN"
)

type Config struct {
	Notify NotifyConfig
}

type NotifyConfig struct {
	Channels []string
	PushPlus PushPlusConfig
}

type PushPlusConfig struct {
	Token string
}

func Load(s *store.Store) (Config, error) {
	values := map[string]string{}

	fileValues, err := readEnvFile(s.ConfigPath)
	if err != nil {
		return Config{}, err
	}
	for k, v := range fileValues {
		values[k] = v
	}

	if v := strings.TrimSpace(os.Getenv(envNotifyChannels)); v != "" {
		values[envNotifyChannels] = v
	}
	if v := strings.TrimSpace(os.Getenv(envPushPlusToken)); v != "" {
		values[envPushPlusToken] = v
	}

	cfg := Config{}
	cfg.Notify.Channels = splitCSV(values[envNotifyChannels])
	cfg.Notify.PushPlus.Token = strings.TrimSpace(values[envPushPlusToken])

	if cfg.Notify.PushPlus.Token != "" && len(cfg.Notify.Channels) == 0 {
		cfg.Notify.Channels = []string{"pushplus"}
	}

	return cfg, nil
}

func SavePushPlusToken(s *store.Store, token string) error {
	values, err := readEnvFile(s.ConfigPath)
	if err != nil {
		return err
	}

	values[envPushPlusToken] = strings.TrimSpace(token)
	if strings.TrimSpace(values[envNotifyChannels]) == "" {
		values[envNotifyChannels] = "pushplus"
	}

	lines := make([]string, 0, len(values))
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines = append(lines, "# watchpid configuration")
	lines = append(lines, "# Environment variables override values in this file.")
	for _, k := range keys {
		lines = append(lines, k+"="+values[k])
	}
	lines = append(lines, "")
	return ioutil.WriteFile(s.ConfigPath, []byte(strings.Join(lines, "\n")), 0644)
}

func MissingTokenHint(s *store.Store) string {
	return "PushPlus token not configured. Set WATCHPID_PUSHPLUS_TOKEN or run `watchpid notify setup` to save one into " + s.ConfigPath + "."
}

func readEnvFile(path string) (map[string]string, error) {
	values := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return values, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}
