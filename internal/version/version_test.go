package version

import (
	"strings"
	"testing"
)

func TestUserAgentFollowsVersion(t *testing.T) {
	original := Version
	t.Cleanup(func() { Version = original })

	Version = "0.9.1"
	if got := UserAgent(); got != "sparkstore/0.9.1" {
		t.Fatalf("user agent = %q", got)
	}
	Version = "dev"
	if got := UserAgent(); got != "sparkstore/dev" {
		t.Fatalf("dev user agent = %q", got)
	}
}

func TestStringReportsInjectedVersion(t *testing.T) {
	originalVersion, originalCommit, originalDate := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = originalVersion, originalCommit, originalDate })

	Version, Commit, Date = "0.9.0", "abc1234", "2026-10-31T00:00:00Z"
	got := String()
	if !strings.Contains(got, "0.9.0") || !strings.Contains(got, "abc1234") {
		t.Fatalf("version = %q", got)
	}
}
