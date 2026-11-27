package version

import "fmt"

// These values are replaced at build time:
//
//	go build -ldflags "-X .../internal/version.Version=0.9.0 ..."
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// UserAgent identifies the client to catalog and image hosts. It follows the
// build version instead of a hard-coded release number.
func UserAgent() string {
	if Version == "dev" {
		return "sparkstore/dev"
	}
	return "sparkstore/" + Version
}

// String is the single-line version reported by `sparkstore --version`.
func String() string {
	if Version == "dev" {
		return fmt.Sprintf("sparkstore dev (commit %s)", Commit)
	}
	return fmt.Sprintf("sparkstore %s (commit %s, built %s)", Version, Commit, Date)
}
