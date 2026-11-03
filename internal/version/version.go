// Package version reports the conjecture binary's build version.
package version

// These are overridden at build time via -ldflags, e.g.:
//
//	go build -ldflags "-X github.com/farlleyferreira/conjecture-ai/internal/version.version=1.2.3"
//
// Left at their defaults, a locally built binary reports "dev".
var (
	version = "dev"
	commit  = "none"
)

// String returns the human-readable version string, e.g. "1.2.3 (commit abc1234)".
func String() string {
	return version + " (commit " + commit + ")"
}
