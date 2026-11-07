package main

// updateMise is mise-seo update-mise (mise run mise-seo:update-mise): the repo's mise tools, in its mise files (mise upgrade --bump), and nothing else.
func (r repo) updateMise(o updates) { r.updatePins(o, "mise") }
