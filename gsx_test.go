package e2e

import (
	"os/exec"
	"strings"
	"testing"
)

// TestWithoutGsx: only gsxseo imports gsx. seo, its audit and config, the command and everything else build without
// it, so an app without gsx never gets gsx in its go.mod, not even through mise-seo.
func TestWithoutGsx(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "./seo/...", "./cmd/mise-seo", "./internal/...", "./templates/...").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	for _, p := range strings.Fields(string(out)) {
		if strings.HasPrefix(p, "github.com/gsxhq/gsx") {
			t.Errorf("%s is built with gsx: only gsxseo may import it", p)
		}
	}
}
