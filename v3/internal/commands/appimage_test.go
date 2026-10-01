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
	for _, rewrite := range webKitExecPathRewrites {
		if len(rewrite.from) != len(rewrite.to) {
			t.Fatalf("rewrite changes length: %q -> %q", rewrite.from, rewrite.to)
		}

		input := []byte("prefix\x00" + rewrite.from + "\x00suffix")
		output, changed, err := rewriteWebKitExecPaths(input)
		if err != nil {
			t.Fatal(err)
		}
		if !changed {
			t.Fatalf("rewrite %q was not applied", rewrite.from)
		}
		if !bytes.Contains(output, []byte(rewrite.to)) {
			t.Fatalf("rewritten data does not contain %q", rewrite.to)
		}
		if bytes.Contains(output, []byte(rewrite.from)) {
			t.Fatalf("rewritten data still contains %q", rewrite.from)
		}
		if len(output) != len(input) {
			t.Fatalf("rewrite changed file length from %d to %d", len(input), len(output))
		}
	}
}

func TestInstallAppRun(t *testing.T) {
	appDir := filepath.Join(t.TempDir(), "test.AppDir")
	if err := os.MkdirAll(filepath.Join(appDir, "usr"), 0755); err != nil {
		t.Fatal(err)
	}
	original := []byte("#!/bin/sh\nexec \"$APPDIR/usr/bin/testapp\" \"$@\"\n")
	if err := os.WriteFile(filepath.Join(appDir, "AppRun"), original, 0755); err != nil {
		t.Fatal(err)
	}

	if err := installAppRun(appDir); err != nil {
		t.Fatal(err)
	}

	wrapper, err := os.ReadFile(filepath.Join(appDir, "AppRun"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`OWD="${OWD:-$PWD}"`,
		`cd "$APPDIR/usr"`,
		`exec "$APPDIR/.wails-app-run" "$@"`,
	} {
		if !strings.Contains(string(wrapper), expected) {
			t.Errorf("AppRun does not contain %q", expected)
		}
	}
	gotOriginal, err := os.ReadFile(filepath.Join(appDir, ".wails-app-run"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotOriginal, original) {
		t.Fatalf("original AppRun changed: got %q, want %q", gotOriginal, original)
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
