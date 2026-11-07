package e2e

import (
	"testing"

	misecheck "github.com/joeblew999/mise-check"
)

// TestRootMise is misecheck (github.com/joeblew999/mise-check) on this repo's own mise.toml, its mise side (the root
// runs Go, it isn't a binary's tasks), and the one shape of a mise tool repo: root:ci, what its ci.yml runs and
// nothing else there, root:verify, the README's opening, no template task wrapping mise-check (mise run
// root:check-mise).
func TestRootMise(t *testing.T) {
	misecheck.Check(t, misecheck.Config{MiseFile: "mise.toml"})
	misecheck.CheckToolRepo(t, ".")
}

// TestTemplateMise is misecheck on the template's tasks, both ways: against mise-seo's Go, against the walk that runs
// them, against the template's skill and docs page, which name every task and flag, and against the other files init
// and template write that name its tasks (mise run template:check-mise).
func TestTemplateMise(t *testing.T) {
	misecheck.Check(t, misecheck.Config{
		MiseFile:  "templates/.mise/tasks/mise-seo.toml",
		GoCmd:     "cmd",
		GoLayout:  "file",
		GoCallers: ".",
		Skills:    []string{"templates/.claude/skills/seo/SKILL.md", "templates/docs/seo/README.md"},
		Names: []string{
			"templates/.mise/config.toml",
			"templates/.claude/hooks/mise-seo-session-start.sh",
			"templates/.github/workflows/seo.yml",
			"templates/mise.toml",
			"templates/seo.toml",
			"templates/README.md",
			"templates/app/main.go",
		},
	})
}
