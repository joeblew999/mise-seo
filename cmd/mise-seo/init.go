package main

import (
	"bytes"
	"encoding/json"
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
	"github.com/joeblew999/mise-seo/seo/config"
	"github.com/joeblew999/mise-seo/templates"
)

// theirs are the template's files a repo may have of its own already: init writes them only when it has none, and
// then only its parts of them (keepOwn). .mise/config.toml is mise-seo's pin: written once, then the repo's (a newer
// release's init, run by hand, moves it forward).
var theirs = map[string]bool{".gitignore": true, ".claude/settings.json": true, ".mise/config.toml": true}

// self is mise-seo as a repo's mise pins it, from its GitHub releases (.mise/config.toml, which init writes).
const self = "github:joeblew999/mise-seo"

// configTool is mise-seo's parts of a repo's .mise/config.toml, kept by mise-check's miseconfig as every tool's init
// keeps its own: its pin (this release, or local for a checkout's build), its own name in
// minimum_release_age_excludes, use_versions_host = false, and its comment at the top, each as the template's
// .mise/config.toml has it.
func configTool() miseconfig.Tool {
	tmpl, err := fs.ReadFile(templates.Tool, ".mise/config.toml")
	must(err)
	release := ""
	if version != "" {
		release = buildVersion()
	}
	return miseconfig.Tool{Name: self, Release: release, Template: tmpl, Init: initTask}
}

// initRepo is mise-seo init (seo:03-init): the repo set up for mise-seo, as mise-intl's init sets one up for intl. First
// the template's tool files (templates.Tool: the numbered tasks, and .mise/config.toml pinning this build, the release
// that ran it or local; the GitHub workflow, the Claude Code hook and skill, the docs, the .gitignore), each at its
// path, and of the repo's own files only mise-seo's parts; then seo.toml, the repo's own, when it has none or sets no
// site (--site); then, in a Go module, mise-seo's packages required in its go.mod at this release. Every step leaves
// alone what is the repo's, so it runs again after upgrading.
func (r repo) initRepo(args []string) {
	f := flags("init")
	site := f.String("site", "", "the live site, https://app.example.com: seo.toml's site, when it sets none")
	parse(f, args, 0)
	r.writeTool()
	trustConfig(r.dir, ".mise/config.toml")
	r.seoToml(*site)
	r.pinLibrary()
	sayNewest()
	fmt.Println("✓ next: mise run seo: what's done, and what to run next")
}

// writeTool writes the template's tool files into the repo, each at its path there, and of a file of the repo's own
// (theirs) only mise-seo's parts.
func (r repo) writeTool() {
	must(fs.WalkDir(templates.Tool, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		to := filepath.Join(r.dir, filepath.FromSlash(p))
		if b, err := os.ReadFile(to); err == nil && theirs[p] {
			if !keepOwn(p, to, b) {
				fmt.Printf("✓ %s is the repo's own, with mise-seo's parts as this version writes them\n", p)
			}
			return nil
		}
		b, err := fs.ReadFile(templates.Tool, p)
		if err == nil {
			if p == ".mise/config.toml" {
				b, err = configTool().Pinned()
				must(err)
			}
			writeFile(to, b)
		}
		if strings.HasSuffix(p, ".sh") {
			must(os.Chmod(to, 0o755))
		}
		return err
	}))
}

// keepOwn brings mise-seo's parts of one of the repo's own files (theirs), at to, its text b, to what this version
// writes, and leaves the rest of it as it is: in .mise/config.toml its pin, its own exclude, use_versions_host and its
// comment (configTool); in .gitignore the template's lines it lacks; in .claude/settings.json the hook's SessionStart
// entry. It says what it wrote, and reports whether it did.
func keepOwn(p, to string, b []byte) bool {
	switch p {
	case ".mise/config.toml":
		out, wrote := configTool().Keep(b)
		if len(wrote) == 0 {
			return false
		}
		must(os.WriteFile(to, out, 0o644))
		for _, w := range wrote {
			fmt.Printf("wrote %s: %s\n", to, w)
		}
		return true
	case ".gitignore":
		return addIgnores(to, b)
	case ".claude/settings.json":
		return addHook(to, b)
	}
	return false
}

// ignores are the template's .gitignore lines that a repo's own .gitignore (b) lacks, each with the template's comment
// just above it.
func ignoresMissing(b []byte) []string {
	want, _ := fs.ReadFile(templates.Tool, ".gitignore")
	have := map[string]bool{}
	for _, l := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		have[strings.TrimSpace(l)] = true
	}
	var out []string
	lines := strings.Split(string(want), "\n")
	for i, l := range lines {
		if l = strings.TrimSpace(l); l == "" || strings.HasPrefix(l, "#") || have[l] {
			continue
		}
		if i > 0 && strings.HasPrefix(lines[i-1], "#") {
			l = lines[i-1] + "\n" + l
		}
		out = append(out, l)
	}
	return out
}

