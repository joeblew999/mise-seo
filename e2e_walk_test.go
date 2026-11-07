// Package e2e is this repo's end-to-end test: a walk of a repo's tasks, in the order a person meets them, through the repo's own mise file, on
// mise-check's walk (github.com/joeblew999/mise-check/walk), which runs the steps in their order and does everything
// that isn't mise-seo's. It sits at the top of this repo, beside the root mise.toml whose tasks run it:
//
//   - e2e_walk_test.go, this file: the walk tests the root's tasks run by name (TestWalkTemplate, TestWalkNewRepo),
//     TestSteps, and what's mise-seo's in making a repo and running its app.
//   - e2e_steps_test.go: the steps, the repo's tasks in the order a person meets them, each with its check.
//
// e2e_tool_test.go is the release (mise run tool:release), mise_test.go checks that the mise files and the Go line
// up, both ways, and gsx_test.go that mise-seo's packages but gsxseo need no gsx.
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/joeblew999/mise-check/walk"
)

// seoWalk is the walk: the template's tasks (templates/.mise/tasks/mise-seo.toml), in the order a person meets them,
// each checked. Which repo is the test's name: the template where it is, or a new repo made as docs/seo/README.md says (at
// the release -version names or this checkout's build).
var seoWalk = walk.Walk[*walker]{Steps: steps, New: newRepo, Prepare: withGo, Of: walkOf}

// TestWalkTemplate walks the template, templates/, where it is (mise run template:e2e). Each walk runs only when its
// task names it, so go test ./... never walks.
func TestWalkTemplate(t *testing.T) { seoWalk.Template(t, "templates") }

// TestWalkNewRepo walks a new repo made as docs/seo/README.md says (mise run new-repo:e2e).
func TestWalkNewRepo(t *testing.T) { seoWalk.NewRepo(t) }

// TestSteps checks that the steps are the template's numbered tasks, in their order, so the walk is that file's walk.
func TestSteps(t *testing.T) { walk.CheckSteps(t, "templates/.mise/tasks/mise-seo.toml", steps) }

// walker is the steps run on a repo in their order: what each step's Check is given.
type walker struct{ walk.Repo }

// walkOf is the walk of a repo the walk has set up (mise install, then its setup task): first, what it runs.
func walkOf(r walk.Repo) *walker {
	w := &walker{r}
	w.Pass("%s runs %s", w.Name, w.versions())
	return w
}

// withGo gives the template, where it is, the Go of the module it's a folder of. It has no go.mod of its own (one
// would make it a module apart from mise-seo's, whose embed it is), and the walk cuts mise off from the repo around
// it, so its own mise files have no Go to give its tasks; the Go this test runs with is that module's. A new repo has
// a go.mod of its own, whose Go mise installs.
func withGo(r walk.Repo) walk.Repo {
	if _, err := os.Stat(filepath.Join(r.Dir, "go.mod")); err == nil {
		return r
	}
	r.With = append(r.With, "MISE_GO_VERSION="+strings.TrimPrefix(runtime.Version(), "go"))
	return r
}

// newApp is a new repo's folder, which seo:template names its app's module for: its walk never touches the template's.
const newApp = "seo-new"

// newRepo is a new repo, made as docs/seo/README.md says a person makes one: an empty folder with its own git,
// mise-seo init at the release version names (mise exec github:joeblew999/mise-seo@<version> -- mise-seo init, which
// pins that release), or at this checkout's build without one (local, which mise run tool:install links), then
// seo:template, which writes the template's app, its module named for the folder. With this checkout's build, the
// go.mod is there first, using this checkout's packages (go mod init, then a replace), as a person trying a change
// has it.
func newRepo(r walk.Repo, version string) walk.Repo {
	at := strings.TrimPrefix(version, "v")
	if at == "" {
		at = "local"
		r.NeedLocal("github:joeblew999/mise-seo")
	}
	n := r.New(newApp)
	if at == "local" {
		n.Must("go", "mod", "init", newApp)
		n.Must("go", "mod", "edit", "-require=github.com/joeblew999/mise-seo@v0.0.0", "-replace=github.com/joeblew999/mise-seo="+r.Top)
	}
	n.Must("mise", "exec", "github:joeblew999/mise-seo@"+at, "--", "mise-seo", "init")
	want := `"github:joeblew999/mise-seo" = "` + at + `"`
	if b, _ := os.ReadFile(filepath.Join(n.Dir, ".mise", "config.toml")); !bytes.Contains(b, []byte(want)) {
		r.T.Fatalf("mise-seo %s's init didn't pin itself in .mise/config.toml (%s):\n%s", at, want, b)
	}
	n.MustTask("seo:template")
	if b, _ := os.ReadFile(filepath.Join(n.Dir, "go.mod")); !bytes.Contains(b, []byte("module "+newApp)) {
		r.T.Fatalf("seo:template didn't name the app's module for its folder, %s:\n%s", newApp, b)
	}
	n.Pass("%s: an empty folder, its own git, mise-seo init at %s, then seo:template (the template's app, its module named for the folder)", n.Name, at)
	return n
}

// versions is what the repo runs, each as its own tool reports it: its mise tools (mise ls --local --json) and the
// build of mise-seo (seo:show-version).
func (w *walker) versions() string {
	w.T.Helper()
	var tools map[string][]struct {
		Version string
		Active  bool
	}
	if err := json.Unmarshal(w.Stdout("mise", "ls", "--local", "--json"), &tools); err != nil || len(tools) == 0 {
		w.T.Fatalf("the repo's mise tools, mise ls --local --json: %v (%d read)", err, len(tools))
	}
	var v []string
	for _, name := range slices.Sorted(maps.Keys(tools)) {
		for _, t := range tools[name] {
			if t.Active {
				v = append(v, name+" "+t.Version)
			}
		}
	}
	build := strings.TrimSpace(string(w.Stdout("mise", "run", "seo:show-version")))
	if build == "" {
		w.T.Fatal("seo:show-version said no version")
	}
	return fmt.Sprintf("%s (mise); mise-seo's build %s", strings.Join(v, ", "), build)
}

// serve runs the app as a person does, with the repo's own serve task (mise run serve), on a free port, until it
// answers; it returns the app's address and what stops it.
func (w *walker) serve() (string, func()) {
	w.T.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		w.T.Fatal(err)
	}
	addr := "http://" + l.Addr().String()
	l.Close()
	cmd := exec.Command("mise", "run", "serve")
	cmd.Dir, cmd.Env = w.Dir, w.Env("ADDR="+strings.TrimPrefix(addr, "http://"))
	walk.OwnGroup(cmd)
	var logs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		w.T.Fatal(err)
	}
	stop := func() { walk.KillGroup(cmd) }
	for deadline := time.Now().Add(2 * time.Minute); time.Now().Before(deadline); time.Sleep(250 * time.Millisecond) {
		if resp, err := http.Get(addr + "/"); err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return addr, stop
		}
	}
	stop()
	w.T.Fatalf("the app (mise run serve) never answered on %s:\n%s", addr, logs.String())
	return "", nil
}
