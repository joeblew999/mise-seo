# mise-seo

[![ci](https://github.com/joeblew999/mise-seo/actions/workflows/ci.yml/badge.svg)](https://github.com/joeblew999/mise-seo/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/joeblew999/mise-seo)](https://github.com/joeblew999/mise-seo/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/joeblew999/mise-seo.svg)](https://pkg.go.dev/github.com/joeblew999/mise-seo)

**The problem.** Getting a Go web app into Google properly takes a lot of small pieces, and each repo ends up doing
them a little differently, or not at all:

- what crawlers ask for, answered at the site's root: robots.txt, a sitemap of every page in every language, llms.txt
  for agents; everything that isn't a page marked noindex; one https address for every page, other hosts and plain
  http moved there;
- every page's head as Google and link previews read it: a title and a description that fit Google's widths, a
  canonical URL, reciprocal hreflang, Open Graph, the icon, the site's name on the home page;
- knowing it's all right before a deploy and after it, not weeks later from Search Console: the live site fetched as
  Googlebot fetches it, Lighthouse's four scores with the elements that cost them, Google's own verdict on each URL;
- the keys Google wants (a PageSpeed Insights key, a Search Console service account) made once and kept out of the
  repo, and all of it the same on every machine, in CI and in a Claude Code cloud session.

**What mise-seo does.** It does all of that through [mise](https://mise.jdx.dev), for any Go web app, with
[gsx](https://github.com/gsxhq/gsx) or without: `seo.Site`, the middleware that answers crawlers; `gsxseo.Head`, the
head, in a gsx app; `audit.Audit`, which an app's own tests run on its local build; and the `mise-seo` binary, whose
commands are numbered mise tasks, from `seo:03-init` to `seo:14-inspect`, the live audit, Lighthouse on Google's
servers (no Node, no Chrome), the README's badges and Search Console among them. One command adds it to a repo, and
`mise run seo` shows where the repo is at any time. Its keys come from fnox, its pin from `.mise/config.toml`, and its
workflow checks the repo on every push. In a repo, at its root:

```sh
mise exec "github:joeblew999/mise-seo[minimum_release_age=0s]@latest" -- mise-seo init --site https://app.example.com
mise run seo
```

Everything a repo gets, and how to use it, is [templates/docs/seo/README.md](templates/docs/seo/README.md), which
`init` writes into the repo as `docs/seo/README.md`. The rest of this README is for working on mise-seo itself.

## Working on mise-seo

This repo is two things, kept apart:

| | What | Where |
|---|---|---|
| **For other repos** | The `mise-seo` binary, the packages an app imports, and the template: a whole repo you can drop anywhere and run, whose tool half `init` writes into a repo (its tasks and their mise config, its workflow, its Claude Code hook and skill, its docs, its `.gitignore`) and whose little app `seo:template` writes into a repo with none | [templates/](templates/), [cmd/mise-seo/](cmd/mise-seo/), [seo/](seo/), [gsxseo/](gsxseo/), [internal/](internal/) |
| **For this repo** | The tasks for working on it, its tests, the release, its CI. This README | [mise.toml](mise.toml), the `*_test.go` at the top, [.github/workflows/](.github/workflows/) |

Nothing for this repo is ever in `templates/`, and nothing in `templates/` is this repo's own. `cmd/mise-seo` is the
one entry point that ships, with the packages; everything this repo runs for itself in Go is a test at the top
(package `e2e`), which the root's tasks run.

### Rules

- The packages never import gsx, but `gsxseo`: an app without gsx gets none (`TestWithoutGsx`).
- Every tool is pinned and comes from mise: mise-seo in a repo's `.mise/config.toml`, fnox and gcloud in the tasks
  that run them (their `tools`), Go in `go.mod`. Lighthouse runs on Google's servers (PageSpeed Insights), so there's
  no Node and no Chrome.
- Every task is one line, with its flags passed by name, and no POSIX shell syntax, so it runs in Windows' cmd too:
  logic is Go, a file per command (`x.go` holding `x`, `main.go`'s `case "x"`), and `commands` in `main.go` is the
  one source of each task's line, which the tasks file, the skill and the docs page have word for word
  (`TestCommands`).
- Test the real thing: the template's app served by its own task and audited over HTTP, its own test auditing it,
  each task run through the repo's own mise.

### Tasks

Every task is named noun:verb, and runs what it needs first: run the furthest you need. `mise tasks` lists them.

| Task | What | When |
|---|---|---|
| `root:ci` | What CI runs: `tool:test`, `root:check-mise`, `template:check-mise`, then both walks | Before every push |
| `tool:test` | `go vet`, the Go tests (none walk), `goreleaser check` | After every change |
| `tool:build` | This checkout's build, `bin/mise-seo` | Any task needs it |
| `tool:install` | That build as the mise version `local`, for the template (`templates/.mise/config.local.toml`, never committed) and any repo on this machine (`mise use github:joeblew999/mise-seo@local` there) | To try a change in another repo, never by waiting on a release |
| `tool:release` | A release from the commit checked out, as `release.yml` runs it; `--dry-run` tags and publishes nothing | Never by hand: a merge to main runs it |
| `template:check-mise` | `TestTemplateMise`: misecheck on the template's tasks against the binary's Go, the walk, the skill and the docs page, both ways | After a change to a task or a command |
| `template:e2e` | `TestWalkTemplate`: the walk on the template, where it is | After every change |
| `new-repo:e2e` | `TestWalkNewRepo`: the walk on a new repo (`--version` pins a release) | After every change |
| `root:check-mise` | `TestRootMise` on this repo's own `mise.toml` | After a change to this file's tasks |

mise-check is used here as in any repo, through its own tasks, which `mise-check init` wrote: `mise run mise-check`
(what's done, what's next), `mise-check:check`, `mise-check:verify`, `mise-check:update` (every pin, mise-check's
binary and packages at one release). [docs/mise-check/README.md](docs/mise-check/README.md) has all of it.

### Tests

**The walk.** A repo's steps are its numbered tasks, `seo:03-init` to `seo:14-inspect`
([templates/.mise/tasks/mise-seo.toml](templates/.mise/tasks/mise-seo.toml)), after steps 00 to 02 (the repo's git,
`mise install`, its own `setup`). `TestSteps` holds the walk's steps to them, in order. The walk runs each through the
repo's own mise file, as a person does, each with its check ([e2e_steps_test.go](e2e_steps_test.go)): `init` writes
exactly what's there; `check` and the workflow's `verify` pass; `show` says what's next; the app, served by its own
`serve` task, passes the audit as Googlebot reads it; and each task that asks Google, run with no key, says the site
and the key it needs. `seo:09-setup` needs a person's own gcloud login, so the walk says so instead. It walks two
repos, each on its own mise only (`MISE_CEILING_PATHS`), on [mise-check's `walk`](https://github.com/joeblew999/mise-check):

- **The template**, [templates/](templates/), in place.
- **A new repo**, made as a person makes one: an empty folder with its own git, `mise-seo init`, then `seo:template`,
  in a folder gone when the walk ends. At this checkout's build (`local`, its `go.mod` using this checkout's packages),
  or at a release with `--version`, as the release tries it.

**The checker**, [mise_test.go](mise_test.go): misecheck on the template's tasks file and on this repo's own.

### Layout

| Path | What |
|---|---|
| [seo/](seo/) | package `seo` (`Site`, `Sitemap`, `LLMsTxt`, `Origin`, `SnippetWidth`), `seo/audit` (the site as Googlebot reads it), `seo/config` (`seo.toml`) |
| [gsxseo/](gsxseo/) | `Head`, the only package that imports gsx; its `head.x.go` is generated (`go -C gsxseo tool gsx generate -q .`) and committed |
| [internal/](internal/) | PageSpeed Insights (`pagespeed`), the Search Console API (`searchconsole`), the badges (`badge`) |
| [cmd/mise-seo/](cmd/mise-seo/) | the command: a file per command |
| [templates/](templates/) | the template; [templates.go](templates/templates.go) lists both halves |
| [e2e_walk_test.go](e2e_walk_test.go), [e2e_steps_test.go](e2e_steps_test.go) | the walk and its steps |
| [e2e_tool_test.go](e2e_tool_test.go) | `TestToolRelease`: only `mise run tool:release` runs it |
| [mise_test.go](mise_test.go), [gsx_test.go](gsx_test.go) | misecheck on the mise files; the packages without gsx |

### CI and releases

Shell steps only, no JavaScript actions. Each workflow checks out, installs mise, then runs the root's tasks:

- [ci.yml](.github/workflows/ci.yml), on every push to main and every pull request, on Linux, macOS and Windows:
  `root:ci`.
- [release.yml](.github/workflows/release.yml), on every change to the tool merged to main: `tool:release`. It takes
  the commit's `v*` tag or tags the next patch version, [goreleaser](.goreleaser.yaml) puts the binary for macOS,
  Linux and Windows on the release, a new repo pinning it is walked, and the version goes on proxy.golang.org, so
  `go get` has the packages at once. So the release is always main.
- [new-repo.yml](.github/workflows/new-repo.yml), once a release run has passed (or by hand, for any release): a new
  repo made at that release, walked on Linux, macOS and Windows, its tasks in each one's own shell.

mise-seo came from [go-htmx4](https://github.com/joeblew999/go-htmx4)'s `seo/`, which uses it. CLAUDE.md only loads
this README.

## License

MIT
