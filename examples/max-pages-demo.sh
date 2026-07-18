#!/bin/bash

# Demo script showing max-pages functionality

echo "=== Max Pages Demo ==="
echo "This demo shows how to limit the number of pages crawled"
echo

# Example 1: Crawl only the homepage
echo "1. Crawling only the homepage (1 page):"
echo "   ./crawler --max-pages 1 https://example.com"
echo

# Example 2: Crawl homepage and immediate links
echo "2. Crawling up to 5 pages:"
echo "   ./crawler --max-pages 5 --depth 1 https://example.com"
echo

# Example 3: Mobile crawl with page limit
echo "3. Mobile crawl limited to 3 pages:"
echo "   ./crawler --mobile --max-pages 3 https://example.com"
echo

# Example 4: Compare single page on mobile
echo "4. Compare single page between www and dev (mobile):"
echo "   ./cmd/compare-page-crawl/compare-page-crawl --mobile https://example.com/page"
echo

echo "Note: --max-pages limits the TOTAL number of pages crawled,"
echo "      while --depth limits how many links deep to follow."