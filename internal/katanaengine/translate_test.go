package katanaengine

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/dotcommander/crawler/internal/config"
)

func TestScopeRegexes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seeds   []string
		allow   []string
		block   []string
		wantErr bool
	}{
		{
			name:  "document seed confines to base path",
			seeds: []string{"https://example.com/docs/index.html"},
			allow: []string{"https://example.com/docs/", "https://example.com/docs/a/b.html"},
			block: []string{"https://example.com/other.html", "https://example.com/docsother/x", "https://other.example.com/docs/a"},
		},
		{
			name:  "root seed allows whole host",
			seeds: []string{"https://example.com/"},
			allow: []string{"https://example.com/anything", "https://example.com/", "https://example.com"},
			block: []string{"https://www.example.com/anything", "http://example.com/anything"},
		},
		{
			name:  "bare-origin seed (no path) is itself in scope",
			seeds: []string{"https://example.com"},
			allow: []string{"https://example.com", "https://example.com/page"},
			block: []string{"https://example.com.evil.net/page"},
		},
		{
			name:  "root-file seed allows whole host (legacy baseDirectory semantics)",
			seeds: []string{"https://example.com/page.html"},
			allow: []string{"https://example.com/other.html"},
			block: []string{"https://api.example.com/other.html"},
		},
		{
			name:  "directory seed keeps trailing slash semantics",
			seeds: []string{"https://example.com/docs/"},
			allow: []string{"https://example.com/docs/page.html"},
			block: []string{"https://example.com/docsx/page.html"},
		},
		{
			name:  "multiple origins become alternation",
			seeds: []string{"https://a.com/x/index.html", "https://b.com/"},
			allow: []string{"https://a.com/x/p", "https://b.com/anything"},
			block: []string{"https://c.com/x/p"},
		},
		{
			name:    "invalid seed rejected",
			seeds:   []string{":::not-a-url:::"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ScopeRegexes(tt.seeds)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ScopeRegexes: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("expected exactly one combined regex, got %v", got)
			}
			re := regexp.MustCompile(got[0])
			for _, u := range tt.allow {
				if !re.MatchString(u) {
					t.Errorf("scope regex %q should allow %q", got[0], u)
				}
			}
			for _, u := range tt.block {
				if re.MatchString(u) {
					t.Errorf("scope regex %q should block %q", got[0], u)
				}
			}
		})
	}
}

func TestExcludeOutOfScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		block   []string
		allow   []string
	}{
		{
			name:    "full path glob",
			pattern: "/private/*",
			block:   []string{"http://h/private/x", "http://h/private/"},
			allow:   []string{"http://h/public/x"},
		},
		{
			name:    "last segment glob",
			pattern: "*.pdf",
			block:   []string{"http://h/a/b.pdf", "http://h/c.pdf"},
			allow:   []string{"http://h/a/b.pdf.txt", "http://h/a/pdfx"},
		},
		{
			name:    "substring pattern",
			pattern: "logout",
			block:   []string{"http://h/x/logout/y", "http://h/p?next=/logout"},
			allow:   []string{"http://h/x/log-in"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExcludeOutOfScope([]string{tt.pattern})
			if len(got) != 1 {
				t.Fatalf("expected one regex for %q, got %v", tt.pattern, got)
			}
			re := regexp.MustCompile(got[0])
			for _, u := range tt.block {
				if !re.MatchString(u) {
					t.Errorf("exclude regex %q should block %q", got[0], u)
				}
			}
			for _, u := range tt.allow {
				if re.MatchString(u) {
					t.Errorf("exclude regex %q should allow %q", got[0], u)
				}
			}
		})
	}
}

func TestPacingFromDelay(t *testing.T) {
	t.Parallel()
	tests := []struct {
		delay          time.Duration
		hostRate       int
		hostRateMinute int
	}{
		{0, 0, 0},
		{250 * time.Millisecond, 4, 0},
		{500 * time.Millisecond, 2, 0},
		{time.Second, 0, 60},
		{2 * time.Second, 0, 30},
	}
	for _, tt := range tests {
		hr, hrm := pacingFromDelay(tt.delay)
		if hr != tt.hostRate || hrm != tt.hostRateMinute {
			t.Errorf("pacingFromDelay(%v) = (%d,%d), want (%d,%d)", tt.delay, hr, hrm, tt.hostRate, tt.hostRateMinute)
		}
	}
}

