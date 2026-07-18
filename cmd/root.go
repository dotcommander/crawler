package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dotcommander/crawler/api"
	"github.com/dotcommander/crawler/internal/crawlers"
	"github.com/dotcommander/crawler/internal/exporters"
	"github.com/dotcommander/crawler/internal/seeders"
	"github.com/dotcommander/crawler/internal/session"

	"github.com/alecthomas/kong"
)

// version is the build version. Defaults to "dev"; overridden at release
// build time via ldflags:
//
//	-ldflags "-X github.com/dotcommander/crawler/cmd.version=<ver>"
var version = "dev"

type commandTree struct {
	Verbose      bool             `short:"v" help:"Show detailed progress" xor:"output-mode"`
	Quiet        bool             `short:"q" help:"Pipeline mode: JSONL to stdout, no UI" xor:"output-mode"`
	OutputDir    string           `name:"output" short:"o" help:"Save files to directory"`
	MaxPages     int              `name:"max-pages" short:"p" help:"Stop after N pages (0=unlimited)"`
	Mobile       bool             `short:"m" help:"Crawl as mobile device"`
	ConfigFile   string           `name:"config" short:"c" help:"Use config file"`
	Profile      string           `help:"Use preset: fast, safe, or thorough"`
	JSCrawl      bool             `name:"jc" help:"Extract endpoints from JavaScript content"`
	URLListFile  string           `name:"url-list" help:"File with newline-delimited URLs to crawl"`
	ExportFormat string           `name:"format" help:"Export format: jsonl, csv, sitemap"`
	ExportFile   string           `name:"export-file" help:"Export output file (default: stdout)"`
	Resume       bool             `help:"Resume a previous crawl session"`
	NoRobots     bool             `name:"no-robots" help:"Skip robots.txt and sitemap seeding"`
	ExtractFlag  string           `name:"extract" help:"CSS selectors: title=h1,desc=.summary"`
	Version      kong.VersionFlag `help:"Print version and exit"`
	URL          string           `arg:"" optional:"" help:"URL to crawl"`
}

func Execute() error {
	return execute(context.Background(), os.Args[1:], os.Stderr)
}

func execute(ctx context.Context, args []string, stderr io.Writer) error {
	if len(args) > 0 && args[0] == "serve" {
		serve, err := parseServeCommand(args[1:], stderr)
		if err != nil {
			return err
		}
		return serve.Run(ctx, stderr)
	}
	tree, err := parseCommand(args, stderr)
	if err != nil {
		return err
	}
	return run(ctx, tree, stderr)
}

func parseCommand(args []string, stderr io.Writer) (*commandTree, error) {
	var tree commandTree
	parser, err := kong.New(&tree,
		kong.Name("crawler"),
		kong.Description("Fast and smart web crawler with JavaScript support. Use `crawler serve [directory]` to browse captured content."),
		kong.Vars{"version": version},
		kong.Writers(stderr, stderr),
	)
	if err != nil {
		return nil, err
	}
	_, err = parser.Parse(args)
	if err != nil {
		return nil, err
	}
	return &tree, nil
}

