package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/joeblew999/mise-seo/seo/config"
	"github.com/joeblew999/mise-seo/templates"
)

// template is mise-seo template (seo:template): the template's little app (templates.App: two pages in plain Go
// behind seo.Site, its test auditing them as Googlebot reads them, its tasks, its README) written into a repo with no
// Go program yet, so it runs straight away; seo.toml's site, when it sets none, is the app on this machine, which
// mise run serve serves. It never writes over a file: a repo with a program of its own gets none of it. Then the
// repo's setup task, and go mod tidy for the app's imports.
func (r repo) template(args []string) {
	parse(flags("template"), args, 0)
	name := moduleName(filepath.Base(r.dir))
	if r.exists("README.md") {
		fmt.Println("✓ README.md is the repo's own: left as it is")
	} else {
		b, _ := fs.ReadFile(templates.App, "README.md")
		writeFile(filepath.Join(r.dir, "README.md"), bytes.ReplaceAll(b, []byte(templates.AppName), []byte(name)))
	}
	if r.hasApp() {
		fmt.Println("✓ the repo has a Go program of its own (main.go, app/main.go or cmd/<name>/main.go): none of the template's app written")
		return
	}
	for _, f := range []string{"app/main.go", "app/main_test.go"} {
		if r.exists(f) {
			fmt.Printf("✓ %s is the repo's own: left as it is\n", f)
			continue
		}
		b, _ := fs.ReadFile(templates.App, f)
		writeFile(filepath.Join(r.dir, filepath.FromSlash(f)), b)
	}
	if r.exists("go.mod") {
		fmt.Println("✓ go.mod is the repo's own: left as it is")
	} else {
		// Written, not go mod init: before the repo has a go.mod, mise has no Go to give (it installs the Go a go.mod
		// names), and a machine needn't have one of its own. Its go line is the Go mise-seo was built with, and it
		// requires mise-seo's packages at this release (a checkout's build leaves them to go mod tidy, or the repo's
		// replace).
		gomod := "module " + name + "\n\ngo " + strings.TrimPrefix(runtime.Version(), "go") + "\n"
		if version != "" {
			gomod += "\nrequire " + selfModule + " v" + buildVersion() + "\n"
		}
		writeFile(filepath.Join(r.dir, "go.mod"), []byte(gomod))
	}
	path := filepath.Join(r.dir, config.File)
	own, err := os.ReadFile(path)
	if err != nil {
		own = []byte(config.Template)
	}
	if b, ok := config.SetSite(string(own), templates.AppSite); ok {
		writeFile(path, []byte(b))
	}
	tasks, _ := fs.ReadFile(templates.App, "mise.toml")
	if r.exists("mise.toml") {
		fmt.Printf("✓ mise.toml is the repo's own: left as it is. The template's app runs and tests with these:\n\n%s\n", tasks)
	} else {
		writeFile(filepath.Join(r.dir, "mise.toml"), tasks)
		trustConfig(r.dir, "mise.toml")
	}
	r.setUpApp()
	fmt.Println("✓ the template's app: next, mise run seo, then mise run serve and, beside it, mise run seo:06-audit")
}

// setUpApp installs what the template's app needs, as mise-intl's template does: the repo's own setup task (mise
// installs the Go go.mod names for it first), then go mod tidy, which puts the app's imports in go.mod and go.sum.
func (r repo) setUpApp() {
	has := exec.Command("mise", "tasks", "info", "setup")
	has.Dir = r.dir
	if has.Run() == nil {
		fmt.Println("$ mise run setup")
		cmd := exec.Command("mise", "run", "setup")
		cmd.Dir, cmd.Stdin, cmd.Stdout, cmd.Stderr = r.dir, os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			fail(fmt.Errorf("mise run setup: %w", err))
		}
	}
	fmt.Println("$ go mod tidy")
	tidy := goCommand(r.dir, "mod", "tidy")
	tidy.Stdout, tidy.Stderr = os.Stdout, os.Stderr
	if err := tidy.Run(); err != nil {
		fail(fmt.Errorf("go mod tidy: %w", err))
	}
}

// hasApp reports whether the repo has a Go program of its own: a main.go at its top, the template's, app/main.go, or
// a binary's, cmd/<name>/main.go.
func (r repo) hasApp() bool {
	cmds, _ := filepath.Glob(filepath.Join(r.dir, "cmd", "*", "main.go"))
	return r.exists("main.go") || r.exists("app/main.go") || len(cmds) > 0
}

// moduleName is a folder's name as a Go module's: lower case, with only letters, digits, dots, dashes and underscores.
func moduleName(folder string) string {
	s := strings.Trim(regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(strings.ToLower(folder), "-"), "-._")
	if s == "" {
		return "app"
	}
	return s
}
