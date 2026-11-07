// Command app is the template's little app: two pages in plain Go (net/http, html/template; no gsx) behind seo.Site,
// which answers search engines at the paths they ask for: robots.txt, the sitemap, llms.txt, noindex on whatever isn't
// a page, and one address for every page. Each page's <head> says what Google reads of it, which mise-seo's audit
// checks (mise run seo:06-audit). go run ./app serves it at http://localhost:8080 (ADDR to change it), seo.toml's site
// on this machine.
package main

import (
	"cmp"
	"html/template"
	"log"
	"net/http"
	"os"
	"slices"

	"github.com/joeblew999/mise-seo/seo"
)

func main() {
	addr := cmp.Or(os.Getenv("ADDR"), "localhost:8080")
	log.Printf("http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler()))
}

// siteName is the site's name, in results beside its pages and in a shared link.
const siteName = "A site Google reads"

// page is one page people read and Google may index: its path, its title and description (the blue link and the text
// under it in a result), and its text.
type page struct{ Path, Title, Description, Text string }

// pages are the app's pages: its routes, the sitemap, llms.txt and seo.Site's IsPage all come from them.
var pages = []page{
	{"/", siteName, "Two pages in plain Go behind seo.Site, each saying what search engines read of it, checked by mise-seo's audit.",
		"Every page has one address, a title and a description of its own, one heading, and is in the sitemap."},
	{"/about", "About this site", "How the template's app answers search engines: robots.txt, the sitemap, llms.txt and each page's head.",
		"seo.Site serves robots.txt, the sitemap and llms.txt, and marks whatever isn't a page noindex."},
}

// handler is the app: seo.Site around its pages, outside everything else, so it sees the paths crawlers ask for.
// APP_DOMAIN is the live site's one address: every other host, and plain http, moves there (none here, nor locally).
func handler() http.Handler {
	var paths []string
	mux := http.NewServeMux()
	for _, p := range pages {
		paths = append(paths, p.Path)
		pattern := "GET " + p.Path
		if p.Path == "/" {
			pattern += "{$}"
		}
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) { render(w, r, p) })
	}
	mux.HandleFunc("GET /icon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><circle cx="8" cy="8" r="7" fill="#0cce6b"/></svg>`))
	})
	site := seo.Site{
		Host:    os.Getenv("APP_DOMAIN"),
		Sitemap: func(origin string) string { return seo.Sitemap(origin, paths) },
		LLMs:    llms,
		IsPage:  func(path string) bool { return slices.Contains(paths, path) },
		Preview: os.Getenv("APP_PREVIEW") != "", // a preview is noindex throughout
	}
	return site.Handler(mux)
}

// llms is /llms.txt: what the site is, and its pages, for agents and language models (llmstxt.org).
func llms(origin string) string {
	var links []seo.LLMsLink
	for _, p := range pages {
		links = append(links, seo.LLMsLink{Title: p.Title, URL: origin + p.Path, Note: p.Description})
	}
	return seo.LLMsTxt(siteName, pages[0].Description, "", seo.LLMsSection{Name: "Pages", Links: links})
}

// render writes a page, with the head search engines and link previews read: its title and description, its one
// address (canonical, og:url), the site's icon and name, and on the home page the site's name as WebSite structured
// data. With gsx, gsxseo.Head writes the same.
func render(w http.ResponseWriter, r *http.Request, p page) {
	origin := seo.Origin(r)
	data := struct {
		page
		URL, SiteName string
		WebSite       map[string]string
		Pages         []page
	}{page: p, URL: origin + p.Path, SiteName: siteName, Pages: pages}
	if p.Path == "/" {
		data.WebSite = map[string]string{"@context": "https://schema.org", "@type": "WebSite", "name": siteName, "url": origin + "/", "inLanguage": "en"}
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var tmpl = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<meta name="description" content="{{.Description}}">
<link rel="canonical" href="{{.URL}}">
<link rel="icon" href="/icon.svg" type="image/svg+xml">
<meta property="og:type" content="website">
<meta property="og:site_name" content="{{.SiteName}}">
<meta property="og:title" content="{{.Title}}">
<meta property="og:description" content="{{.Description}}">
<meta property="og:url" content="{{.URL}}">
<meta name="twitter:card" content="summary">
{{- if .WebSite}}
<script type="application/ld+json">{{.WebSite}}</script>
{{- end}}
</head>
<body>
<h1>{{.Title}}</h1>
<p>{{.Text}}</p>
<nav>{{range .Pages}}
<a href="{{.Path}}">{{.Title}}</a>{{end}}
</nav>
</body>
</html>
`))
