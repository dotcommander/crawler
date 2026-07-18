#!/bin/bash

# Simple demo of the new crawler interface

echo "=== Crawler Simple Interface Demo ==="
echo "This demo shows the new simplified CLI in action"
echo

# Example 1: Basic crawl with defaults
echo "1. Basic crawl (smart defaults):"
echo "   ./crawler https://example.com"
echo "   → Crawls with depth=3, concurrency=5, 1s delays"
echo

# Example 2: Limit pages for safety
echo "2. Limited crawl (safety first):"
echo "   ./crawler --max-pages 20 https://large-site.com"
echo "   → Stops after 20 pages to avoid runaway crawls"
echo

# Example 3: Mobile crawling
echo "3. Mobile view:"
echo "   ./crawler --mobile https://responsive-site.com"
echo "   → Uses iPhone 14 viewport and user agent"
echo

# Example 4: Using profiles
echo "4. Profile presets:"
echo "   ./crawler --profile fast https://robust-site.com"
echo "   → 10 workers, 0.5s delays, depth 2"
echo
echo "   ./crawler --profile safe https://fragile-site.com"
echo "   → 2 workers, 2s delays, depth 3"
echo
echo "   ./crawler --profile thorough https://docs-site.com"
echo "   → 5 workers, depth 5, more retries"
echo

# Example 5: Advanced configuration
echo "5. Advanced users:"
echo "   cp crawl.yml.example crawl.yml"
echo "   # Edit crawl.yml with your settings"
echo "   ./crawler https://complex-site.com"
echo

echo "Tips:"
echo "- Use --verbose (-v) to debug issues"
echo "- The crawler auto-detects JavaScript sites"
echo "- Check crawl.yml.example for all options"
echo "- Output goes to ~/.config/crawler/storage by default"