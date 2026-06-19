// Package version exposes the build-time version stamp. The value is
// injected by the Makefile / install.sh / GitHub Actions workflow via
//
//	-ldflags "-X github.com/dutraph/repofleet/internal/version.Version=<x.y.z>"
//
// reading the VERSION file at the repo root (or `git describe --tags`
// inside CI). When the binary is built without that flag (e.g. `go
// run`, or a bare `go build`) the value stays at "dev" so a `version`
// subcommand or on-screen banner never displays an empty string.
//
// Why a separate package instead of stamping main.version?
// `cmd/fleet/main.go` is the only file that lives in `package main`,
// and other packages (notably `internal/ui` for any on-screen banner)
// also want to print the version. Putting Version in its own importable
// package avoids the import cycle without exposing the rest of `main`
// to the world.
package version

// Version is the current build identifier (e.g. "1.0.0"). Override at
// link time with -X.
var Version = "dev"

// String returns Version prefixed with "v" so the canonical form is
// what's printed everywhere ("v1.0.0"). Falls back to plain "dev"
// when the build was unstamped, since "vdev" looks wrong.
func String() string {
	if Version == "" || Version == "dev" {
		return "dev"
	}
	return "v" + Version
}
