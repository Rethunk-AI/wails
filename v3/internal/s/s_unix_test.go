//go:build unix

package s

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestMOVE_CrossDevice(t *testing.T) {
	srcDir, err := os.MkdirTemp("/dev/shm", "wails-move-")
	if err != nil {
		t.Skip("no /dev/shm:", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(srcDir) })
	dstDir, err := os.MkdirTemp("/var/tmp", "wails-move-")
	if err != nil {
		t.Skip("no /var/tmp:", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dstDir) })
	var a, b syscall.Stat_t
	if syscall.Stat(srcDir, &a) != nil || syscall.Stat(dstDir, &b) != nil || a.Dev == b.Dev {
		t.Skip("/dev/shm and /var/tmp share a filesystem")
	}

	src := filepath.Join(srcDir, "app.AppImage")
	if err := os.WriteFile(src, []byte("elf"), 0o755); err != nil {
		t.Fatal(err)
	}
	MOVE(src, dstDir)

	got, err := os.Stat(filepath.Join(dstDir, "app.AppImage"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, want 0755", got.Mode().Perm())
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source still present: %v", err)
	}
}

func TestCOPY_KeepsMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "app")
	if err := os.WriteFile(src, []byte("elf"), 0o755); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "bin")
	if err := os.Mkdir(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	COPY(src, dst)

	got, err := os.Stat(filepath.Join(dst, "app"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, want 0755", got.Mode().Perm())
	}
}
