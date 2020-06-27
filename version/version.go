package version

import (
	"fmt"
)

//DiagToolInfo stores information about diagnosis tool
type DiagToolInfo struct {
	Release string `json:"release"`
	Repo    string `json:"repo"`
	Commit  string `json:"commit"`
}

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
