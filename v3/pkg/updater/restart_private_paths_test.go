package updater_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

func TestRestartKeepsTheHelperLogAndReadyFileInAPrivateDirectory(t *testing.T) {
	var cmd *exec.Cmd
	t.Cleanup(updater.SetSelfExecutableForTest(os.Executable))
	t.Cleanup(updater.SetNewDetachedCommandForTest(func(string) *exec.Cmd {
		cmd = exec.Command("true")
		return cmd
	}))
	t.Cleanup(updater.SetWaitForHelperReadyForTest(func(string, time.Duration) error { return nil }))

	body := []byte("payload")
	p := &fakeProvider{name: "p", rel: &updater.Release{Version: "2.0.0", Artifact: updater.Artifact{Filename: "app.bin", Size: int64(len(body))}}, body: body}
	u := updater.New(&fakeHost{})
	if err := u.Init(updater.Config{CurrentVersion: "1.0.0", Providers: []updater.Provider{p}}); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := u.DownloadAndInstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(u.DownloadedPath())) })
	if err := u.Restart(context.Background()); err != nil {
		t.Fatal(err)
	}
	var logPath string
	for _, e := range cmd.Env {
		if v, ok := strings.CutPrefix(e, "WAILS_UPDATER_HELPER_LOG="); ok {
			logPath = v
		}
	}
	if logPath == "" {
		t.Fatal("no helper log path")
	}
	guessable := filepath.Join(os.TempDir(), "wails-update-"+strconv.Itoa(os.Getpid())+".log")
	if logPath == guessable {
		t.Fatalf("the helper log is at the guessable %s", logPath)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(logPath)) })
	info, err := os.Stat(filepath.Dir(logPath))
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("log directory: %v, %v", info, err)
	}
}