func TestPageLoadStrategy(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"":                 "",
		"commit":           "domcontentloaded",
		"load":             "load",
		"domcontentloaded": "domcontentloaded",
		"networkidle":      "networkidle",
		"NetworkIdle":      "networkidle",
		"bogus":            "",
	}
	for in, want := range tests {
		if got := pageLoadStrategy(in); got != want {
			t.Errorf("pageLoadStrategy(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTranslateEngineTimeoutAndBodyRead(t *testing.T) {
	t.Parallel()

	base := &config.CrawlerConfig{StartURL: "https://example.com/"}

	opts, err := TranslateOptions(&config.CrawlerConfig{
		StartURL:             "https://example.com/",
		EngineTimeoutSeconds: 3,
	}, []string{"https://example.com/"})
	if err != nil {
		t.Fatalf("TranslateOptions: %v", err)
	}
	if opts.Timeout != 3 {
		t.Errorf("Timeout = %d, want 3", opts.Timeout)
	}
	if opts.BodyReadSize != 50*1024*1024 {
		t.Errorf("BodyReadSize = %d, want 50MB", opts.BodyReadSize)
	}

	defaulted, err := TranslateOptions(base, []string{"https://example.com/"})
	if err != nil {
		t.Fatalf("TranslateOptions: %v", err)
	}
	if defaulted.Timeout != 10 {
		t.Errorf("default Timeout = %d, want upstream 10", defaulted.Timeout)
	}
}

func TestMobileHeadersAndHeadlessArgs(t *testing.T) {
	t.Parallel()

	cfg := &config.CrawlerConfig{
		StartURL:        "https://example.com/",
		UserAgent:       "DesktopBot/1.0",
		MobileUserAgent: "MobileBot/1.0",
		Mobile:          true,
	}
	opts, err := TranslateOptions(cfg, []string{cfg.StartURL})
	if err != nil {
		t.Fatalf("TranslateOptions: %v", err)
	}

	foundUA := false
	for _, h := range opts.CustomHeaders {
		if h == "User-Agent: MobileBot/1.0" {
			foundUA = true
		}
		if h == "User-Agent: DesktopBot/1.0" {
			t.Errorf("mobile crawl must not send the desktop UA, got %q", h)
		}
	}
	if !foundUA {
		t.Errorf("CustomHeaders %v missing mobile UA", opts.CustomHeaders)
	}

	if len(opts.HeadlessOptionalArguments) == 0 {
		t.Fatal("mobile crawl must pass headless viewport args")
	}
	joined := strings.Join(opts.HeadlessOptionalArguments, " ")
	if !strings.Contains(joined, "--window-size=390,844") {
		t.Errorf("headless args %v missing window size", opts.HeadlessOptionalArguments)
	}

	desktop, err := TranslateOptions(&config.CrawlerConfig{StartURL: "https://example.com/"}, []string{"https://example.com/"})
	if err != nil {
		t.Fatalf("TranslateOptions: %v", err)
	}
	if len(desktop.HeadlessOptionalArguments) != 0 {
		t.Errorf("non-mobile headless args = %v, want none", desktop.HeadlessOptionalArguments)
	}
}

func TestWantsHeadless(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  *config.CrawlerConfig
		want bool
	}{
		{"mobile", &config.CrawlerConfig{Mobile: true}, true},
		{"commit strategy", &config.CrawlerConfig{WaitStrategy: "commit"}, true},
		{"load strategy", &config.CrawlerConfig{WaitStrategy: "load"}, true},
		{"networkidle default", &config.CrawlerConfig{WaitStrategy: "networkidle"}, false},
		{"empty strategy", &config.CrawlerConfig{}, false},
		{"long extra wait", &config.CrawlerConfig{ExtraWaitTime: 600 * time.Millisecond}, true},
		{"short extra wait", &config.CrawlerConfig{ExtraWaitTime: 400 * time.Millisecond}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := WantsHeadless(tc.cfg); got != tc.want {
				t.Errorf("WantsHeadless(%+v) = %v, want %v", tc.cfg, got, tc.want)
			}
		})
	}
}

