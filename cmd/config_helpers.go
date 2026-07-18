package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dotcommander/crawler/api"
	"github.com/dotcommander/crawler/internal/config"
)

// loadAndBuildConfig handles all configuration loading and building logic using Viper
func loadAndBuildConfig(configFile, startURL, outputDir string, profile string, mobile bool, maxPages int) (*config.CrawlerConfig, error) {
	// Find config file if not specified
	if configFile == "" {
		configFile = findConfigFile()
	}

	// Set default output directory if not specified
	if outputDir == "" {
		outputDir = config.GetDefaultOutputDir()
	}

	// Use new Viper-based configuration system
	return config.LoadConfigWithViper(configFile, startURL, outputDir, profile, mobile, maxPages)
}

// setupLogging configures the logging based on verbose mode
func setupLogging(verbose, quiet bool) {
	if quiet {
		log.SetOutput(io.Discard)
		return
	}
	if verbose {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	} else {
		log.SetFlags(log.Ldate | log.Ltime)
	}
}

// runCrawlerWithSignalHandling executes the crawler with proper signal handling
func runCrawlerWithSignalHandling(crawler api.Crawler) error {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run crawler in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- crawler.Start()
	}()

	// Wait for either completion or interrupt
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("crawler failed: %w", err)
		}
		return nil
	case <-sigChan:
		log.Println("\nShutdown signal received. Gracefully stopping...")
		crawler.Cancel()

		// Wait for crawler to finish with timeout
		t := time.NewTimer(5 * time.Second)
		defer t.Stop()
		select {
		case <-done:
			log.Println("Crawler stopped gracefully")
		case <-t.C:
			log.Println("Shutdown timeout exceeded")
		}
		return nil
	}
}
