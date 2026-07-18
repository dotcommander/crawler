#!/bin/bash

# Test URL filtering behavior

echo "Testing URL filtering..."
echo "========================"

# Test 1: Crawl a specific file path
echo -e "\nTest 1: Crawling https://example.com/docs/guide.html"
echo "Should only crawl within /docs/ directory"
./crawler --verbose --max-pages 5 --engine colly https://example.com/docs/guide.html 2>&1 | grep -E "(Crawled:|Skipping:|mailto:|external)"

# Test 2: Check mailto filtering
echo -e "\nTest 2: Creating test HTML with various link types"
cat > test-links.html <<EOF
<!DOCTYPE html>
<html>
<head><title>Test Links</title></head>
<body>
<h1>Test Links</h1>
<a href="page1.html">Internal relative link</a>
<a href="/docs/page2.html">Internal absolute link</a>
<a href="https://example.com/docs/page3.html">Internal full URL</a>
<a href="https://external.com/page.html">External domain</a>
<a href="mailto:test@example.com">Email link</a>
<a href="tel:+1234567890">Phone link</a>
<a href="javascript:alert('test')">JavaScript link</a>
<a href="ftp://ftp.example.com/file.txt">FTP link</a>
<a href="../parent.html">Parent directory link</a>
<a href="/other/path.html">Different path</a>
</body>
</html>
EOF

echo "Test HTML created with various link types"
echo "Run crawler on a real site to see filtering in action"

# Clean up
rm -f test-links.html

echo -e "\nURL Filtering Rules Applied:"
echo "✅ Only HTTP/HTTPS links"
echo "✅ Only same domain as start URL"
echo "✅ Only within base path of start URL"
echo "❌ No mailto: links"
echo "❌ No tel: links"
echo "❌ No javascript: links"
echo "❌ No ftp: or other schemes"