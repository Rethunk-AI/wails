//go:build linux && !android && !server

package application

import "testing"

func TestCaseGlobs(t *testing.T) {
	for in, want := range map[string]string{
		"*.zip":           "*.[zZ][iI][pP]",
		"*.7z;*.RAR":      "*.7[zZ];*.[rR][aA][rR]",
		"*":               "*",
		"mod_?.json":      "[mM][oO][dD]_?.[jJ][sS][oO][nN]",
		"*.[jJ]pg":        "*.[jJ][pP][gG]",
		"*.tar.gz; *.tgz": "*.[tT][aA][rR].[gG][zZ]; *.[tT][gG][zZ]",
		"":                "",
	} {
		if got := caseGlobs(in); got != want {
			t.Errorf("caseGlobs(%q) = %q, want %q", in, got, want)
		}
	}
}