func run(ctx context.Context, opts *commandTree, stderr io.Writer) error {
	// Collect URLs from positional arg and/or --url-list file
	var urls []string
	if opts.URL != "" {
		urls = append(urls, opts.URL)
	}
	if opts.URLListFile != "" {
		fileURLs, err := parseURLFile(opts.URLListFile)
		if err != nil {
			return fmt.Errorf("failed to read URL list: %w", err)
		}
		urls = append(urls, fileURLs...)
	}
	if len(urls) == 0 {
		return fmt.Errorf("provide at least one URL as an argument or via --url-list")
	}

	// Validate all URLs before starting
	for _, u := range urls {
		parsed, err := url.Parse(u)
		if err != nil {
			return fmt.Errorf("invalid URL %q: %w", u, err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("invalid URL %q: must be http or https", u)
		}
	}

	if !opts.Quiet {
		fmt.Fprintf(stderr, "Loaded %d seed URL(s)\n", len(urls))
	}

	// Use first URL as primary for config building
	startURL := urls[0]

	// Load and build configuration
	crawlerConfig, err := loadAndBuildConfig(opts.ConfigFile, startURL, opts.OutputDir, opts.Profile, opts.Mobile, opts.MaxPages)
	if err != nil {
		return err
	}
	crawlerConfig.JSCrawl = opts.JSCrawl
	crawlerConfig.StartURLs = urls
	crawlerConfig.ExportFormat = opts.ExportFormat
	crawlerConfig.ExportFile = opts.ExportFile
	crawlerConfig.Resume = opts.Resume
	crawlerConfig.NoRobots = opts.NoRobots
	crawlerConfig.Quiet = opts.Quiet

	// Parse --extract selectors
	if opts.ExtractFlag != "" {
		selectors, parseErr := parseExtractFlag(opts.ExtractFlag)
		if parseErr != nil {
			return parseErr
		}
		crawlerConfig.ExtractSelectors = selectors
	}

	// Quiet mode: default to JSONL export on stdout
	if opts.Quiet {
		if crawlerConfig.ExportFormat == "" {
			crawlerConfig.ExportFormat = "jsonl"
		}
	}

	// Set up logging
	setupLogging(opts.Verbose, opts.Quiet)

	// Pre-seed from robots.txt and sitemaps
	var robotsChecker func(path, userAgent string) bool
	if !crawlerConfig.NoRobots {
		if robots, robotsErr := seeders.FetchRobotsTxt(ctx, crawlerConfig.StartURL, opts.Verbose); robotsErr == nil && robots != nil {
			robotsChecker = robots.IsAllowed
		} else if robotsErr != nil && opts.Verbose {
			log.Printf("[WARN] robots.txt fetch for discovery enforcement failed: %v", robotsErr)
		}

		seedResult, seedErr := seeders.Seed(ctx, crawlerConfig.StartURL, crawlerConfig.UserAgent, crawlerConfig.ExcludePatterns, opts.Verbose)
		if seedErr != nil {
			log.Printf("[WARN] Seeding failed: %v", seedErr)
		} else if len(seedResult.URLs) > 0 {
			crawlerConfig.SeedURLs = seedResult.URLs
			if opts.Verbose {
				log.Printf("[INFO] Pre-seeded %d URLs from sitemaps (%d filtered by robots.txt)",
					len(seedResult.URLs), seedResult.DisallowedCount)
			}
		}
	}

	// Create visited store (always SQLite for persistence)
	sessionsDir := session.GetSessionsDir()
	var store session.VisitedStore
	sqliteStore, storeErr := session.NewSQLiteStore(sessionsDir, startURL, opts.Resume)
	if storeErr != nil {
		log.Printf("Warning: failed to create session store, using in-memory: %v", storeErr)
		store = nil // NewEngineCrawler falls back to MemoryStore
	} else {
		store = sqliteStore
	}

	// Create crawler instance (auto-detect engine)
	crawler, err := crawlers.CreateCrawler(crawlerConfig, opts.Verbose, "", store)
	if err != nil {
		if store != nil {
			store.Close()
		}
		return fmt.Errorf("failed to create crawler: %w", err)
	}

	// Enforce robots.txt on links discovered mid-crawl, not just seeds.
	if robotsChecker != nil {
		setRobotsCheckerOnCrawler(crawler, robotsChecker)
	}

	// Set up exporter if format is specified
	if crawlerConfig.ExportFormat != "" {
		extractedKeys := make([]string, 0, len(crawlerConfig.ExtractSelectors))
		for name := range crawlerConfig.ExtractSelectors {
			extractedKeys = append(extractedKeys, name)
		}
		exp, cleanup, expErr := setupExporter(crawlerConfig.ExportFormat, crawlerConfig.ExportFile, extractedKeys)
		if expErr != nil {
			crawler.Close()
			return fmt.Errorf("failed to set up exporter: %w", expErr)
		}
		defer func() {
			_ = exp.Close()
			if cleanup != nil {
				cleanup()
			}
		}()
		setExporterOnCrawler(crawler, exp)
	}
	// Close the crawler (which drains workers) BEFORE the exporter defer fires.
	// Defers run LIFO, so registering this after the exporter defer above makes
	// it execute first: in-flight workers finish writing, then the exporter is
	// closed and flushed against a complete record stream.
	defer crawler.Close()

	// Run crawler with signal handling
	return runCrawlerWithSignalHandling(crawler)
}

// setupExporter creates an exporter and its output writer.
func setupExporter(format, filePath string, extractedKeys []string) (exporters.Exporter, func(), error) {
	var w *os.File
	var cleanup func()

	if filePath == "" {
		if format == "sitemap" {
			return nil, nil, fmt.Errorf("sitemap format requires --export-file")
		}
		w = os.Stdout
	} else {
		f, err := os.Create(filePath)
		if err != nil {
			return nil, nil, fmt.Errorf("create export file %s: %w", filePath, err)
		}
		w = f
		cleanup = func() { _ = f.Close() }
	}

	exp, err := exporters.New(format, w, extractedKeys...)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	return exp, cleanup, nil
}

// setExporterOnCrawler sets the exporter on the underlying EngineCrawler.
func setExporterOnCrawler(c api.Crawler, exp exporters.Exporter) {
	type exporterSetter interface {
		SetExporter(exporters.Exporter)
	}

	if es, ok := c.(exporterSetter); ok {
		es.SetExporter(exp)
		return
	}

	type crawlerWithInner interface {
		GetInnerCrawler() api.Crawler
	}
	if cw, ok := c.(crawlerWithInner); ok {
		if es, ok := cw.GetInnerCrawler().(exporterSetter); ok {
			es.SetExporter(exp)
		}
	}
}

// setRobotsCheckerOnCrawler installs the robots.txt discovery-time allow
// predicate on the underlying EngineCrawler, mirroring setExporterOnCrawler.
func setRobotsCheckerOnCrawler(c api.Crawler, fn func(path, userAgent string) bool) {
	type robotsSetter interface {
		SetRobotsChecker(func(path, userAgent string) bool)
	}

	if rs, ok := c.(robotsSetter); ok {
		rs.SetRobotsChecker(fn)
		return
	}

	type crawlerWithInner interface {
		GetInnerCrawler() api.Crawler
	}
	if cw, ok := c.(crawlerWithInner); ok {
		if rs, ok := cw.GetInnerCrawler().(robotsSetter); ok {
			rs.SetRobotsChecker(fn)
		}
	}
}

// parseURLFile reads a file of newline-delimited URLs, skipping empty lines and # comments.
func parseURLFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var urls []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs found in %s", path)
	}
	return urls, nil
}

