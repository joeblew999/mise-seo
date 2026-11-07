# mise-seo-template

A Go web app search engines can read, with [mise-seo](https://github.com/joeblew999/mise-seo): two pages in plain Go
(`net/http`, `html/template`, no gsx) behind `seo.Site`, which answers robots.txt, the sitemap and llms.txt, marks
whatever isn't a page noindex and gives every page one address; each page's `<head>` says what Google reads of it.

- `go run ./app` (`mise run serve`): the app, at http://localhost:8080, `seo.toml`'s site on this machine.
- `go test ./...` (`mise run test`): its tests, which audit it as Googlebot reads it.
- `mise-seo` (`mise run seo`): what's done, and what to run next. Beside `mise run serve`, `mise run seo:06-audit`
  audits the app as it runs.
- [docs/seo/README.md](docs/seo/README.md): mise-seo in this repo, every command.

Every tool comes from mise: `.mise/config.toml` pins mise-seo (and Go, `go.mod`'s), and `mise run setup` downloads the
modules `go.mod` pins, mise-seo's packages among them. Once the app is live, its address goes in `seo.toml`.
