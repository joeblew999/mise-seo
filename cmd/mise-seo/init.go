package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joeblew999/mise-check/miseconfig"
	"github.com/joeblew999/mise-check/toolfiles"
	"github.com/joeblew999/mise-seo/seo/config"
	"github.com/joeblew999/mise-seo/templates"
)

// self is mise-seo as a repo's mise pins it, from its GitHub releases (.mise/config.toml, which init writes).
const self = "github:joeblew999/mise-seo"

// toolFiles is mise-seo as its init writes its files into a repo, by mise-check's toolfiles, as every tool's init
// does: the template's tool half (templates.Tool), the hook, and its parts of .mise/config.toml (miseconfig): its pin
// at this release (local, a checkout's build), its own name in minimum_release_age_excludes, use_versions_host =
// false, and its comment at the file's top, each as the template's .mise/config.toml has it.
func toolFiles() toolfiles.Tool {
	tmpl, err := fs.ReadFile(templates.Tool, ".mise/config.toml")
	must(err)
	release := ""
	if version != "" {
		release = buildVersion()
	}
	return toolfiles.Tool{Files: templates.Tool, Hook: templates.HookFile,
		Config: miseconfig.Tool{Name: self, Release: release, Template: tmpl, Init: initTask}}
}

// initRepo is mise-seo init (seo:init): the repo set up for mise-seo, as mise-intl's init sets one up for intl. First
// the template's tool half (toolFiles: the tasks, .mise/config.toml pinning this build, the GitHub workflow, the Claude
// Code hook and skill, the docs, the .gitignore), each at its path, and of the repo's own files only mise-seo's parts;
// then mise-check's own init, so the repo has mise-check's tasks itself (toolfiles.MiseCheckInit, at the release
// mise-seo is built with); then seo.toml, the repo's own, when it has none or sets no site (--site); then, in a Go
// module, mise-seo's packages required in its go.mod at this release. Every step leaves alone what is the repo's, so
// it runs again after upgrading.
func (r repo) initRepo(args []string) {
	f := flags("init")
	site := f.String("site", "", "the live site, https://app.example.com: seo.toml's site, when it sets none")
	parse(f, args, 0)
	must(toolFiles().Write(r.dir))
	toolfiles.Trust(r.dir, ".mise/config.toml", nil)
	must(toolfiles.MiseCheckInit(r.dir))
	r.seoToml(*site)
	r.pinLibrary()
	sayNewest()
	fmt.Println("✓ next: mise run seo: what's done, and what to run next")
}

// stale are what init would write again: toolFiles' problems, each naming seo:init.
func (r repo) stale() []string { return toolFiles().Problems(r.dir) }

// older reports whether release a is older than release b, both x.y.z.
func older(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		x, errx := strconv.Atoi(pa[i])
		y, erry := strconv.Atoi(pb[i])
		if errx != nil || erry != nil {
			return false
		}
		if x != y {
			return x < y
		}
	}
	return len(pa) < len(pb)
}

// newestRelease is mise-seo's newest release, as GitHub's page for it names it (…/releases/latest redirects to its
// tag), or "" if GitHub can't say.
func newestRelease() string {
	c := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Get("https://github.com/joeblew999/mise-seo/releases/latest")
	if err != nil {
		return ""
	}
	resp.Body.Close()
	_, tag, _ := strings.Cut(resp.Header.Get("Location"), "/releases/tag/")
	return strings.TrimPrefix(tag, "v")
}

// sayNewest says, of a release, when GitHub has a newer one, and the one line that sets the repo up at it, on the same
// line: @latest runs the release mise lists as its newest, and mise lists one some minutes after it's out (its
// versions host, and GitHub's own API, which caches a minute or more). An exact release mise gets at once, and a
// latest pin then runs it, the newest installed.
func sayNewest() {
	if version == "" {
		return
	}
	switch n := newestRelease(); {
	case n == "":
		fmt.Printf("· this is mise-seo %s; GitHub didn't say what its newest release is\n", buildVersion())
	case older(buildVersion(), n):
		fmt.Printf("· this is mise-seo %s, and its newest release is %s (mise lists one as latest some minutes after it's out). To set the repo up at %s: %s\n", buildVersion(), n, n, initAt(n))
	default:
		fmt.Printf("✓ mise-seo %s, the newest release\n", buildVersion())
	}
}

