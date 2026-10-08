package tests

// Covers webui.RenderMarkdown, webui.RenderMarkdownWithHeadings and the
// shell's conditional Mermaid loading
// from docs/functions/webui-spec.md.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/webui"
)

const (
	mermaidScriptSrc = `src="https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"`
	mermaidInit      = "mermaid.initialize("
)

func TestRenderPage_MermaidLoadedForDiagramBlock(t *testing.T) {
	var buf bytes.Buffer
	err := webui.RenderPage(&buf, webui.PageData{
		Title:   "Test Page",
		Content: webui.RenderMarkdown("```mermaid\ngraph LR\n    A --> B\n```\n", true),
	})
	require.NoError(t, err)

	body := buf.String()
	assert.Contains(t, body, mermaidScriptSrc)
	assert.Contains(t, body, mermaidInit)
	assert.Contains(t, body, `securityLevel: "strict"`)
	assert.Contains(t, body, "prefers-color-scheme: dark")
	// the anchor is scrolled to again after the diagrams are drawn
	assert.Contains(t, body, "mermaid.run().then(scrollToAnchor, scrollToAnchor)")
	assert.Contains(t, body, "location.hash")
	assert.Contains(t, body, "scrollIntoView()")
}

func TestRenderPage_MermaidNotLoadedWithoutDiagramBlock(t *testing.T) {
	var buf bytes.Buffer
	err := webui.RenderPage(&buf, webui.PageData{
		Title:   "Test Page",
		Content: webui.RenderMarkdown("```go\nfmt.Println(1)\n```\n\nmermaid is only a word here\n", true),
	})
	require.NoError(t, err)

	body := buf.String()
	assert.NotContains(t, body, mermaidScriptSrc)
	assert.NotContains(t, body, mermaidInit)
}

func TestRenderMarkdown_RendersCommonMarkdown(t *testing.T) {
	html := string(webui.RenderMarkdown("# Heading\n\n- one\n- two\n\n[link](https://example.com)\n", true))

	// headings have no ids
	assert.Contains(t, html, "<h1>Heading</h1>")
	assert.Contains(t, html, "<li>one</li>")
	assert.Contains(t, html, `<a href="https://example.com">link</a>`)
}

func TestRenderMarkdown_CodeBlocks(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:            "mermaid block becomes a diagram block with escaped source",
			source:          "```mermaid\ngraph LR\n    A[\"<b>x</b>\"] --> B\n```\n",
			wantContains:    []string{"<pre class=\"mermaid\">graph LR\n    A[&#34;&lt;b&gt;x&lt;/b&gt;&#34;] --&gt; B\n</pre>"},
			wantNotContains: []string{"<code", "<b>x</b>"},
		},
		{
			name:            "block of another language stays a code block",
			source:          "```go\ngraph LR\n```\n",
			wantContains:    []string{`<pre><code class="language-go">graph LR`},
			wantNotContains: []string{`class="mermaid"`},
		},
		{
			name:            "block with no language stays a code block",
			source:          "```\ngraph LR\n```\n",
			wantContains:    []string{"<pre><code>graph LR"},
			wantNotContains: []string{`class="mermaid"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(webui.RenderMarkdown(tt.source, true))

			for _, want := range tt.wantContains {
				assert.Contains(t, html, want)
			}
			for _, notWant := range tt.wantNotContains {
				assert.NotContains(t, html, notWant)
			}
		})
	}
}

func TestRenderMarkdown_SafeMode(t *testing.T) {
	source := "<script>alert(1)</script>\n\n<pre class=\"mermaid\">raw</pre>\n\n[bad](javascript:alert(1))\n\n```mermaid\ngraph LR\n```\n"

	t.Run("safe drops raw HTML and unsafe links, also next to a diagram", func(t *testing.T) {
		html := string(webui.RenderMarkdown(source, true))

		assert.NotContains(t, html, "<script>")
		assert.NotContains(t, html, "raw</pre>")
		assert.NotContains(t, html, `href="javascript:`)
		assert.Contains(t, html, "<pre class=\"mermaid\">graph LR\n</pre>")
	})

	t.Run("not safe keeps raw HTML", func(t *testing.T) {
		html := string(webui.RenderMarkdown(source, false))

		assert.Contains(t, html, "<script>alert(1)</script>")
		assert.Contains(t, html, "<pre class=\"mermaid\">graph LR\n</pre>")
	})
}

func TestRenderMarkdownWithHeadings_IDsAndList(t *testing.T) {
	source := "# One\n\ntext\n\n## Two *em* `code` [link](https://example.com)\n\n```\n# in code\n```\n\n### Custom {#mine}\n\n# One\n"

	html, headings := webui.RenderMarkdownWithHeadings(source, true, "doc-")

	assert.Equal(t, []webui.MarkdownHeading{
		{Level: 1, Text: "One", ID: "doc-h1"},
		{Level: 2, Text: "Two em code link", ID: "doc-h2"},
		{Level: 3, Text: "Custom", ID: "doc-h3"},
		{Level: 1, Text: "One", ID: "doc-h4"},
	}, headings)

	out := string(html)
	assert.Contains(t, out, `<h1 id="doc-h1">One</h1>`)
	assert.Contains(t, out, `<h2 id="doc-h2">Two <em>em</em> <code>code</code> <a href="https://example.com">link</a></h2>`)
	assert.Contains(t, out, `<h3 id="doc-h3">Custom</h3>`)
	assert.Contains(t, out, `<h1 id="doc-h4">One</h1>`)
	assert.NotContains(t, out, "mine")
	assert.Contains(t, out, "<pre><code># in code\n</code></pre>")
}

func TestRenderMarkdownWithHeadings_NoHeadings(t *testing.T) {
	html, headings := webui.RenderMarkdownWithHeadings("just text\n\n```\n# in code\n```\n", true, "doc-")

	assert.Empty(t, headings)
	assert.Equal(t, webui.RenderMarkdown("just text\n\n```\n# in code\n```\n", true), html)
}

func TestRenderMarkdownWithHeadings_SameHTMLBesidesIDs(t *testing.T) {
	source := "# One\n\n<script>alert(1)</script>\n\n[bad](javascript:alert(1))\n\n## Two\n\n```mermaid\ngraph LR\n```\n"

	for _, safe := range []bool{true, false} {
		html, headings := webui.RenderMarkdownWithHeadings(source, safe, "p-")

		require.Len(t, headings, 2)
		withoutIDs := string(html)
		for _, heading := range headings {
			withoutIDs = strings.Replace(withoutIDs, ` id="`+heading.ID+`"`, "", 1)
		}
		assert.Equal(t, string(webui.RenderMarkdown(source, safe)), withoutIDs)
	}
}
