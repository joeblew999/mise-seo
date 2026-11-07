package main

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/joeblew999/mise-seo/seo/config"
	"github.com/joeblew999/mise-seo/templates"
)

// TestShow: show's next step, as a repo meets them: init's files missing; then init done and no Go program, the
// template next; then a Go program and no site, init --site; then the audit. The template's own seo.toml is init's
// with the template app's site.
func TestShow(t *testing.T) {
	r := repo{dir: t.TempDir()}
	if s := r.where(); s.Next != initTask {
		t.Fatalf("no init's files: next %q, want %s (problems %q)", s.Next, initTask, s.Problems)
	}
	r.initRepo(nil)
	if s := r.where(); s.Next != "mise run seo:template" {
		t.Fatalf("init's files, no Go program: next %q, want mise run seo:template (problems %q)", s.Next, s.Problems)
	}
	must(os.WriteFile(filepath.Join(r.dir, "go.mod"), []byte("module example.com/app\n"), 0o644))
	if s := r.where(); s.Next != initTask+" --site https://your.site" || len(s.Problems) != 1 {
		t.Fatalf("a go.mod, no site: next %q, want init --site, for one problem: %q", s.Next, s.Problems)
	}
	r.initRepo([]string{"--site", "https://shop.example.com/"})
	if s := r.where(); s.Next != "mise run seo:06-audit" || s.Site != "https://shop.example.com" || len(s.Problems) > 0 {
		t.Fatalf("init --site: next %q, site %q, problems %q; want the audit next, on the site", s.Next, s.Site, s.Problems)
	}
	b, _ := os.ReadFile("../../templates/seo.toml")
	if want, _ := config.SetSite(config.Template, templates.AppSite); string(b) != want {
		t.Errorf("templates/seo.toml isn't init's seo.toml with the template app's site, %s:\n%s", templates.AppSite, b)
	}
}

// TestOwnFiles: init writes mise-seo's parts of the repo's own files as this version has them, and nothing else of
// them: its pin and its own name beside the repo's in minimum_release_age_excludes (miseconfig), the .gitignore lines,
// the hook in a .claude/settings.json with a hook of its own; check says to run init until it has.
func TestOwnFiles(t *testing.T) {
	r := repo{dir: t.TempDir()}
	r.initRepo([]string{"--site", "https://shop.example.com"})
	at := func(p string) string { return filepath.Join(r.dir, filepath.FromSlash(p)) }
	write := func(p, s string) { must(os.WriteFile(at(p), []byte(s), 0o644)) }
	read := func(p string) string { b, _ := os.ReadFile(at(p)); return string(b) }
	write(".mise/config.toml", "# the repo's own\n\n[settings]\nminimum_release_age_excludes = [\"npm:*\"]\n\n[tools]\nnode = \"26\"\n")
	write(".gitignore", "bin/\n")
	write(".claude/settings.json", `{"model": "opus", "hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "own.sh"}]}]}}`)
	if s := r.where(); s.Next != initTask || len(s.Problems) != 5 {
		t.Fatalf("next %q, want %s, for 5 problems: %q", s.Next, initTask, s.Problems)
	}
	r.initRepo(nil)
	if s := r.where(); len(s.Problems) > 0 {
		t.Fatalf("after init: %q", s.Problems)
	}
	cfg, ignore, settings := read(".mise/config.toml"), read(".gitignore"), read(".claude/settings.json")
	for _, want := range []string{"# the repo's own\n", `minimum_release_age_excludes = ["npm:*", "github:joeblew999/mise-seo"]`, "\nuse_versions_host = false\n", `"github:joeblew999/mise-seo" = "local"`, `node = "26"`} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config.toml has no %q:\n%s", want, cfg)
		}
	}
	if !strings.HasPrefix(ignore, "bin/\n") || !strings.Contains(ignore, ".mise/config.local.toml") {
		t.Errorf(".gitignore:\n%s", ignore)
	}
	if !strings.Contains(settings, `"opus"`) || !strings.Contains(settings, "own.sh") || !strings.Contains(settings, templates.HookFile) {
		t.Errorf("settings.json:\n%s", settings)
	}
}

// TestParse: a command's flags before or after its argument, as a person types them and as the tasks pass them.
func TestParse(t *testing.T) {
	for _, args := range [][]string{{"/de/", "--json", "--open"}, {"--json", "/de/", "--open"}, {"--json", "--open", "/de/"}} {
		f := flag.NewFlagSet("inspect", flag.ContinueOnError)
		jsonOut, open := f.Bool("json", false, ""), f.Bool("open", false, "")
		if got := parse(f, args, 1); !slices.Equal(got, []string{"/de/"}) || !*jsonOut || !*open {
			t.Errorf("%q: arguments %q, --json %v, --open %v", args, got, *jsonOut, *open)
		}
	}
}

// TestDomainProperty: a site's Search Console Domain property from its host; none for an IP address or localhost.
func TestDomainProperty(t *testing.T) {
	for host, want := range map[string]string{
		"go-htmx4.ubuntusoftware.net": "sc-domain:ubuntusoftware.net",
		"example.com":                 "sc-domain:example.com",
		"127.0.0.1":                   "",
		"::1":                         "",
		"localhost":                   "",
	} {
		if got := domainProperty(host); got != want {
			t.Errorf("domainProperty(%q) = %q, want %q", host, got, want)
		}
	}
	if got := defaultProject("https://go-htmx4.ubuntusoftware.net"); got != "go-htmx4-search" {
		t.Errorf("defaultProject = %q, want go-htmx4-search", got)
	}
}