// addIgnores adds the template's .gitignore lines the repo's own lacks, at its end. The rest stays as it is.
func addIgnores(path string, b []byte) bool {
	missing := ignoresMissing(b)
	if len(missing) == 0 {
		return false
	}
	s := strings.TrimRight(string(b), "\n") + "\n" + strings.Join(missing, "\n") + "\n"
	must(os.WriteFile(path, []byte(s), 0o644))
	fmt.Printf("wrote %s: mise-seo's lines it lacked, %d\n", path, len(missing))
	return true
}

// addHook adds the hook's SessionStart entry, as the template's .claude/settings.json has it, to the repo's own (b)
// when it doesn't run the hook, keeping every other setting and hook. One it can't read as JSON is left as it is, and
// check goes on saying so.
func addHook(path string, b []byte) bool {
	if bytes.Contains(b, []byte(templates.HookFile)) {
		return false
	}
	var own, want map[string]any
	tb, _ := fs.ReadFile(templates.Tool, ".claude/settings.json")
	if json.Unmarshal(b, &own) != nil || json.Unmarshal(tb, &want) != nil {
		fmt.Printf("· %s isn't JSON mise-seo can read: add the SessionStart entry of docs/seo/README.md to it\n", path)
		return false
	}
	hooks, _ := own["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	ours, _ := want["hooks"].(map[string]any)["SessionStart"].([]any)
	have, _ := hooks["SessionStart"].([]any)
	hooks["SessionStart"] = append(have, ours...)
	own["hooks"] = hooks
	out, err := json.MarshalIndent(own, "", "  ")
	must(err)
	must(os.WriteFile(path, append(out, '\n'), 0o644))
	fmt.Printf("wrote %s: the hook's SessionStart entry, .claude/hooks/%s\n", path, templates.HookFile)
	return true
}

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

// stale names the files init wrote that this version would write differently, and the repo's own files whose
// mise-seo parts init would write again (keepOwn): its parts of .mise/config.toml (miseconfig's problems), a .gitignore
// without the template's lines, a .claude/settings.json that doesn't run the hook.
func (r repo) stale() []string {
	var out []string
	fs.WalkDir(templates.Tool, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || theirs[p] {
			return err
		}
		want, _ := fs.ReadFile(templates.Tool, p)
		// Line endings aside: a Windows checkout of the repo may have \r\n where init wrote \n.
		crlf := func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }
		b, err := os.ReadFile(filepath.Join(r.dir, filepath.FromSlash(p)))
		switch {
		case err != nil:
			out = append(out, p+" is missing: run "+initTask)
		case !bytes.Equal(crlf(b), crlf(want)):
			out = append(out, p+" is out of date: run "+initTask)
		}
		return nil
	})
	if b, err := os.ReadFile(filepath.Join(r.dir, ".mise", "config.toml")); err != nil {
		out = append(out, ".mise/config.toml is missing: it pins mise-seo; run "+initTask)
	} else {
		out = append(out, configTool().Problems(b)...)
	}
	if b, err := os.ReadFile(filepath.Join(r.dir, ".claude", "settings.json")); err == nil && !bytes.Contains(b, []byte(templates.HookFile)) {
		out = append(out, ".claude/settings.json doesn't run .claude/hooks/"+templates.HookFile+": run "+initTask)
	}
	if b, err := os.ReadFile(filepath.Join(r.dir, ".gitignore")); err == nil && len(ignoresMissing(b)) > 0 {
		out = append(out, ".gitignore lacks mise-seo's lines ("+strings.Join(ignoresMissing(b), "; ")+"): run "+initTask)
	}
	return out
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

// trustConfig has mise trust a config file mise-seo wrote, so its tasks run straight away.
func trustConfig(dir, file string) {
	path := filepath.Join(dir, filepath.FromSlash(file))
	if _, err := os.Stat(path); err != nil {
		return
	}
	if _, err := exec.LookPath("mise"); err != nil {
		return
	}
	if out, err := exec.Command("mise", "trust", "-q", path).CombinedOutput(); err != nil {
		fmt.Printf("mise didn't trust %s (%v %s): run mise trust %s\n", file, err, strings.TrimSpace(string(out)), file)
		return
	}
	fmt.Printf("✓ %s trusted by mise\n", file)
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
		fmt.Printf("· %s has no site, which every task works on: mise run seo:03-init --site https://your.site (the live one; mise run seo:template sets the template app's, on this machine)\n", config.File)
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