// parseExtractFlag parses "key=selector,key=selector" into a map.
// Commas inside square brackets are not treated as separators.
func parseExtractFlag(val string) (map[string]string, error) {
	selectors := make(map[string]string)
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(val); i++ {
		switch val[i] {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, val[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, val[start:])

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.IndexByte(part, '=')
		if idx <= 0 {
			return nil, fmt.Errorf("invalid --extract entry %q: expected key=selector", part)
		}
		key := strings.TrimSpace(part[:idx])
		sel := strings.TrimSpace(part[idx+1:])
		if key == "" || sel == "" {
			return nil, fmt.Errorf("invalid --extract entry %q: key and selector must be non-empty", part)
		}
		selectors[key] = sel
	}
	if len(selectors) == 0 {
		return nil, fmt.Errorf("--extract provided but no valid selectors parsed")
	}
	return selectors, nil
}

// findConfigFile searches for crawl.yml in multiple locations
func findConfigFile() string {
	searchPaths := []string{
		"crawl.yml",
		filepath.Join(getConfigDir(), "crawl.yml"),
	}

	if envConfig := os.Getenv("CRAWLER_CONFIG"); envConfig != "" {
		searchPaths = append([]string{envConfig}, searchPaths...)
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// getConfigDir returns the user configuration directory
func getConfigDir() string {
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "crawler")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "crawler")
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "crawler")
	default:
		return filepath.Join(home, ".config", "crawler")
	}
}
