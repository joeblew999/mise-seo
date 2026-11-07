// The steps: the repo's numbered tasks, in their order, each with its check (the walk tests walk them).

package e2e

import (
	"cmp"
	"strings"

	"github.com/joeblew999/mise-check/walk"
)

// steps are the repo's numbered tasks (its .mise/tasks/mise-seo.toml, which is the template's), in their order, each
// with its check: how the walk runs it and checks it did what it says, as a person would see it. TestSteps fails
// unless they're exactly the template's numbered tasks, in their order. The tasks that ask Google run as they do for a
// person with no key yet: each says which it needs and how to get it, and fails.
var steps = []walk.Step[*walker]{
	{Task: "seo:03-init", Check: func(w *walker) {
		before := w.Snapshot()
		w.MustTask("seo:03-init")
		if changed := walk.Diff(before, w.Snapshot()); len(changed) > 0 {
			w.T.Fatalf("seo:03-init changed %s: the repo isn't what init writes", strings.Join(changed, ", "))
		}
		w.Pass("seo:03-init changed nothing: the repo is what init writes")
	}},
	{Task: "seo:04-check", Check: func(w *walker) {
		if out := w.MustTask("seo:04-check"); !strings.Contains(out, "✓ rules: pass") {
			w.T.Fatalf("seo:04-check: %s", out)
		}
		w.Pass("seo:04-check: the rules pass")
		if out := w.MustTask("seo:verify"); !strings.Contains(out, "ok") || !strings.Contains(out, "✓ rules: pass") {
			w.T.Fatalf("seo:verify: %s", out)
		}
		w.Pass("seo:verify: what the repo's GitHub workflow runs passes (its setup, its test auditing the app, then check)")
	}},
	{Task: "seo:05-show", Check: func(w *walker) {
		var s struct {
			Version, Site, Next string
			App                 bool
		}
		w.TaskJSON(&s, "seo:05-show", "--json")
		if !s.App || s.Site != "http://localhost:8080" || s.Next != "mise run seo:06-audit" {
			w.T.Fatalf("seo:05-show --json: %+v", s)
		}
		w.Pass("seo:05-show: the app, its site %s, and next: %s", s.Site, s.Next)
		// The version is the release .mise/config.toml pins (-version), or this checkout's build (local).
		if want := strings.TrimPrefix(*walk.Version, "v"); want != "" && s.Version != want || s.Version == "" {
			w.T.Fatalf("seo:05-show's version is %q, not %q: the one .mise/config.toml pins", s.Version, cmp.Or(want, "this checkout's build"))
		}
		w.Pass("seo:05-show's version is the one .mise/config.toml pins: %s", s.Version)
	}},
	{Task: "seo:06-audit", Check: func(w *walker) {
		addr, stop := w.serve()
		defer stop()
		var a struct {
			Site     string
			URLs     int
			OK       bool
			Problems []string
		}
		w.TaskJSON(&a, "seo:06-audit", "--json", "--site", addr)
		if !a.OK || a.URLs != 2 || a.Site != addr {
			w.T.Fatalf("seo:06-audit --json --site %s: %+v", addr, a)
		}
		w.Pass("seo:06-audit --site %s: the app, run as a person runs it (mise run serve), read as Googlebot reads it: robots.txt, llms.txt and its %d pages pass", addr, a.URLs)
	}},
	{Task: "seo:07-lighthouse", Check: func(w *walker) { w.needsKey("seo:07-lighthouse", "PAGESPEED_API_KEY") }},
	{Task: "seo:08-badges", Check: func(w *walker) { w.needsKey("seo:08-badges", "PAGESPEED_API_KEY") }},
	{Task: "seo:09-setup", Skip: "it makes a Google Cloud project, its APIs and a service account with a person's own gcloud login, and stores the key in their fnox: the walk checks instead that each task that asks Google says what it needs"},
	{Task: "seo:10-sites", Check: func(w *walker) { w.needsKey("seo:10-sites", "GOOGLE_SEARCH_CONSOLE_KEY", "seo:09-setup") }},
	{Task: "seo:11-submit", Check: func(w *walker) { w.needsKey("seo:11-submit", "GOOGLE_SEARCH_CONSOLE_KEY", "seo:09-setup") }},
	{Task: "seo:12-status", Check: func(w *walker) { w.needsKey("seo:12-status", "GOOGLE_SEARCH_CONSOLE_KEY", "seo:09-setup") }},
	{Task: "seo:13-todo", Check: func(w *walker) { w.needsKey("seo:13-todo", "GOOGLE_SEARCH_CONSOLE_KEY", "seo:09-setup") }},
	{Task: "seo:14-inspect", Check: func(w *walker) { w.needsKey("seo:14-inspect", "GOOGLE_SEARCH_CONSOLE_KEY", "seo:09-setup") }},
}

// needsKey runs a task that asks Google as a person with no key yet does (none in the environment, and fnox not
// asked: each set empty), and checks it fails, saying which site it works on and each of says: the key, and how to
// get it.
func (w *walker) needsKey(task string, says ...string) {
	w.T.Helper()
	out, err := w.Task([]string{"GOOGLE_SEARCH_CONSOLE_KEY=", "PAGESPEED_API_KEY="}, "", task)
	if err == nil {
		w.T.Fatalf("%s passed with no key", task)
	}
	for _, s := range append([]string{"mise-seo: http://localhost:8080 (from "}, says...) {
		if !strings.Contains(out, s) {
			w.T.Fatalf("%s with no key didn't say %q:\n%s", task, s, out)
		}
	}
	w.Pass("%s with no key: it says the site, and that it needs %s", task, strings.Join(says, ", made by "))
}
