package version

import "strings"

// Version is the squad-oc release string. Releases overwrite this via
// -ldflags -X. Local go build leaves the dev default.
var Version = "dev"

// Name is the binary and archive stem (goreleaser project_name).
const Name = "squad-oc"

// Module is the Go module path. Must match go.mod (enforced by test).
const Module = "github.com/xeaser/squad-opencode"

// Repo is owner/name used by update-check and upgrade --self.
// Default follows Module. Releases may overwrite via -ldflags -X with
// GITHUB_REPOSITORY so forks self-update from themselves.
var Repo = repoFromModule(Module)

func repoFromModule(module string) string {
	return strings.TrimPrefix(module, "github.com/")
}
