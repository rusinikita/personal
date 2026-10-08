package webui

import (
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
// <pre><code>.
type markdownRenderer struct {
	*blackfriday.HTMLRenderer
}

func (r *markdownRenderer) RenderNode(w io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
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

// RenderMarkdown renders markdown to HTML. A fenced "mermaid" code block
// becomes a diagram block, drawn in the browser by Mermaid. With safe set,
// raw HTML in the source is dropped and links are limited to safe protocols.
func RenderMarkdown(source string, safe bool) template.HTML {
	flags := blackfriday.CommonHTMLFlags
	if safe {
		flags |= blackfriday.SkipHTML | blackfriday.Safelink
	}
	renderer := &markdownRenderer{blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{Flags: flags})}
	html := blackfriday.Run([]byte(source), blackfriday.WithExtensions(blackfriday.CommonExtensions), blackfriday.WithRenderer(renderer))
	return template.HTML(html)
}
