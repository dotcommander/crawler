package cmd

import (
	"context"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dotcommander/crawler/internal/config"

	"github.com/alecthomas/kong"
)

type serveCommand struct {
	Directory string `arg:"" optional:"" help:"Directory to serve"`
	Port      int    `short:"p" default:"8080" help:"Port to listen on"`
	Host      string `default:"localhost" help:"Host to bind to"`
}

func parseServeCommand(args []string, stderr io.Writer) (*serveCommand, error) {
	var cmd serveCommand
	parser, err := kong.New(&cmd, kong.Name("crawler serve"), kong.Description("Browse crawled content via local HTTP server"), kong.Writers(stderr, stderr))
	if err != nil {
		return nil, err
	}
	if _, err := parser.Parse(args); err != nil {
		return nil, err
	}
	return &cmd, nil
}

func (cmd *serveCommand) Run(ctx context.Context, stderr io.Writer) error {
	dir := config.GetDefaultOutputDir()
	if cmd.Directory != "" {
		dir = cmd.Directory
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve directory: %w", err)
	}
	if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	addr := net.JoinHostPort(cmd.Host, fmt.Sprintf("%d", cmd.Port))

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			if q := r.URL.Query().Get("q"); q != "" {
				serveSearch(w, dir, q)
				return
			}
			serveIndex(w, dir)
			return
		}
		// Resolve and validate the requested path stays within the serve directory
		cleaned := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if !strings.HasPrefix(cleaned, dir) {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, cleaned)
	})

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Fprintf(stderr, "Serving crawled content from %s at http://%s\n", dir, addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func serveIndex(w http.ResponseWriter, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "failed to read directory", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Crawled Sites</title></head><body>`)
	fmt.Fprint(w, `<h1>Crawled Sites</h1>`)
	fmt.Fprint(w, `<form action="/" method="get"><input name="q" placeholder="Search content..." size="40"> <button type="submit">Search</button></form>`)
	fmt.Fprint(w, `<ul>`)
	for _, e := range entries {
		if e.IsDir() {
			name := html.EscapeString(e.Name())
			fmt.Fprintf(w, `<li><a href="/%s/">%s</a></li>`, name, name)
		}
	}
	fmt.Fprint(w, `</ul></body></html>`)
}

func serveSearch(w http.ResponseWriter, dir, query string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Search Results</title></head><body>`)
	fmt.Fprintf(w, `<h1>Results for "%s"</h1>`, html.EscapeString(query))
	fmt.Fprint(w, `<ul>`)

	lowerQuery := strings.ToLower(query)
	count := 0
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || count >= 50 {
			if count >= 50 {
				return filepath.SkipAll
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".htm" {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil || info.Size() > 5*1024*1024 {
			return nil // skip files we can't stat or larger than 5MB
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(strings.ToLower(string(data)), lowerQuery) {
			rel, _ := filepath.Rel(dir, path)
			escaped := html.EscapeString(rel)
			fmt.Fprintf(w, `<li><a href="/%s">%s</a></li>`, escaped, escaped)
			count++
		}
		return nil
	})

	if count == 0 {
		fmt.Fprint(w, `<li>No results found</li>`)
	}
	fmt.Fprint(w, `</ul>`)
	fmt.Fprintf(w, `<p><a href="/">Back to index</a></p></body></html>`)
}
