# Error Solutions

## Error: "Shutdown signal received. Gracefully stopping..." but crawler hangs

**Cause**: Workers were blocked on channel operations when context was cancelled

**Solution**: 
This has been fixed in the codebase. If using an older version:
```bash
# Update to latest code
git pull

# Rebuild
go build -o crawler .
```

**Prevention**: Always use latest version. The fix adds proper shutdown timeout and queue draining.

## Error: "failed to create output directory: permission denied"

**Cause**: Insufficient permissions for `~/.config/crawler/storage`

**Solution**:
```bash
# Fix permissions
chmod -R 755 ~/.config/crawler/

# Or use a different output directory
./crawler --output /tmp/crawl-output https://example.com
```

**Prevention**: Ensure your user has write access to home directory

## Error: "failed to launch browser: could not start playwright"

**Cause**: Playwright browsers not installed

**Solution**:
```bash
# Install Playwright browsers
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install

# Or install specific browser
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
```

**Prevention**: Add to your setup script or README

## Error: Context timeout when crawling JavaScript-heavy sites

**Cause**: Page takes too long to load/render

**Solution**:
```bash
# Use longer timeout in config
cat > crawl.yml <<EOF
extraWaitTime: 5000  # 5 seconds extra wait
waitStrategy: "networkidle"
EOF

# Or force Playwright engine
./crawler --engine playwright https://spa-site.com
```

**Prevention**: Use appropriate engine for site type (Playwright for JS-heavy)

## Error: "too many open files"

**Cause**: System file descriptor limit too low

**Solution**:
```bash
# Check current limit
ulimit -n

# Increase limit (temporary)
ulimit -n 4096

# For permanent fix, edit /etc/security/limits.conf
# Add: * soft nofile 4096
```

**Prevention**: Set reasonable concurrency limit in crawler config

## Error: Crawler saves files to wrong location

**Cause**: Using old version that defaults to `./tmp`

**Solution**:
```bash
# Check where files are being saved
find . -name "*.html" -mtime -1

# Default storage is ~/.config/crawler/storage (override with CRAWLER_OUTPUT_DIR)

# Verify new location is used
./crawler --verbose https://example.com 2>&1 | grep "Saved:"
```

**Prevention**: Always use `~/.config/crawler/storage` or set `CRAWLER_OUTPUT_DIR`

## Error: "panic: send on closed channel"

**Cause**: Race condition in shutdown sequence

**Solution**:
```bash
# This is fixed in latest version
go build -o crawler .
```

**Prevention**: The fix uses sync.Once to prevent double-closing channels

## Error: Crawling external domains when shouldn't

**Cause**: URL filtering not properly configured

**Solution**: Update to latest version which includes:
- Domain restriction (same host only)
- Path restriction (stays within base path)
- Scheme filtering (HTTP/HTTPS only)

**Prevention**: URL filtering is now automatic in the engine crawler

## Error: Circuit breaker blocking domain

**Cause**: Multiple failures from a domain trigger circuit breaker

**Solution**:
```bash
# Clear cache to reset circuit breaker state
rm -rf ~/Library/Caches/crawler/  # macOS
rm -rf ~/.cache/crawler/          # Linux

# Or wait 5 minutes for circuit breaker to reset
```

**Prevention**: Fix underlying issues causing failures (timeouts, 404s)

## Error: No space left on device

**Cause**: Disk full from crawled content

**Solution**:
```bash
# Check disk usage
du -sh ~/.config/crawler/storage/

# Remove old crawls
rm -rf ~/.config/crawler/storage/old-domain.com/

# Or move to external drive
mv ~/.config/crawler/storage /Volumes/External/crawler-storage
ln -s /Volumes/External/crawler-storage ~/.config/crawler/storage
```

**Prevention**: Monitor disk space, use --max-pages limit

<!-- Last updated: 2024-07-14 -->
<!-- Source: Bug fixes and implementation experience -->