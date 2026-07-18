#!/bin/bash

# Test script to verify global crawler usage
echo "Testing crawler global usage..."
echo "================================"

# Show where crawler will run from
echo "Crawler location: $(which crawler)"
echo "Current directory: $(pwd)"
echo ""

# Test help command
echo "1. Testing help command:"
crawler --help
echo ""

# Show where files will be saved
echo "2. Default output locations:"
echo "Output directory: ${CRAWLER_OUTPUT_DIR:-$TMPDIR/crawler-output}"
echo "Cache directory: ${CRAWLER_CACHE_DIR:-$HOME/Library/Caches/crawler}"
echo "Config search paths:"
echo "  - ./crawl.yml (current directory)"
echo "  - ${CRAWLER_CONFIG:-not set}"
echo "  - $HOME/Library/Application Support/crawler/crawl.yml"
echo ""

# Test verbose mode with small crawl
echo "3. Testing verbose mode (should show logs, not UI):"
crawler -v --depth 1 https://example.com