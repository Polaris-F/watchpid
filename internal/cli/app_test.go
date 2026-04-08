package cli

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Polaris-F/watchpid/internal/buildinfo"
)

func TestRunHelpDoesNotRequireWritableHome(t *testing.T) {
	restoreHome, cleanupHome := makeReadOnlyHome(t)
	defer cleanupHome()
	defer restoreHome()

	restoreWatchHome := unsetEnv(t, "WATCHPID_HOME")
	defer restoreWatchHome()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code: %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "watchpid") {
		t.Fatalf("unexpected help output: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunVersionJSONDoesNotRequireWritableHome(t *testing.T) {
	restoreHome, cleanupHome := makeReadOnlyHome(t)
	defer cleanupHome()
	defer restoreHome()

	restoreWatchHome := unsetEnv(t, "WATCHPID_HOME")
	defer restoreWatchHome()

	oldVersion := buildinfo.Version
	oldCommit := buildinfo.Commit
	oldDate := buildinfo.Date
	buildinfo.Version = "v-test"
	buildinfo.Commit = "abc123"
	buildinfo.Date = "2026-04-08T10:00:00Z"
	defer func() {
		buildinfo.Version = oldVersion
		buildinfo.Commit = oldCommit
		buildinfo.Date = oldDate
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"version", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code: %d, stderr=%s", code, stderr.String())
	}

	var payload map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}

	if payload["version"] != "v-test" {
		t.Fatalf("unexpected version payload: %#v", payload)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func makeReadOnlyHome(t *testing.T) (func(), func()) {
	root, err := ioutil.TempDir("", "watchpid-cli-")
	if err != nil {
		t.Fatal(err)
	}

	home := filepath.Join(root, "home")
	if err := os.Mkdir(home, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(home, 0555); err != nil {
		t.Fatal(err)
	}

	restore := setEnv(t, "HOME", home)
	cleanup := func() {
		_ = os.Chmod(home, 0755)
		_ = os.RemoveAll(root)
	}
	return restore, cleanup
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
