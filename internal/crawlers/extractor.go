package crawlers

import (
	"bytes"
	"strings"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

// compiledSelector holds a pre-compiled CSS selector with optional attribute extraction.
type compiledSelector struct {
	key      string
	selector cascadia.Selector
	attrName string // non-empty means extract attribute instead of text
}

// CompileSelectors pre-compiles CSS selector strings into reusable selectors.
// Invalid selectors are silently skipped. Call once at startup, pass to ExtractFields.
func CompileSelectors(selectors map[string]string) []compiledSelector {
	compiled := make([]compiledSelector, 0, len(selectors))
	for key, rawSel := range selectors {
		sel, attrName := parseAttrSuffix(rawSel)
		cs, err := cascadia.Compile(sel)
		if err != nil {
			continue
		}
		compiled = append(compiled, compiledSelector{
			key:      key,
			selector: cs,
			attrName: attrName,
		})
	}
	return compiled
}

// ExtractFields parses HTML content and extracts text or attribute values
// using pre-compiled CSS selectors.
func ExtractFields(content []byte, selectors []compiledSelector) map[string]string {
	if len(content) == 0 || len(selectors) == 0 {
		return nil
	}

	doc, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return nil
	}

	result := make(map[string]string, len(selectors))
	for _, cs := range selectors {
		node := cascadia.Query(doc, cs.selector)
		if node == nil {
			continue
		}

		if cs.attrName != "" {
			result[cs.key] = getAttr(node, cs.attrName)
		} else {
			result[cs.key] = textContent(node)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// parseAttrSuffix splits "selector::attr(name)" into ("selector", "name").
func parseAttrSuffix(sel string) (string, string) {
	const prefix = "::attr("
	idx := strings.LastIndex(sel, prefix)
	if idx < 0 {
		return sel, ""
	}
	rest := sel[idx+len(prefix):]
	if !strings.HasSuffix(rest, ")") {
		return sel, ""
	}
	return sel[:idx], rest[:len(rest)-1]
}

func getAttr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return strings.TrimSpace(sb.String())
}
