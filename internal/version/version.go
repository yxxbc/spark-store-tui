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

// String is the single-line version reported by `sparkstore --version`.
func String() string {
	if Version == "dev" {
		return fmt.Sprintf("sparkstore dev (commit %s)", Commit)
	}
	return fmt.Sprintf("sparkstore %s (commit %s, built %s)", Version, Commit, Date)
}
