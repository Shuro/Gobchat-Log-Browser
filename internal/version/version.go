// Package version holds the application version, injected at build time via
// -ldflags "-X gobchat-log-browser/internal/version.Version=x.y.z".
package version

// Version is "dev" for local builds; release builds overwrite it from the git tag.
var Version = "dev"
