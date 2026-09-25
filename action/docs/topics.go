// Package docs serves hand-written convention documents embedded from
// content/*.md: a read-only web page per doc (GET /web/docs, GET
// /web/docs/:topic) and the list_docs/get_doc MCP tools. The topic list is
// discovered from the embedded directory, not hardcoded — adding a doc is
// just adding a file.
package docs

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

//go:embed content/*.md
var contentFS embed.FS

// Topic describes one discovered doc: its slug (content filename minus
// ".md"), title (the file's first "# " heading), and one-line description
// (the first blockquote line following the heading).
type Topic struct {
	Slug        string `json:"slug" jsonschema:"Doc slug, pass it to get_doc"`
	Title       string `json:"title" jsonschema:"Doc title"`
	Description string `json:"description" jsonschema:"One-line summary of what the doc covers"`
}

// Topics returns the current topic list, derived from the embedded content
// directory and sorted by slug.
func Topics() ([]Topic, error) {
	entries, err := fs.ReadDir(contentFS, "content")
	if err != nil {
		return nil, err
	}
	topics := make([]Topic, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".md" {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		content, err := Content(slug)
		if err != nil {
			return nil, err
		}
		title, description := parseHeader(content)
		topics = append(topics, Topic{Slug: slug, Title: title, Description: description})
	}
	return topics, nil
}

// Content returns the raw markdown of the doc with the given slug. An
// unknown slug is an error.
func Content(slug string) ([]byte, error) {
	if slug == "" || strings.ContainsAny(slug, "/\\.") {
		return nil, fmt.Errorf("unknown doc topic %q", slug)
	}
	b, err := contentFS.ReadFile("content/" + slug + ".md")
	if err != nil {
		return nil, fmt.Errorf("unknown doc topic %q", slug)
	}
	return b, nil
}

// parseHeader extracts the first "# " heading as the title and the first
// "> " blockquote line after it as the description.
func parseHeader(content []byte) (title, description string) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if title == "" {
			if strings.HasPrefix(line, "# ") {
				title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			}
			continue
		}
		if strings.HasPrefix(line, ">") {
			if d := strings.TrimSpace(strings.TrimPrefix(line, ">")); d != "" {
				return title, d
			}
		}
	}
	return title, description
}
