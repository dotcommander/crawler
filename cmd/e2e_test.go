package cmd_test

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// crawlerBin is built once for all e2e tests in this package.
var crawlerBin string

func TestMain(m *testing.M) {
	bin, err := os.CreateTemp("", "crawler-e2e-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e: temp binary path:", err)
		os.Exit(1)
	}
	bin.Close()
	build := exec.Command("go", "build", "-o", bin.Name(), ".")
	build.Dir = repoRoot()
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: build failed: %v\n%s", err, out)
		os.Exit(1)
	}
	crawlerBin = bin.Name()
	code := m.Run()
	os.Remove(crawlerBin)
	os.Exit(code)
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return filepath.Dir(wd)
}

// e2eSite serves a small site with a robots.txt that disallows /private.
func e2eSite(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /private\n"))
	})
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`))
	})
	pages := map[string]string{
		"/":    `<html><head><title>Home</title></head><body><h1>hello</h1><a href="/sub">sub</a> <a href="/private">private</a></body></html>`,
		"/sub": `<html><head><title>Sub</title></head><body><h1>sub</h1></body></html>`,
	}
	for path, body := range pages {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(body))
		})
	}
	site := httptest.NewServer(mux)
	t.Cleanup(site.Close)
	return site
}

// runCrawler executes the built binary with an isolated sessions dir and
// returns combined output.
func runCrawler(t *testing.T, sessionsDir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(crawlerBin, args...)
	// Keep the katana queue's minimum lifetime short so e2e runs do not
	// each pay the 10s default floor.
	cmd.Env = append(os.Environ(),
		"CRAWLER_SESSIONS_DIR="+sessionsDir,
		"CRAWLER_ENGINETIMEOUTSECONDS=2",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("crawler %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestE2EQuietJSONLSchemaAndRobots(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: binary e2e")
	}
	site := e2eSite(t)
	sessions := t.TempDir()
	output := t.TempDir()

	out := runCrawler(t, sessions, "--quiet", "--format", "jsonl", "--output", output, site.URL+"/")

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var urls []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		var rec struct {
			URL         string `json:"url"`
			Title       string `json:"title"`
			StatusCode  int    `json:"status_code"`
			ContentType string `json:"content_type"`
			LinksFound  int    `json:"links_found"`
			CrawledAt   string `json:"crawled_at"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("record %q is not valid schema JSON: %v", line, err)
		}
		if rec.CrawledAt == "" {
			t.Errorf("record %q missing crawled_at", line)
		}
		urls = append(urls, rec.URL)
	}

	want := []string{site.URL + "/", site.URL + "/sub"}
	for _, w := range want {
		found := false
		for _, u := range urls {
			if u == w {
				found = true
			}
		}
		if !found {
			t.Errorf("exported URLs %v missing %s", urls, w)
		}
	}
	for _, u := range urls {
		if strings.Contains(u, "/private") {
			t.Errorf("robots-disallowed /private exported: %v", urls)
		}
	}
}

func TestE2ECSVAndSitemapExports(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: binary e2e")
	}
	site := e2eSite(t)
	output := t.TempDir()

	// Fresh sessions dir per run: the session store dedups visited pages,
	// so a shared session would export nothing on the second format run.
	csvOut := runCrawler(t, t.TempDir(), "--quiet", "--format", "csv", "--output", output, site.URL+"/")
	csvLines := strings.Split(strings.TrimSpace(csvOut), "\n")
	if len(csvLines) < 2 {
		t.Fatalf("csv output too short: %q", csvOut)
	}
	if !strings.Contains(csvLines[0], "url") {
		t.Errorf("csv header %q missing url column", csvLines[0])
	}

	// Sitemap exports buffer until Close and therefore require an export
	// file (legacy CLI contract).
	smFile := filepath.Join(t.TempDir(), "sitemap.xml")
	runCrawler(t, t.TempDir(), "--quiet", "--format", "sitemap", "--export-file", smFile, "--output", output, site.URL+"/")
	smBytes, err := os.ReadFile(smFile)
	if err != nil {
		t.Fatalf("read sitemap export: %v", err)
	}
	smOut := string(smBytes)
	var urlset struct {
		XMLName xml.Name `xml:"urlset"`
		URLs    []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal([]byte(smOut), &urlset); err != nil {
		t.Fatalf("sitemap output not valid XML: %v\n%s", err, smOut)
	}
	if len(urlset.URLs) < 2 {
		t.Errorf("sitemap has %d <url> entries, want at least 2:\n%s", len(urlset.URLs), smOut)
	}
}

func TestE2EServeBrowsesStoredSession(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: binary e2e")
	}
	site := e2eSite(t)
	sessions := t.TempDir()
	output := t.TempDir()
	runCrawler(t, sessions, "--quiet", "--output", output, site.URL+"/")

	port := freePort(t)
	serve := exec.Command(crawlerBin, "serve", output, "--port", fmt.Sprintf("%d", port), "--host", "127.0.0.1")
	serve.Env = append(os.Environ(), "CRAWLER_SESSIONS_DIR="+sessions)
	if err := serve.Start(); err != nil {
		t.Fatalf("serve start: %v", err)
	}
	t.Cleanup(func() { _ = serve.Process.Kill() })

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	var resp *http.Response
	var err error
	for i := 0; i < 50; i++ {
		resp, err = http.Get(base + "/")
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("serve never came up on %s: %v", base, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("serve / status = %d, want 200", resp.StatusCode)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
