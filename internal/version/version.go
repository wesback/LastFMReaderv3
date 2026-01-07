package version

import "fmt"

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func String() string {
	return fmt.Sprintf("lastfm-sync %s (built %s, commit %s)", Version, BuildTime, GitCommit)
}
