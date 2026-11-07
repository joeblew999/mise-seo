package main

// updatePackages is mise-seo update-packages (mise run mise-seo:update-packages): the repo's Node packages, package.json and its lockfile (aube update), and nothing else.
func (r repo) updatePackages(o updates) { r.updatePins(o, "packages") }
