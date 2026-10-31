package version

import (
	"strings"
	"testing"
)

func TestStringReportsInjectedVersion(t *testing.T) {
	originalVersion, originalCommit, originalDate := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = originalVersion, originalCommit, originalDate })

	Version, Commit, Date = "0.9.0", "abc1234", "2026-10-31T00:00:00Z"
	got := String()
	if !strings.Contains(got, "0.9.0") || !strings.Contains(got, "abc1234") {
		t.Fatalf("version = %q", got)
	}
}
