// Package templates is mise-seo's template, embedded for mise-seo: a whole repo that runs where it is, in two halves.
// The tool's half is what mise-seo init writes into any repo, at the same paths: the numbered tasks and mise-seo's pin
// (.mise/), the GitHub workflow, the Claude Code hook and its settings, the agent skill, the docs and the .gitignore.
// The rest is the template's little app, which mise-seo template writes into a repo with no Go program yet: two pages
// in plain Go behind seo.Site (app/), their test auditing them as Googlebot reads them, the mise.toml of its tasks,
// and its README; seo.toml, the repo's own, has the app's site on this machine (AppSite). This file is the template's
// only Go at its top, as mise-intl's templates.go is. What is for mise-seo itself (its own mise.toml, tests, CI and
// README) is never here, and neither is the machine's own .mise/config.local.toml.
package templates

import (
	"embed"
	"io/fs"
)

//go:embed .mise/tasks/mise-seo.toml .mise/config.toml
//go:embed .github/workflows/seo.yml
//go:embed .claude/hooks/mise-seo-session-start.sh .claude/settings.json .claude/skills
//go:embed docs/seo
//go:embed .gitignore
var tool embed.FS

// Tool is the template's tool half: the files init writes into a repo, at their paths there.
var Tool fs.FS = tool

//go:embed app/main.go app/main_test.go mise.toml README.md
var app embed.FS

// App is the template's little app: the files mise-seo template writes into a repo with no Go program yet, at their
// paths there. Its mise.toml runs and tests it, and README.md is the repo's front page.
var App fs.FS = app

// AppName is the template's name in its README.md: a repo gets its own instead, its folder's.
const AppName = "mise-seo-template"

// AppSite is the template app's site on this machine, where mise run serve serves it: seo.toml's site in the
// template, and in a repo whose seo.toml sets none, once mise-seo template has written the app.
const AppSite = "http://localhost:8080"

// HookFile is the Claude Code hook's name in a repo's .claude/hooks.
const HookFile = "mise-seo-session-start.sh"
