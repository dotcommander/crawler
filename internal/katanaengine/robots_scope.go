package katanaengine

import (
	"net/url"
	"regexp"
	"strings"
)

// OutOfScopeFromRobots converts robots.txt disallow paths into katana
// OutOfScope regexes anchored to the seed origin. The regex set is a
// superset of the disallow rules that apply to any single user agent:
// the seeders' DisallowedPaths scan is user-agent agnostic and carries
// no Allow-rule precedence. Deviations therefore only ever over-block
// (skip pages a specific agent is allowed to fetch), never under-block.
// Precise per-agent semantics stay enforced by the seeders.IsAllowed
// gate in the engine for anything katana still surfaces.
func OutOfScopeFromRobots(seeds []string, disallowedPaths []string) []string {
	out := make([]string, 0, len(disallowedPaths))
	seen := make(map[string]struct{}, len(disallowedPaths))
	for _, seed := range seeds {
		u, err := url.Parse(seed)
		if err != nil || u.Scheme == "" || u.Host == "" {
			continue
		}
		origin := u.Scheme + "://" + u.Host
		for _, p := range disallowedPaths {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			re := "^" + regexp.QuoteMeta(origin) + robotsPathRegex(p)
			if _, dup := seen[re]; dup {
				continue
			}
			seen[re] = struct{}{}
			out = append(out, re)
		}
	}
	return out
}

// robotsPathRegex translates a robots.txt disallow path into an
// unanchored suffix regex. Robots matching is prefix-based, so plain
// paths get a trailing ".*"; Google-style wildcards map "*" to ".*" and
// a trailing "$" anchors the end.
func robotsPathRegex(p string) string {
	anchorEnd := strings.HasSuffix(p, "$")
	p = strings.TrimSuffix(p, "$")
	var b strings.Builder
	for _, r := range p {
		if r == '*' {
			b.WriteString(".*")
			continue
		}
		b.WriteString(regexp.QuoteMeta(string(r)))
	}
	if anchorEnd {
		return b.String() + "$"
	}
	return b.String() + ".*"
}
