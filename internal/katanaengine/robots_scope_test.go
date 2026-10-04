package katanaengine

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/dotcommander/crawler/internal/seeders"
)

func fetchRobots(t *testing.T, body string) *seeders.RobotsResult {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	result, err := seeders.FetchRobotsTxt(t.Context(), server.URL, false)
	if err != nil {
		t.Fatalf("FetchRobotsTxt: %v", err)
	}
	return result
}

func TestOutOfScopeFromRobots(t *testing.T) {
	t.Parallel()

	robots := fetchRobots(t, "User-agent: *\nDisallow: /private\nDisallow: /tmp/*.pdf$\nDisallow: /blocked\n")
	got := OutOfScopeFromRobots([]string{"https://example.com/"}, robots.DisallowedPaths)
	if len(got) != 3 {
		t.Fatalf("expected 3 regexes, got %v", got)
	}

	tests := []struct {
		url   string
		block bool
	}{
		{"https://example.com/private", true},          // exact disallowed path
		{"https://example.com/private/x/y.html", true}, // prefix semantics
		{"https://example.com/privatefoo", true},       // robots prefix match includes suffixes
		{"https://example.com/tmp/a.pdf", true},        // wildcard + $ anchor
		{"https://example.com/tmp/a.pdfx", false},      // $ anchor excludes suffixes
		{"https://example.com/blocked?x=1", true},      // query after prefix
		{"https://example.com/public", false},          // allowed path
		{"https://other.com/private", false},           // different origin is not blocked
	}
	for _, tt := range tests {
		matched := false
		for _, re := range got {
			if regexp.MustCompile(re).MatchString(tt.url) {
				matched = true
				break
			}
		}
		if matched != tt.block {
			t.Errorf("%q matched=%v, want block=%v", tt.url, matched, tt.block)
		}
	}
}

func TestOutOfScopeFromRobotsRootDisallow(t *testing.T) {
	t.Parallel()
	robots := fetchRobots(t, "User-agent: *\nDisallow: /\n")
	got := OutOfScopeFromRobots([]string{"https://example.com/"}, robots.DisallowedPaths)
	if len(got) != 1 {
		t.Fatalf("expected 1 regex, got %v", got)
	}
	re := regexp.MustCompile(got[0])
	if !re.MatchString("https://example.com/anything") {
		t.Error("root disallow should block the whole origin")
	}
	if re.MatchString("https://other.com/") {
		t.Error("root disallow must stay origin-anchored")
	}
}

// TestOutOfScopeParityNoUnderBlock is the D1 parity gate: every URL that
// the authoritative per-agent robots decision (robotstxt.TestAgent via
// seeders) disallows must also match a generated OutOfScope regex, so the
// translation never lets katana fetch a disallowed page.
func TestOutOfScopeParityNoUnderBlock(t *testing.T) {
	t.Parallel()

	const body = "User-agent: *\nDisallow: /private\nDisallow: /tmp\nAllow: /tmp/public\n"
	robots := fetchRobots(t, body)
	regexes := OutOfScopeFromRobots([]string{"https://example.com/"}, robots.DisallowedPaths)

	disallowed := []string{
		"https://example.com/private",
		"https://example.com/private/deep.html",
		"https://example.com/tmp/x.html", // /tmp/public allowed, but /tmp itself disallowed
	}
	allowed := []string{
		"https://example.com/",
		"https://example.com/tmp/public/x",
		"https://example.com/open",
	}

	for _, u := range disallowed {
		if robots.IsAllowed(pathOf(u), "*") {
			t.Fatalf("fixture error: %q should be disallowed by TestAgent", u)
		}
		matched := false
		for _, re := range regexes {
			if regexp.MustCompile(re).MatchString(u) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("under-block: %q is robots-disallowed but no OutOfScope regex matches (%v)", u, regexes)
		}
	}
	for _, u := range allowed {
		if !robots.IsAllowed(pathOf(u), "*") {
			t.Fatalf("fixture error: %q should be allowed by TestAgent", u)
		}
	}
}

// TestOutOfScopeDocumentedOverBlock pins the documented deviation: the
// naive DisallowedPaths scan ignores agent groups and Allow precedence,
// so it can block pages the authoritative decision allows. Over-blocking
// is the accepted safe direction; under-blocking is not.
func TestOutOfScopeDocumentedOverBlock(t *testing.T) {
	t.Parallel()

	const body = "User-agent: BadBot\nDisallow: /tmp\n\nUser-agent: *\nDisallow: \n"
	robots := fetchRobots(t, body)
	regexes := OutOfScopeFromRobots([]string{"https://example.com/"}, robots.DisallowedPaths)

	if !robots.IsAllowed("/tmp/x", "*") {
		t.Fatal("fixture error: default agent should be allowed in /tmp")
	}
	matched := false
	for _, re := range regexes {
		if regexp.MustCompile(re).MatchString("https://example.com/tmp/x") {
			matched = true
			break
		}
	}
	if !matched {
		t.Error("expected documented over-block: BadBot-only /tmp rule should still generate a regex")
	}
}