func TestTranslateOptions(t *testing.T) {
	t.Parallel()

	cfg := &config.CrawlerConfig{
		StartURL:        "https://example.com/docs/",
		MaxDepth:        4,
		Concurrency:     7,
		DefaultDelay:    500 * time.Millisecond,
		MaxRetries:      2,
		ExcludePatterns: []string{"/private/*", "logout"},
		UserAgent:       "crawler-test",
		Headers:         map[string]string{"X-Custom": "yes"},
		JSCrawl:         true,
		ExtraWaitTime:   1500 * time.Millisecond,
		WaitStrategy:    "commit",
	}
	opts, err := TranslateOptions(cfg, []string{cfg.StartURL})
	if err != nil {
		t.Fatalf("TranslateOptions: %v", err)
	}

	if len(opts.URLs) != 1 || opts.URLs[0] != cfg.StartURL {
		t.Errorf("URLs = %v", opts.URLs)
	}
	if opts.MaxDepth != 4 {
		t.Errorf("MaxDepth = %d, want 4 (hop parity with crawler semantics)", opts.MaxDepth)
	}
	if opts.Concurrency != 7 || opts.Parallelism != 7 {
		t.Errorf("Concurrency/Parallelism = %d/%d, want 7/7", opts.Concurrency, opts.Parallelism)
	}
	if opts.HostRateLimit != 2 || opts.HostRateLimitMinute != 0 {
		t.Errorf("HostRateLimit/Minute = %d/%d, want 2/0", opts.HostRateLimit, opts.HostRateLimitMinute)
	}
	if opts.Retries != 2 {
		t.Errorf("Retries = %d, want 2", opts.Retries)
	}
	if len(opts.CustomHeaders) != 2 {
		t.Errorf("CustomHeaders = %v, want UA + X-Custom", opts.CustomHeaders)
	}
	if opts.FieldScope != "" {
		t.Errorf("FieldScope = %q, want empty (Scope regexes own it)", opts.FieldScope)
	}
	if len(opts.Scope) != 1 {
		t.Errorf("Scope = %v", opts.Scope)
	}
	if len(opts.OutOfScope) != 2 {
		t.Errorf("OutOfScope = %v, want 2 exclude regexes", opts.OutOfScope)
	}
	// Katana's KnownFiles discovery stays disabled: seeding and robots
	// compliance are crawler-owned (seeders + OutOfScope translation +
	// result-level gate); katana's parsers fetch robots/sitemap at the
	// seed path and enqueue duplicate requests (see translate.go).
	if opts.KnownFiles != "" {
		t.Errorf("KnownFiles = %q, want empty (crawler owns seeding)", opts.KnownFiles)
	}
	if !opts.ScrapeJSResponses {
		t.Error("ScrapeJSResponses should mirror JSCrawl")
	}
	if opts.PageLoadStrategy != "domcontentloaded" {
		t.Errorf("PageLoadStrategy = %q, want domcontentloaded", opts.PageLoadStrategy)
	}
	if opts.DOMWaitTime != 2 {
		t.Errorf("DOMWaitTime = %d, want 2", opts.DOMWaitTime)
	}
	if !opts.Silent || !opts.DisableUpdateCheck {
		t.Error("Silent and DisableUpdateCheck must be set")
	}
	if opts.BodyReadSize != 50*1024*1024 {
		t.Errorf("BodyReadSize = %d, want legacy-parity 50MB", opts.BodyReadSize)
	}
}

func TestTranslateOptionsNoRobots(t *testing.T) {
	t.Parallel()
	cfg := &config.CrawlerConfig{StartURL: "https://example.com/", NoRobots: true}
	opts, err := TranslateOptions(cfg, []string{cfg.StartURL})
	if err != nil {
		t.Fatalf("TranslateOptions: %v", err)
	}
	if opts.KnownFiles != "" {
		t.Errorf("KnownFiles = %q, want empty for NoRobots", opts.KnownFiles)
	}
}

func TestTranslateOptionsEmptySeeds(t *testing.T) {
	t.Parallel()
	if _, err := TranslateOptions(&config.CrawlerConfig{}, nil); err == nil {
		t.Fatal("expected error for empty seeds")
	}
}
