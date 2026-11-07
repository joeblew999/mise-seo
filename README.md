# mise-seo

A mise tool repo, made by `mise-check new-tool github.com/joeblew999/mise-seo` from mise-check v0.4.31: the `mise-seo` binary
(`cmd/mise-seo/`), its template (`templates/`), its walks, its release and its CI, as mise-check's are.

## Next

1. `mise install`, then `mise run root:ci`: what its CI runs, here the same. It passes as it comes.
2. Its commands are mise-check's, renamed: keep those the tool needs. To remove one, delete its file,
   `cmd/mise-seo/<command>.go`, and run `mise run root:ci`: misecheck names its task, its skill row and its docs row
   still left.
3. Write the tool's own: a file per command in `cmd/mise-seo/`, its row in `var commands` in `main.go`, its task in
   `templates/.mise/tasks/mise-seo.toml`, and its rows in the template's skill and docs. misecheck names each one missing.
4. Replace this README with the tool's own: the problem it solves, and what it does.

mise-check is used here as in any repo, through its own tasks, which new-tool wrote with `mise-check init`:
`mise run mise-check` (what's done, what's next), `mise-check:check`, `mise-check:update` (every pin, mise-check's two at
one release). `docs/mise-check/README.md` has all of it. `mise tasks` lists every task here. CLAUDE.md loads this README.
