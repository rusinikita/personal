package webui

import (
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/russross/blackfriday/v2"
)

// mermaidBlockOpen opens a diagram block. RenderPage looks for it in the
// page content to decide whether the shell loads Mermaid.
const mermaidBlockOpen = `<pre class="mermaid">`

const mermaidLanguage = "mermaid"

// markdownRenderer is blackfriday's HTML renderer, except that a fenced code
// block with the language "mermaid" becomes a diagram block instead of
// <pre><code>. With headingIDs set it also numbers the headings: each one
// gets the id "h{n}" (the renderer adds its prefix) and is collected in
// headings.
type markdownRenderer struct {
	*blackfriday.HTMLRenderer
	headingIDs bool
	idPrefix   string
	headings   []MarkdownHeading
}

func (r *markdownRenderer) RenderNode(w io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	if r.headingIDs && node.Type == blackfriday.Heading && entering {
		node.HeadingID = fmt.Sprintf("h%d", len(r.headings)+1)
		r.headings = append(r.headings, MarkdownHeading{
			Level: node.Level,
			Text:  nodeText(node),
			ID:    r.idPrefix + node.HeadingID,
		})
	}
	if node.Type != blackfriday.CodeBlock || !isMermaidInfo(node.Info) {
		return r.HTMLRenderer.RenderNode(w, node, entering)
	}
	io.WriteString(w, mermaidBlockOpen)
	template.HTMLEscape(w, node.Literal)
	io.WriteString(w, "</pre>\n")
	return blackfriday.GoToNext
}

func isMermaidInfo(info []byte) bool {
	fields := strings.Fields(string(info))
	return len(fields) > 0 && fields[0] == mermaidLanguage
}

// nodeText is the text of a node without inline markup.
func nodeText(node *blackfriday.Node) string {
	var b strings.Builder
	node.Walk(func(n *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		if entering && (n.Type == blackfriday.Text || n.Type == blackfriday.Code) {
			b.Write(n.Literal)
		}
		return blackfriday.GoToNext
	})
	return b.String()
}

// RenderMarkdown renders markdown to HTML. A fenced "mermaid" code block
// becomes a diagram block, drawn in the browser by Mermaid. With safe set,
// raw HTML in the source is dropped and links are limited to safe protocols.
func RenderMarkdown(source string, safe bool) template.HTML {
	return runMarkdown(source, &markdownRenderer{HTMLRenderer: newHTMLRenderer(safe, "")})
}

// RenderMarkdownWithHeadings renders markdown as RenderMarkdown does and
// gives every heading the id "{idPrefix}h{n}", n being its 1-based number in
// the source. It returns the headings in source order, for a table of
// contents. idPrefix must be unique on the page.
func RenderMarkdownWithHeadings(source string, safe bool, idPrefix string) (template.HTML, []MarkdownHeading) {
	renderer := &markdownRenderer{HTMLRenderer: newHTMLRenderer(safe, idPrefix), headingIDs: true, idPrefix: idPrefix}
	html := runMarkdown(source, renderer)
	return html, renderer.headings
}

func newHTMLRenderer(safe bool, headingIDPrefix string) *blackfriday.HTMLRenderer {
	flags := blackfriday.CommonHTMLFlags
	if safe {
		flags |= blackfriday.SkipHTML | blackfriday.Safelink
	}
	return blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{Flags: flags, HeadingIDPrefix: headingIDPrefix})
}

func runMarkdown(source string, renderer blackfriday.Renderer) template.HTML {
	html := blackfriday.Run([]byte(source), blackfriday.WithExtensions(blackfriday.CommonExtensions), blackfriday.WithRenderer(renderer))
	return template.HTML(html)
}
