package utils

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)
var newlineSpaceRe = regexp.MustCompile(`\s+`)

// bluemonday policy: strict + allow safe formatting tags for rendered Markdown
var sanitizer = bluemonday.UGCPolicy().
	AllowAttrs("class", "id").OnElements("pre", "code", "span", "div", "table", "th", "td", "tr", "blockquote").
	AllowAttrs("href", "target", "rel").OnElements("a").
	AllowAttrs("src", "alt", "title", "width", "height", "loading").OnElements("img").
	AllowAttrs("align").OnElements("img", "table", "th", "td")

// Goldmark renderer: use WithXHTML instead of WithUnsafe to prevent raw HTML injection
var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(html.WithXHTML()),
)

// RenderMarkdown renders Markdown content to safe HTML.
// Goldmark is configured to NOT allow raw HTML (XHTML mode).
// Bluemonday provides an additional sanitization layer for defense-in-depth.
func RenderMarkdown(content string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		return escapeHTML(content)
	}
	return sanitizer.Sanitize(buf.String())
}

// escapeHTML provides a fallback when Markdown rendering fails
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// SanitizeHTML applies bluemonday sanitization to raw HTML content.
// Used for user-submitted content that may contain HTML (e.g. comments).
func SanitizeHTML(input string) string {
	return sanitizer.Sanitize(input)
}

// ExcerptFromContent extracts plain text from Markdown content.
// It renders the Markdown to HTML first, then strips all HTML tags
// and collapses whitespace, returning clean readable text.
func ExcerptFromContent(content string, length int) string {
	// Render Markdown to HTML (already sanitized via RenderMarkdown)
	rendered := RenderMarkdown(content)

	// Strip all HTML tags
	plain := htmlTagRe.ReplaceAllString(rendered, "")

	// Collapse newlines and multiple spaces
	plain = newlineSpaceRe.ReplaceAllString(plain, " ")
	plain = strings.TrimSpace(plain)

	runes := []rune(plain)
	if len(runes) == 0 {
		return ""
	}
	if length <= 0 || len(runes) <= length {
		return plain
	}
	return string(runes[:length]) + "..."
}

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "?", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "（", "")
	s = strings.ReplaceAll(s, "）", "")
	s = strings.ReplaceAll(s, "，", "")
	s = strings.ReplaceAll(s, "。", "")
	s = strings.ReplaceAll(s, "、", "")
	s = strings.ReplaceAll(s, "！", "")
	s = strings.ReplaceAll(s, "？", "")
	return s
}
