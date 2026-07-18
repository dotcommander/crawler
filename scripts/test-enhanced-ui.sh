#!/bin/bash

# Test script for the enhanced Bubbletea UI (now default!)

echo "🛸 Testing Crawler Enhanced UI (Default)"
echo "========================================"
echo ""
echo "The enhanced UI is now the default interface!"
echo "This script will demonstrate the new beautiful TUI."
echo ""

# Build the crawler first
echo "Building crawler..."
go build -o crawler .

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "✅ Build successful!"
echo ""

# Create a test output directory
TEST_DIR="./test-crawl-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$TEST_DIR"

echo "📁 Output directory: $TEST_DIR"
echo ""
echo "Starting crawler with enhanced UI (default)..."
echo "Press '?' for help, 'q' to quit"
echo ""
echo "----------------------------------------"

# Run the crawler with a simple test site
# No environment variables needed - enhanced UI is default!
./crawler \
    --output "$TEST_DIR" \
    --depth 2 \
    --concurrency 3 \
    --delay 1 \
    "https://example.com"

# Check exit status
if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Crawl completed successfully!"
    echo "📊 Results saved to: $TEST_DIR"
else
    echo ""
    echo "❌ Crawl failed or was interrupted"
fi

echo ""
echo "To test other UI modes:"
echo "  - Verbose mode: ./crawler -v https://example.com"
echo "  - Legacy UI: CRAWLER_LEGACY_UI=1 ./crawler https://example.com"