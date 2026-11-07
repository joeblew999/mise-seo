package main

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/joeblew999/mise-seo/seo/audit"
)

// TestSearchEngines: the app as Googlebot reads it, over HTTP: robots.txt, the sitemap, llms.txt and every page pass
// mise-seo's audit, which mise run seo:06-audit runs on the live site, so a page Google couldn't index fails here first.
func TestSearchEngines(t *testing.T) {
	srv := httptest.NewServer(handler())
	defer srv.Close()
	urls, problems, err := audit.Audit(context.Background(), srv.Client(), srv.URL)
	if err != nil || urls != len(pages) {
		t.Fatalf("the sitemap: %d URLs, %v; want %d", urls, err, len(pages))
	}
	for _, p := range problems {
		t.Error(p)
	}
}