// initAt is the one line that sets a repo up at a release of mise-seo (0.4.30), whatever runs now: mise gets an exact
// release at once, as no list of releases is read for it; init then writes its files as that release does and moves
// an older pin to it, and a pin of latest runs it, the newest installed.
func initAt(release string) string {
	return "mise exec " + self + "@" + release + " -- mise-seo init"
}

func writeFile(path string, b []byte) {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, b) {
		fmt.Printf("✓ %s is up to date\n", path)
		return
	}
	must(os.MkdirAll(filepath.Dir(path), 0o755))
	must(os.WriteFile(path, b, 0o644))
	fmt.Println("wrote", path)
}

// seoToml writes seo.toml, the repo's own, when it has none: every setting at its default (config.Template), with
// site, init's --site, set in it. In a seo.toml of the repo's own, init sets the site only when it sets none; it
// says so when there's still none, as every task works on it.
func (r repo) seoToml(site string) {
	path := filepath.Join(r.dir, config.File)
	b, err := os.ReadFile(path)
	text := config.Template
	if err == nil {
		text = string(b)
	}
	if site != "" {
		if t, ok := config.SetSite(text, site); ok {
			text = t
		} else {
			fmt.Printf("✓ %s sets a site of its own: left as it is (edit it there)\n", config.File)
		}
	}
	if err != nil || text != string(b) {
		writeFile(path, []byte(text))
	} else {
		fmt.Printf("✓ %s is the repo's own: left as it is\n", config.File)
	}
	if c, err := config.Find(r.dir); err == nil && c.Site == "" {
		fmt.Printf("· %s has no site, which every task works on: mise run seo:init --site https://your.site (the live one; mise run seo:template sets the template app's, on this machine)\n", config.File)
	}
}

// pinLibrary requires this release of mise-seo's packages in the repo's go.mod (go get …@vX.Y.Z, with the Go mise
// gives the repo), so the packages the app imports are the release its mise-seo is: run by hand at a newer release,
// init is the move to it. A repo with no go.mod yet gets it from mise-seo template; a go.mod that requires this
// release or a newer one, or replaces the module with a folder, stays as it is, and so does every go.mod when this is
// a checkout's build.
func (r repo) pinLibrary() {
	gomod := filepath.Join(r.dir, "go.mod")
	b, err := os.ReadFile(gomod)
	if err != nil || version == "" {
		return
	}
	text := string(b)
	at := buildVersion()
	if pinned := requiredVersion(text, selfModule); strings.Contains(text, selfModule+" =>") || pinned != "" && !older(strings.TrimPrefix(pinned, "v"), at) {
		return
	}
	cmd := goCommand(r.dir, "get", selfModule+"@v"+at)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	must(cmd.Run())
	fmt.Printf("✓ go.mod requires mise-seo's packages at %s, mise-seo's release\n", at)
}

// selfModule is mise-seo's Go module: an app's go.mod requires it for the packages it imports (seo, gsxseo, seo/audit).
const selfModule = "github.com/joeblew999/mise-seo"

// goCommand is go with args in dir: through mise when it's there (mise exec, so it's the Go the repo's mise gives it,
// go.mod's), else the go on the PATH.
func goCommand(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("go", args...)
	if _, err := exec.LookPath("mise"); err == nil {
		cmd = exec.Command("mise", append([]string{"exec", "--", "go"}, args...)...)
	}
	cmd.Dir = dir
	return cmd
}

// requiredVersion is the version a go.mod's text requires of module, "" if it doesn't.
func requiredVersion(text, module string) string {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		f := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "require "))
		if len(f) >= 2 && f[0] == module && strings.HasPrefix(f[1], "v") {
			return f[1]
		}
	}
	return ""
}
