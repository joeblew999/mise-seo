package main

import (
	"fmt"

	"github.com/joeblew999/mise-seo/seo/config"
)

// report is mise-seo show --json: where the repo is, and what to run next.
type report struct {
	Version  string   `json:"version"`  // this mise-seo: the release .mise/config.toml pins
	Site     string   `json:"site"`     // seo.toml's site, "" for none
	App      bool     `json:"app"`      // the repo has Go of its own (a go.mod, or a Go program), or the template's app
	Problems []string `json:"problems"` // check's
	Next     string   `json:"next"`     // the task to run next
}

// show is mise-seo show (seo:05-show), and mise-seo with no command (mise run seo): the steps, which are done for this
// repo, and the next one. It asks no server anything, so it's quick.
func (r repo) show(args []string) {
	f := flags("show")
	jsonOut := f.Bool("json", false, "one JSON document on stdout")
	parse(f, args, 0)
	s := r.where()
	if *jsonOut {
		emit(s)
		return
	}
	mark := map[bool]string{true: "✓", false: "·"}
	fmt.Printf("mise-seo %s; site: %s\n", s.Version, or(s.Site, "none"))
	fmt.Printf("%s 03-init    mise-seo's files, as this version writes them, and seo.toml\n", mark[!s.initDue()])
	fmt.Printf("%s app        Go of the repo's own (or the template's app: mise run seo:template)\n", mark[s.App])
	fmt.Printf("%s site       seo.toml's live site, which every task works on\n", mark[s.Site != ""])
	fmt.Printf("%s 04-check   the repo's rules\n", mark[len(s.Problems) == 0])
	fmt.Println("· 06-audit   the site as Googlebot reads it, after every deploy; then Lighthouse and Google (mise run seo:show-help)")
	for _, p := range s.Problems {
		fmt.Println("  ✗ " + p)
	}
	fmt.Println("next: " + s.Next)
}

// where is where the repo is, which show says: the problems, whether it has Go of its own and a site, and the step to
// take next. A repo with a go.mod has code of its own: the template, which would add its little app beside it, isn't
// next.
func (r repo) where() report {
	s := report{Version: buildVersion(), Problems: append([]string{}, r.rules()...), App: r.hasApp() || r.exists("go.mod")}
	if c, err := config.Find(r.dir); err == nil {
		s.Site = c.Site
	}
	initDue := len(r.initRules()) > 0
	switch {
	case initDue:
		s.Next = initTask
	case !s.App:
		s.Next = "mise run seo:template"
	case s.Site == "":
		s.Next = initTask + " --site https://your.site"
	case len(s.Problems) > 0:
		s.Next = "mise run seo:04-check"
	default:
		s.Next = "mise run seo:06-audit"
	}
	return s
}

// initDue reports whether init's step is next, or before it.
func (s report) initDue() bool { return s.Next == initTask }

// initTask is the task that writes init's files again.
const initTask = "mise run seo:03-init"
