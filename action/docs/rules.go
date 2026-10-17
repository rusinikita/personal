package docs

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// maxChunkLines is the most lines a chunk's body can have.
	maxChunkLines = 60
	// maxLineLength is the most characters (runes) in a body line and in a
	// document description.
	maxLineLength = 200
	// appendSeparator goes between the existing body and appended text.
	appendSeparator = "\n\n"
)

// normalizeBody turns the CRLF line endings of a web form into LF.
func normalizeBody(body string) string {
	return strings.ReplaceAll(body, "\r\n", "\n")
}

// validateBody checks the chunk limits on an already normalized body.
func validateBody(body string) error {
	lines := strings.Split(body, "\n")
	if len(lines) > maxChunkLines {
		return fmt.Errorf("body has %d lines, maximum is %d: split it into two chunks", len(lines), maxChunkLines)
	}
	for i, line := range lines {
		if n := utf8.RuneCountInString(line); n > maxLineLength {
			return fmt.Errorf("line %d of body has %d characters, maximum is %d", i+1, n, maxLineLength)
		}
	}
	return nil
}

func validateTitle(title string) error {
	if title == "" {
		return fmt.Errorf("title is required")
	}
	return nil
}

func validateDescription(description string) error {
	if description == "" {
		return fmt.Errorf("description is required")
	}
	if strings.ContainsAny(description, "\r\n") {
		return fmt.Errorf("description must be one line")
	}
	if n := utf8.RuneCountInString(description); n > maxLineLength {
		return fmt.Errorf("description has %d characters, maximum is %d", n, maxLineLength)
	}
	return nil
}
