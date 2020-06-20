package version

import (
	"fmt"
)

var (
	// RELEASE returns the release version
	RELEASE = "UNKNOWN"
	// REPO returns the git repository URL
	REPO = "UNKNOWN"
	// COMMIT returns the short sha from git
	COMMIT = "UNKNOWN"
)

// showVersion returns information about the release.
func ShowVersion() string {
	return fmt.Sprintf(`
	-----------------------------------------------------------------------
	EKS DNS Troubleshooter
	Release:    %v
	Build:      %v
	Repository: %v
	-----------------------------------------------------------------------
	`, RELEASE, COMMIT, REPO)
}
