//go:build full_test

package commands

import (
	"bytes"
	"github.com/wailsapp/wails/v3/internal/s"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_generateAppImage(t *testing.T) {

	tests := []struct {
		name     string
		options  *GenerateAppImageOptions
		wantErr  bool
		setup    func()
		teardown func()
	}{
		{
			name:    "Should fail if binary path is not provided",
			options: &GenerateAppImageOptions{},
			wantErr: true,
		},
		{
			name: "Should fail if Icon is not provided",
			options: &GenerateAppImageOptions{
				Binary: "testapp",
			},
			wantErr: true,
		},

		{
			name: "Should fail if desktop file is not provided",
			options: &GenerateAppImageOptions{
				Binary: "testapp",
				Icon:   "testicon",
			},
			wantErr: true,
		},
		{
			name: "Should work if inputs are valid",
			options: &GenerateAppImageOptions{
				Binary:      "testapp",
				Icon:        "appicon.png",
				DesktopFile: "testapp.desktop",
			},
			setup: func() {
				// Compile the test application
				s.CD("appimage_testfiles")
				testDir := s.CWD()
				_, err := s.EXEC(`go build -ldflags="-s -w" -o testapp`)
				if err != nil {
					t.Fatal(err)
				}
				s.DEFER(func() {
					s.CD(testDir)
					s.RM("testapp")
					s.RM("testapp-x86_64.AppImage")
				})
			},
			teardown: func() {
				s.CALLDEFER()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if err := GenerateAppImage(tt.options); (err != nil) != tt.wantErr {
				t.Errorf("generateAppImage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.teardown != nil {
				tt.teardown()
			}
		})
	}
}

func TestRewriteWebKitExecPaths(t *testing.T) {
	for _, tc := range []struct{ dir, want string }{
		{"/usr/libexec/webkitgtk-6.0", "././webkitgtk-6.0/"},
		{"/usr/lib/x86_64-linux-gnu/webkitgtk-6.0", "././webkitgtk-6.0/"},
		{"/usr/lib/aarch64-linux-gnu/webkit2gtk-4.1", "././webkit2gtk-4.1/"},
	} {
		bundle := "\x00" + tc.dir + "/injected-bundle/\x00"
		input := []byte("prefix\x00" + tc.dir + bundle + "suffix")
		output, changed, err := rewriteWebKitExecPaths(input, []string{tc.dir})
		if err != nil {
			t.Fatal(err)
		}
		if !changed || !bytes.Contains(output, []byte("\x00"+tc.want+"\x00")) {
			t.Fatalf("%s: helper path not rewritten to %q: %q", tc.dir, tc.want, output)
		}
		if !bytes.Contains(output, []byte(bundle)) {
			t.Fatalf("%s: injected-bundle path changed: %q", tc.dir, output)
		}
		if len(output) != len(input) {
			t.Fatalf("%s: rewrite changed file length from %d to %d", tc.dir, len(input), len(output))
		}
	}
	if _, _, err := rewriteWebKitExecPaths(nil, []string{"/opt/webkit"}); err == nil {
		t.Fatal("a helper directory outside /usr was accepted")
	}
}

func TestPatchWebKitLibrariesMovesHelpers(t *testing.T) {
	appDir := t.TempDir()
	dir := "/usr/lib/x86_64-linux-gnu/webkitgtk-6.0"
	if err := os.MkdirAll(filepath.Join(appDir, dir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, dir, "WebKitWebProcess"), nil, 0755); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(appDir, "usr/lib/libwebkitgtk-6.0.so.4")
	if err := os.WriteFile(lib, []byte("a\x00"+dir+"\x00b"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := patchWebKitLibraries(appDir, []string{dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(appDir, "usr/lib/webkitgtk-6.0/WebKitWebProcess")); err != nil {
		t.Fatalf("helpers not moved: %v", err)
	}
	got, _ := os.ReadFile(lib)
	if !bytes.Contains(got, []byte("\x00././webkitgtk-6.0/\x00")) {
		t.Fatalf("library not patched: %q", got)
	}
}

func TestWebKitHelperDirs(t *testing.T) {
	got := webKitHelperDirs([]string{
		"/usr/lib/x86_64-linux-gnu/webkitgtk-6.0/WebKitWebProcess",
		"/usr/lib/x86_64-linux-gnu/webkitgtk-6.0/WebKitNetworkProcess",
		"/usr/lib/x86_64-linux-gnu/libwebkitgtkinjectedbundle.so",
	})
	if len(got) != 1 || got[0] != "/usr/lib/x86_64-linux-gnu/webkitgtk-6.0" {
		t.Fatalf("helper dirs %q", got)
	}
}

func TestInstallAppRun(t *testing.T) {
	appDir := t.TempDir()
	if err := installAppRun(appDir, "testapp"); err != nil {
		t.Fatal(err)
	}

	wrapper, err := os.ReadFile(filepath.Join(appDir, "AppRun"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`OWD="${OWD:-$PWD}"`,
		`export PATH="$usr/bin:$PATH"`,
		`export LD_LIBRARY_PATH="$usr/lib:`,
		`export XDG_DATA_DIRS="$usr/share:`,
		`export GSETTINGS_SCHEMA_DIR="$usr/share/glib-2.0/schemas"`,
		`cd "$usr/lib"`,
		`exec "$usr/bin/testapp" "$@"`,
	} {
		if !strings.Contains(string(wrapper), expected) {
			t.Errorf("AppRun does not contain %q", expected)
		}
	}
}

func TestCopyGTKFilesSkipsFilesAlreadyInsideAppDir(t *testing.T) {
	appDir := filepath.Join(t.TempDir(), "test.AppDir")
	source := filepath.Join(appDir, "usr", "libexec", "WebKitNetworkProcess")
	if err := os.MkdirAll(filepath.Dir(source), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("helper"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := copyGTKFiles(appDir, []string{source}); err != nil {
		t.Fatal(err)
	}

	nestedCopy := filepath.Join(appDir, strings.TrimPrefix(filepath.Dir(source), string(os.PathSeparator)), filepath.Base(source))
	if _, err := os.Stat(nestedCopy); !os.IsNotExist(err) {
		t.Fatalf("unexpected nested helper copy at %s", nestedCopy)
	}
}

func TestCopyGTKFilesPreservesExecutableMode(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "test.AppDir")
	source := filepath.Join(root, "source", "WebKitNetworkProcess")
	if err := os.MkdirAll(filepath.Dir(source), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("helper"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := copyGTKFiles(appDir, []string{source}); err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(appDir, strings.TrimPrefix(filepath.Dir(source), string(os.PathSeparator)))
	info, err := os.Stat(filepath.Join(targetDir, filepath.Base(source)))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0755 {
		t.Fatalf("copied helper mode = %o, want 0755", got)
	}
}
