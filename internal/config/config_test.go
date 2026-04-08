package config

import (
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Polaris-F/watchpid/internal/store"
)

func TestLoadPrefersEnvironment(t *testing.T) {
	root, err := ioutil.TempDir("", "watchpid-config-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(s.ConfigPath, []byte(strings.Join([]string{
		"WATCHPID_NOTIFY_CHANNELS=file1,file2",
		"WATCHPID_PUSHPLUS_TOKEN=file-token",
		"",
	}, "\n")), 0644)
	if err != nil {
		t.Fatal(err)
	}

	restoreNotifyChannels := setEnv(t, envNotifyChannels, "pushplus")
	defer restoreNotifyChannels()
	restorePushPlusToken := setEnv(t, envPushPlusToken, "env-token")
	defer restorePushPlusToken()

	cfg, err := Load(s)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Notify.PushPlus.Token != "env-token" {
		t.Fatalf("unexpected token: %q", cfg.Notify.PushPlus.Token)
	}
	if !reflect.DeepEqual(cfg.Notify.Channels, []string{"pushplus"}) {
		t.Fatalf("unexpected channels: %#v", cfg.Notify.Channels)
	}
}

func TestLoadAddsDefaultChannelWhenTokenExists(t *testing.T) {
	root, err := ioutil.TempDir("", "watchpid-config-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(s.ConfigPath, []byte("WATCHPID_PUSHPLUS_TOKEN=file-token\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	restoreNotifyChannels := unsetEnv(t, envNotifyChannels)
	defer restoreNotifyChannels()
	restorePushPlusToken := unsetEnv(t, envPushPlusToken)
	defer restorePushPlusToken()

	cfg, err := Load(s)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(cfg.Notify.Channels, []string{"pushplus"}) {
		t.Fatalf("unexpected channels: %#v", cfg.Notify.Channels)
	}
}

func TestSavePushPlusTokenWritesConfig(t *testing.T) {
	root, err := ioutil.TempDir("", "watchpid-config-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	err = SavePushPlusToken(s, "saved-token")
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadFile(s.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)

	if !strings.Contains(text, "WATCHPID_NOTIFY_CHANNELS=pushplus") {
		t.Fatalf("config missing default channel: %s", text)
	}
	if !strings.Contains(text, "WATCHPID_PUSHPLUS_TOKEN=saved-token") {
		t.Fatalf("config missing token: %s", text)
	}
}

func setEnv(t *testing.T, key, value string) func() {
	oldValue, hadValue := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatal(err)
	}
	return func() {
		if hadValue {
			_ = os.Setenv(key, oldValue)
			return
		}
		_ = os.Unsetenv(key)
	}
}

func unsetEnv(t *testing.T, key string) func() {
	oldValue, hadValue := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	return func() {
		if hadValue {
			_ = os.Setenv(key, oldValue)
			return
		}
		_ = os.Unsetenv(key)
	}
}
