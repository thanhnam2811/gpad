// Package buildinfo is the single source for application release identity.
package buildinfo

import "fmt"

// Linker overrides are set by scripts/release.sh; development builds remain explicit.
var (
	Version   = "0.1.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func String() string {
	return fmt.Sprintf("Gopad++ %s (commit %s, built %s)", Version, Commit, BuildDate)
}
