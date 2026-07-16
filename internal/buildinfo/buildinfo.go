// Package buildinfo exposes build identity and process start time.
//
// SHA and BuildTime are injected at build time via -ldflags -X (see Dockerfile).
// StartedAt is captured at process start for uptime reporting.
package buildinfo

import "time"

var (
	// SHA is the git commit the binary was built from. Overridden via ldflags.
	SHA = "dev"
	// BuildTime is the RFC3339 UTC time the binary was built. Overridden via ldflags.
	BuildTime = "unknown"
	// StartedAt is the process start time, used to compute uptime.
	StartedAt = time.Now()
)

// ShortSHA returns the first 12 characters of SHA, or the full value when shorter.
func ShortSHA() string {
	if len(SHA) > 12 {
		return SHA[:12]
	}
	return SHA
}
