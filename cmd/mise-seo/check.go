package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joeblew999/mise-seo/seo/config"
)

// check is mise-seo check (seo:check): the repo's rules, each with what fixes it; exit 1 on any.
func (r repo) check(args []string) {
	f := flags("check")
	jsonOut := f.Bool("json", false, "one JSON document on stdout")
	parse(f, args, 0)
	problems := r.rules()
	if *jsonOut {
		emit(struct {
			OK       bool     `json:"ok"`
			Problems []string `json:"problems"`
		}{len(problems) == 0, append([]string{}, problems...)})
	} else {
		if len(problems) == 0 {
			fmt.Println("✓ rules: pass")
		}
		for _, p := range problems {
			fmt.Println("✗ " + p)
		}
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
}

// rules are check's problems: init's (initRules), seo.toml with a site, and mise-seo as one release, its binary and
// its packages in go.mod. None, and the repo passes.
func (r repo) rules() []string {
	problems := r.initRules()
	switch c, err := config.Find(r.dir); {
	case err != nil:
		problems = append(problems, fmt.Sprintf("%s: %v", config.File, err))
	case c.Path == "":
		problems = append(problems, config.File+" is missing: run mise run seo:init")
	case c.Site == "":
		problems = append(problems, config.File+" has no site, which every task works on: mise run seo:init --site https://your.site")
	}
	if p := r.apart(); p != "" {
		problems = append(problems, p)
	}
	return problems
}

// initRules are the problems init fixes: its files as this version writes them, and its parts of the repo's own
// (stale).
func (r repo) initRules() []string { return r.stale() }

// apart is the problem with mise-seo's two pins in the repo, "" when they're one release: the binary running this, a
// release (.mise/config.toml's pin), and the packages its go.mod requires. When the binary is the older, the way out
// is the one line that needs nothing of it: init at the packages' release, an exact one, which mise gets at once.
func (r repo) apart() string {
	b, err := os.ReadFile(filepath.Join(r.dir, "go.mod"))
	if err != nil || version == "" {
		return ""
	}
	pkgs := strings.TrimPrefix(requiredVersion(string(b), selfModule), "v")
	bin := buildVersion()
	if pkgs == "" || pkgs == bin {
		return ""
	}
	fix := "mise run seo:init moves go.mod's to the binary's"
	if older(bin, pkgs) {
		fix = "for the binary at the packages' release, run " + initAt(pkgs)
	}
	return fmt.Sprintf("mise-seo is one release, and the repo pins two: %s in .mise/config.toml, %s in go.mod (%s); %s", bin, pkgs, selfModule, fix)
}
