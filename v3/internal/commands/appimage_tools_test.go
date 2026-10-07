package commands

import (
	"path/filepath"
	"testing"
)

func TestToolPath(t *testing.T) {
	t.Setenv("WAILS_TEST_TOOL", "")
	if got, err := toolPath("", "WAILS_TEST_TOOL"); err != nil || got != "" {
		t.Fatalf("unset: got %q, %v", got, err)
	}
	t.Setenv("WAILS_TEST_TOOL", "from-env")
	if got, _ := toolPath("", "WAILS_TEST_TOOL"); !filepath.IsAbs(got) || filepath.Base(got) != "from-env" {
		t.Fatalf("env: got %q", got)
	}
	if got, _ := toolPath("/x/flag", "WAILS_TEST_TOOL"); got != "/x/flag" {
		t.Fatalf("flag must win over env: got %q", got)
	}
}
