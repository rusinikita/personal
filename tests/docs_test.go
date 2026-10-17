package tests

// Covers the MCP tools from docs/functions/docs-spec.md:
// list_docs, get_doc, create_doc,
// edit_doc, create_doc_chunk, edit_doc_chunk,
// revert_doc_chunk, delete_doc_chunk.

import (
	"context"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/docs"
	"personal/domain"
	"personal/gateways"
	"personal/util"
)

const (
	docMaxLines      = 60
	docMaxLineLength = 200
)

// createDoc seeds a document through the repository.
func (s *IntegrationTestSuite) createDoc(ctx context.Context, title string) int64 {
	id, err := s.Repo().CreateDoc(ctx, &domain.Doc{
		UserID:      gateways.UserIDFromContext(ctx),
		Title:       title,
		Description: title + " description",
	})
	require.NoError(s.T(), err)
	return id
}

// createDocChunks seeds chunks at the end of a document through the
// repository and returns their ids in order.
func (s *IntegrationTestSuite) createDocChunks(ctx context.Context, docID int64, bodies ...string) []int64 {
	ids := make([]int64, len(bodies))
	for i, body := range bodies {
		id, err := s.Repo().CreateDocChunk(ctx, &domain.DocChunk{
			UserID: gateways.UserIDFromContext(ctx),
			DocID:  docID,
			Body:   body,
		})
		require.NoError(s.T(), err)
		ids[i] = id
	}
	return ids
}

func (s *IntegrationTestSuite) getDoc(ctx context.Context, docID int64) *domain.Doc {
	doc, err := s.Repo().GetDoc(ctx, docID, gateways.UserIDFromContext(ctx))
	require.NoError(s.T(), err)
	require.NotNil(s.T(), doc)
	return doc
}

func (s *IntegrationTestSuite) getDocChunk(ctx context.Context, chunkID int64) *domain.DocChunk {
	chunk, err := s.Repo().GetDocChunk(ctx, chunkID, gateways.UserIDFromContext(ctx))
	require.NoError(s.T(), err)
	require.NotNil(s.T(), chunk)
	return chunk
}

func (s *IntegrationTestSuite) listDocChunks(ctx context.Context, docID int64) []domain.DocChunk {
	chunks, err := s.Repo().ListDocChunks(ctx, docID, gateways.UserIDFromContext(ctx))
	require.NoError(s.T(), err)
	return chunks
}

// assertDocChunks checks that the document's chunks have exactly
// these bodies in this order, at positions 1..N.
func (s *IntegrationTestSuite) assertDocChunks(ctx context.Context, docID int64, wantBodies ...string) {
	chunks := s.listDocChunks(ctx, docID)
	bodies := make([]string, len(chunks))
	for i, chunk := range chunks {
		bodies[i] = chunk.Body
		assert.Equal(s.T(), i+1, chunk.Position)
	}
	if len(wantBodies) == 0 {
		assert.Empty(s.T(), bodies)
		return
	}
	assert.Equal(s.T(), wantBodies, bodies)
}

func docLines(count int, line string) string {
	lines := make([]string, count)
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

// --- create_doc ---------------------------------------------------

func (s *IntegrationTestSuite) TestCreateDoc() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)

	tests := []struct {
		name        string
		title       string
		description string
		wantErr     string
	}{
		{name: "created with title and description", title: "Development plan", description: "Where I am going"},
		{name: "same title creates a second document", title: "Development plan", description: "Another one"},
		{name: "description of the max length", title: "Long", description: strings.Repeat("я", docMaxLineLength)},
		{name: "empty title fails", title: "  ", description: "d", wantErr: "title is required"},
		{name: "empty description fails", title: "t", description: " ", wantErr: "description is required"},
		{name: "multi-line description fails", title: "t", description: "one\ntwo", wantErr: "description must be one line"},
		{name: "too long description fails", title: "t", description: strings.Repeat("a", docMaxLineLength+1), wantErr: "maximum is 200"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			before, err := s.Repo().ListDocs(ctx, userID)
			require.NoError(s.T(), err)

			_, out, err := docs.CreateDoc(ctx, nil, docs.CreateDocInput{Title: tt.title, Description: tt.description})

			after, listErr := s.Repo().ListDocs(ctx, userID)
			require.NoError(s.T(), listErr)
			if tt.wantErr != "" {
				require.Error(s.T(), err)
				assert.Contains(s.T(), err.Error(), tt.wantErr)
				assert.Len(s.T(), after, len(before))
				return
			}
			require.NoError(s.T(), err)
			assert.Len(s.T(), after, len(before)+1)

			require.NotZero(s.T(), out.Doc.ID)
			doc := s.getDoc(ctx, out.Doc.ID)
			assert.Equal(s.T(), tt.title, doc.Title)
			assert.Equal(s.T(), tt.description, doc.Description)
			assert.Empty(s.T(), s.listDocChunks(ctx, doc.ID))
		})
	}
}

// --- list_docs ----------------------------------------------------

func (s *IntegrationTestSuite) TestListDocs() {
	ctx := s.Context()

	first := s.createDoc(ctx, "First doc")
	second := s.createDoc(ctx, "Second doc")
	s.createDoc(s.otherUserContext(ctx), "Other user's doc")

	_, out, err := docs.ListDocs(ctx, nil, docs.ListDocsInput{})
	require.NoError(s.T(), err)

	require.Len(s.T(), out.Docs, 2)
	assert.Equal(s.T(), first, out.Docs[0].ID)
	assert.Equal(s.T(), "First doc", out.Docs[0].Title)
	assert.Equal(s.T(), "First doc description", out.Docs[0].Description)
	assert.Equal(s.T(), second, out.Docs[1].ID)
	assert.Equal(s.T(), "Second doc", out.Docs[1].Title)
	for _, doc := range out.Docs {
		assert.NotEmpty(s.T(), doc.UpdatedAt)
		assert.Equal(s.T(), doc.CreatedAt, doc.UpdatedAt)
	}
	stored := s.getDoc(ctx, first)
	assert.Equal(s.T(), stored.CreatedAt, stored.UpdatedAt)
}

// --- edit_doc -----------------------------------------------------

func (s *IntegrationTestSuite) TestEditDoc() {
	ctx := s.Context()
	otherCtx := s.otherUserContext(ctx)
	foreign := s.createDoc(otherCtx, "Foreign")

	tests := []struct {
		name            string
		title           *string
		description     *string
		docID           *int64 // nil = a fresh document of the user
		wantTitle       string
		wantDescription string
		wantErr         string
	}{
		{name: "title alone", title: util.Ptr("New title"), wantTitle: "New title", wantDescription: "Doc description"},
		{name: "description alone", description: util.Ptr("New description"), wantTitle: "Doc", wantDescription: "New description"},
		{name: "both", title: util.Ptr("New title"), description: util.Ptr("New description"), wantTitle: "New title", wantDescription: "New description"},
		{name: "no field fails", wantErr: "at least one of title or description is required"},
		{name: "values equal to the current ones fail", title: util.Ptr("Doc"), description: util.Ptr("Doc description"), wantErr: "nothing to change"},
		{name: "empty title fails", title: util.Ptr(" "), wantErr: "title is required"},
		{name: "empty description fails", description: util.Ptr(""), wantErr: "description is required"},
		{name: "multi-line description fails", description: util.Ptr("one\ntwo"), wantErr: "description must be one line"},
		{name: "too long description fails", description: util.Ptr(strings.Repeat("a", docMaxLineLength+1)), wantErr: "maximum is 200"},
		{name: "unknown document fails", title: util.Ptr("x"), docID: util.Ptr(int64(-1)), wantErr: "document not found"},
		{name: "other user's document fails", title: util.Ptr("x"), docID: &foreign, wantErr: "document not found"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			own := s.createDoc(ctx, "Doc")
			before := s.getDoc(ctx, own)
			docID := own
			if tt.docID != nil {
				docID = *tt.docID
			}

			_, _, err := docs.EditDoc(ctx, nil, docs.EditDocInput{DocID: docID, Title: tt.title, Description: tt.description})

			after := s.getDoc(ctx, own)
			if tt.wantErr != "" {
				require.Error(s.T(), err)
				assert.Contains(s.T(), err.Error(), tt.wantErr)
				assert.Equal(s.T(), before, after)
				assert.Equal(s.T(), "Foreign", s.getDoc(otherCtx, foreign).Title)
				return
			}
			require.NoError(s.T(), err)
			assert.Equal(s.T(), tt.wantTitle, after.Title)
			assert.Equal(s.T(), tt.wantDescription, after.Description)
			assert.True(s.T(), after.UpdatedAt.After(before.UpdatedAt))
			assert.Equal(s.T(), before.CreatedAt, after.CreatedAt)
		})
	}
}

// --- docs.updated_at -------------------------------------------------------

func (s *IntegrationTestSuite) TestDocUpdatedAt() {
	ctx := s.Context()

	docID := s.createDoc(ctx, "Doc")
	untouchedID := s.createDoc(ctx, "Untouched")
	untouched := s.getDoc(ctx, untouchedID)
	var chunkID int64

	steps := []struct {
		name    string
		call    func() error
		wantErr bool
	}{
		{name: "create_doc_chunk", call: func() error {
			_, out, err := docs.CreateDocChunk(ctx, nil, docs.CreateDocChunkInput{DocID: docID, Body: "first"})
			chunkID = out.Chunk.ID
			return err
		}},
		{name: "second chunk", call: func() error {
			_, _, err := docs.CreateDocChunk(ctx, nil, docs.CreateDocChunkInput{DocID: docID, Body: "second"})
			return err
		}},
		{name: "edit_doc_chunk with body", call: func() error {
			_, _, err := docs.EditDocChunk(ctx, nil, docs.EditDocChunkInput{ChunkID: chunkID, Body: util.Ptr("edited")})
			return err
		}},
		{name: "edit_doc_chunk with append_body", call: func() error {
			_, _, err := docs.EditDocChunk(ctx, nil, docs.EditDocChunkInput{ChunkID: chunkID, AppendBody: util.Ptr("appended")})
			return err
		}},
		{name: "edit_doc_chunk with position", call: func() error {
			_, _, err := docs.EditDocChunk(ctx, nil, docs.EditDocChunkInput{ChunkID: chunkID, Position: util.Ptr(2)})
			return err
		}},
		{name: "revert_doc_chunk", call: func() error {
			_, _, err := docs.RevertDocChunk(ctx, nil, docs.RevertDocChunkInput{ChunkID: chunkID})
			return err
		}},
		{name: "failed create leaves it unchanged", wantErr: true, call: func() error {
			_, _, err := docs.CreateDocChunk(ctx, nil, docs.CreateDocChunkInput{DocID: docID, Body: "x", Position: util.Ptr(9)})
			return err
		}},
		{name: "failed edit leaves it unchanged", wantErr: true, call: func() error {
			_, _, err := docs.EditDocChunk(ctx, nil, docs.EditDocChunkInput{ChunkID: chunkID, Position: util.Ptr(9)})
			return err
		}},
		{name: "delete_doc_chunk", call: func() error {
			_, _, err := docs.DeleteDocChunk(ctx, nil, docs.DeleteDocChunkInput{ChunkID: chunkID})
			return err
		}},
	}

	for _, step := range steps {
		s.Run(step.name, func() {
			before := s.getDoc(ctx, docID)

			err := step.call()

			after := s.getDoc(ctx, docID)
			if step.wantErr {
				require.Error(s.T(), err)
				assert.Equal(s.T(), before.UpdatedAt, after.UpdatedAt)
			} else {
				require.NoError(s.T(), err)
				assert.True(s.T(), after.UpdatedAt.After(before.UpdatedAt))
			}
			assert.Equal(s.T(), untouched.UpdatedAt, s.getDoc(ctx, untouchedID).UpdatedAt)
		})
	}
}

// --- create_doc_chunk ---------------------------------------------

func (s *IntegrationTestSuite) TestCreateDocChunk() {
	ctx := s.Context()
	foreign := s.createDoc(s.otherUserContext(ctx), "Foreign")

	maxBody := docLines(docMaxLines, "line")
	maxLine := strings.Repeat("a", docMaxLineLength)
	cyrillicLine := strings.Repeat("я", docMaxLineLength)

	tests := []struct {
		name     string
		existing []string
		body     string
		position *int
		docID    *int64 // nil = a fresh document of the user
		want     []string
		wantErr  string
	}{
		{name: "created at the end", existing: []string{"a", "b"}, body: "new", want: []string{"a", "b", "new"}},
		{name: "first chunk of an empty document", body: "new", want: []string{"new"}},
		{name: "position 1 puts it first", existing: []string{"a", "b"}, body: "new", position: util.Ptr(1), want: []string{"new", "a", "b"}},
		{name: "taken position in the middle inserts there", existing: []string{"a", "b", "c"}, body: "new", position: util.Ptr(2), want: []string{"a", "new", "b", "c"}},
		{name: "position after the last puts it last", existing: []string{"a", "b"}, body: "new", position: util.Ptr(3), want: []string{"a", "b", "new"}},
		{name: "position 0 fails", existing: []string{"a", "b"}, body: "new", position: util.Ptr(0), wantErr: "position must be between 1 and 3"},
		{name: "negative position fails", existing: []string{"a", "b"}, body: "new", position: util.Ptr(-1), wantErr: "position must be between 1 and 3"},
		{name: "position too far fails", existing: []string{"a", "b"}, body: "new", position: util.Ptr(4), wantErr: "position must be between 1 and 3"},
		{name: "max number of lines", body: maxBody, want: []string{maxBody}},
		{name: "line of the max length", body: maxLine, want: []string{maxLine}},
		{name: "length is counted in characters", body: cyrillicLine, want: []string{cyrillicLine}},
		{name: "one line too many fails", body: maxBody + "\nline", wantErr: "body has 61 lines"},
		{name: "one character too long fails", body: "ok\n" + maxLine + "a", wantErr: "line 2 of body has 201 characters"},
		{name: "empty body fails", existing: []string{"a"}, body: " \n ", wantErr: "body is required"},
		{name: "unknown document fails", body: "new", docID: util.Ptr(int64(-1)), wantErr: "document not found"},
		{name: "other user's document fails", body: "new", docID: &foreign, wantErr: "document not found"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			own := s.createDoc(ctx, "Doc")
			s.createDocChunks(ctx, own, tt.existing...)
			docID := own
			if tt.docID != nil {
				docID = *tt.docID
			}

			_, out, err := docs.CreateDocChunk(ctx, nil, docs.CreateDocChunkInput{DocID: docID, Body: tt.body, Position: tt.position})

			if tt.wantErr != "" {
				require.Error(s.T(), err)
				assert.Contains(s.T(), err.Error(), tt.wantErr)
				s.assertDocChunks(ctx, own, tt.existing...)
				return
			}
			require.NoError(s.T(), err)
			s.assertDocChunks(ctx, own, tt.want...)

			chunk := s.getDocChunk(ctx, out.Chunk.ID)
			assert.Equal(s.T(), tt.body, chunk.Body)
			assert.Nil(s.T(), chunk.PreviousBody)
			assert.Equal(s.T(), chunk.Position, out.Chunk.Position)
		})
	}
}

// --- get_doc ------------------------------------------------------

func (s *IntegrationTestSuite) TestGetDoc() {
	ctx := s.Context()

	docID := s.createDoc(ctx, "Plan")
	chunkIDs := s.createDocChunks(ctx, docID, "# Intro", "second", "third")
	longID := s.createDoc(ctx, "Long")
	s.createDocChunks(ctx, longID, "c1", "c2", "c3", "c4", "c5", "c6", "c7")
	emptyID := s.createDoc(ctx, "Empty")
	foreign := s.createDoc(s.otherUserContext(ctx), "Foreign")

	tests := []struct {
		name          string
		docID         int64
		offset        *int
		limit         *int
		wantTitle     string
		wantBodies    []string
		wantPositions []int
		wantTotal     int
		wantErr       string
	}{
		{name: "document with chunks", docID: docID, wantTitle: "Plan", wantBodies: []string{"# Intro", "second", "third"}, wantPositions: []int{1, 2, 3}, wantTotal: 3},
		{name: "default page is the first 5 chunks", docID: longID, wantTitle: "Long", wantBodies: []string{"c1", "c2", "c3", "c4", "c5"}, wantPositions: []int{1, 2, 3, 4, 5}, wantTotal: 7},
		{name: "offset returns the rest", docID: longID, offset: util.Ptr(5), wantTitle: "Long", wantBodies: []string{"c6", "c7"}, wantPositions: []int{6, 7}, wantTotal: 7},
		{name: "offset and limit cut a page from the middle", docID: longID, offset: util.Ptr(2), limit: util.Ptr(3), wantTitle: "Long", wantBodies: []string{"c3", "c4", "c5"}, wantPositions: []int{3, 4, 5}, wantTotal: 7},
		{name: "limit over the number of chunks returns all", docID: longID, limit: util.Ptr(100), wantTitle: "Long", wantBodies: []string{"c1", "c2", "c3", "c4", "c5", "c6", "c7"}, wantPositions: []int{1, 2, 3, 4, 5, 6, 7}, wantTotal: 7},
		{name: "offset equal to the number of chunks returns none", docID: longID, offset: util.Ptr(7), wantTitle: "Long", wantBodies: []string{}, wantPositions: []int{}, wantTotal: 7},
		{name: "offset past the end returns none", docID: longID, offset: util.Ptr(50), wantTitle: "Long", wantBodies: []string{}, wantPositions: []int{}, wantTotal: 7},
		{name: "negative offset fails", docID: longID, offset: util.Ptr(-1), wantErr: "offset must be 0 or greater"},
		{name: "limit 0 fails", docID: longID, limit: util.Ptr(0), wantErr: "limit must be 1 or greater"},
		{name: "negative limit fails", docID: longID, limit: util.Ptr(-3), wantErr: "limit must be 1 or greater"},
		{name: "empty document", docID: emptyID, wantTitle: "Empty", wantBodies: []string{}, wantPositions: []int{}, wantTotal: 0},
		{name: "unknown document fails", docID: -1, wantErr: "document not found"},
		{name: "other user's document fails", docID: foreign, wantErr: "document not found"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, out, err := docs.GetDoc(ctx, nil, docs.GetDocInput{DocID: tt.docID, Offset: tt.offset, Limit: tt.limit})
			if tt.wantErr != "" {
				require.Error(s.T(), err)
				assert.Contains(s.T(), err.Error(), tt.wantErr)
				return
			}
			require.NoError(s.T(), err)

			assert.Equal(s.T(), tt.docID, out.Doc.ID)
			assert.Equal(s.T(), tt.wantTitle, out.Doc.Title)
			assert.Equal(s.T(), tt.wantTitle+" description", out.Doc.Description)
			assert.Equal(s.T(), tt.wantTotal, out.TotalChunks)
			bodies := make([]string, len(out.Chunks))
			positions := make([]int, len(out.Chunks))
			for i, chunk := range out.Chunks {
				bodies[i] = chunk.Body
				positions[i] = chunk.Position
				assert.NotEmpty(s.T(), chunk.UpdatedAt)
			}
			assert.Equal(s.T(), tt.wantBodies, bodies)
			assert.Equal(s.T(), tt.wantPositions, positions)
			if tt.docID == docID {
				assert.Equal(s.T(), chunkIDs[0], out.Chunks[0].ID)
				assert.Equal(s.T(), chunkIDs[2], out.Chunks[2].ID)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestGetDoc_TableOfContents() {
	ctx := s.Context()

	item := func(heading string, level, position int) docs.ContentsItemResult {
		return docs.ContentsItemResult{Heading: heading, Level: level, Position: position}
	}

	tests := []struct {
		name   string
		bodies []string
		limit  *int
		want   []docs.ContentsItemResult
	}{
		{
			name:   "headings of all chunks in reading order",
			bodies: []string{"# Plan\n\ntext", "## Goals\n\ntext", "### Details"},
			want:   []docs.ContentsItemResult{item("Plan", 1, 1), item("Goals", 2, 2), item("Details", 3, 3)},
		},
		{
			name:   "headings of chunks outside the returned page are listed",
			bodies: []string{"# One", "# Two", "# Three"},
			limit:  util.Ptr(1),
			want:   []docs.ContentsItemResult{item("One", 1, 1), item("Two", 1, 2), item("Three", 1, 3)},
		},
		{
			name:   "two headings in one chunk share its position",
			bodies: []string{"# First\n\ntext\n\n## Second\n\nmore"},
			want:   []docs.ContentsItemResult{item("First", 1, 1), item("Second", 2, 1)},
		},
		{
			name:   "chunk with no headings adds no items",
			bodies: []string{"# Top", "just text", "## Bottom"},
			want:   []docs.ContentsItemResult{item("Top", 1, 1), item("Bottom", 2, 3)},
		},
		{
			name:   "same heading text in two chunks gives two items",
			bodies: []string{"## Notes", "## Notes"},
			want:   []docs.ContentsItemResult{item("Notes", 2, 1), item("Notes", 2, 2)},
		},
		{
			name:   "markup inside a heading is returned as plain text",
			bodies: []string{"# A **bold** and `code` title"},
			want:   []docs.ContentsItemResult{item("A bold and code title", 1, 1)},
		},
		{
			name:   "line with # inside a code block is not a heading",
			bodies: []string{"# Real\n\n```\n# not a heading\n```"},
			want:   []docs.ContentsItemResult{item("Real", 1, 1)},
		},
		{
			name:   "document with no headings",
			bodies: []string{"text", "more text"},
			want:   []docs.ContentsItemResult{},
		},
		{
			name:   "empty document",
			bodies: nil,
			want:   []docs.ContentsItemResult{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			docID := s.createDoc(ctx, "Contents")
			s.createDocChunks(ctx, docID, tt.bodies...)

			_, out, err := docs.GetDoc(ctx, nil, docs.GetDocInput{DocID: docID, Limit: tt.limit})
			require.NoError(s.T(), err)

			assert.Equal(s.T(), tt.want, out.TableOfContents)
			if tt.limit != nil {
				assert.Len(s.T(), out.Chunks, *tt.limit)
			}
		})
	}
}

// --- edit_doc_chunk -----------------------------------------------

func (s *IntegrationTestSuite) TestEditDocChunk() {
	ctx := s.Context()
	otherCtx := s.otherUserContext(ctx)
	foreignDoc := s.createDoc(otherCtx, "Foreign")
	foreignChunk := s.createDocChunks(otherCtx, foreignDoc, "foreign")[0]

	maxBody := docLines(docMaxLines, "line")
	longLine := strings.Repeat("a", docMaxLineLength+1)

	// Every case runs on a fresh document with the chunks a, b, c, d and
	// edits chunk b unless chunkID is set.
	tests := []struct {
		name         string
		body         *string
		appendBody   *string
		position     *int
		chunkID      *int64
		startBody    string // body of chunk b, "b" when empty
		want         []string
		wantPrevious *string
		bodyEdited   bool
		wantErr      string
	}{
		{name: "body overwrites and keeps the old one", body: util.Ptr("new"), want: []string{"a", "new", "c", "d"}, wantPrevious: util.Ptr("b"), bodyEdited: true},
		{name: "append_body adds after a blank line", appendBody: util.Ptr("more"), want: []string{"a", "b\n\nmore", "c", "d"}, wantPrevious: util.Ptr("b"), bodyEdited: true},
		{name: "position moves up", position: util.Ptr(1), want: []string{"b", "a", "c", "d"}},
		{name: "position moves down", position: util.Ptr(4), want: []string{"a", "c", "d", "b"}},
		{name: "position moves down by one", position: util.Ptr(3), want: []string{"a", "c", "b", "d"}},
		{name: "body and position together", body: util.Ptr("new"), position: util.Ptr(3), want: []string{"a", "c", "new", "d"}, wantPrevious: util.Ptr("b"), bodyEdited: true},
		{name: "same body with a new position only moves", body: util.Ptr("b"), position: util.Ptr(1), want: []string{"b", "a", "c", "d"}},
		{name: "position 0 fails", position: util.Ptr(0), wantErr: "position must be between 1 and 4"},
		{name: "position after the last fails", position: util.Ptr(5), wantErr: "position must be between 1 and 4"},
		{name: "bad position fails even with a valid body", body: util.Ptr("new"), position: util.Ptr(5), wantErr: "position must be between 1 and 4"},
		{name: "body over the line limit fails", body: util.Ptr(maxBody + "\nline"), wantErr: "body has 61 lines"},
		{name: "append over the line limit fails", startBody: maxBody, appendBody: util.Ptr("more"), wantErr: "body has 62 lines"},
		{name: "too long line fails", body: util.Ptr(longLine), wantErr: "has 201 characters"},
		{name: "body and append_body fail", body: util.Ptr("x"), appendBody: util.Ptr("y"), wantErr: "mutually exclusive"},
		{name: "no field fails", wantErr: "at least one of body, append_body or position is required"},
		{name: "same body fails", body: util.Ptr("b"), wantErr: "nothing to change"},
		{name: "same position fails", position: util.Ptr(2), wantErr: "nothing to change"},
		{name: "empty body fails", body: util.Ptr(" "), wantErr: "body cannot be empty"},
		{name: "empty append_body fails", appendBody: util.Ptr(""), wantErr: "append_body cannot be empty"},
		{name: "unknown chunk fails", body: util.Ptr("x"), chunkID: util.Ptr(int64(-1)), wantErr: "chunk not found"},
		{name: "other user's chunk fails", body: util.Ptr("x"), chunkID: &foreignChunk, wantErr: "chunk not found"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			startBody := "b"
			if tt.startBody != "" {
				startBody = tt.startBody
			}
			docID := s.createDoc(ctx, "Doc")
			ids := s.createDocChunks(ctx, docID, "a", startBody, "c", "d")
			before := s.getDocChunk(ctx, ids[1])
			chunkID := ids[1]
			if tt.chunkID != nil {
				chunkID = *tt.chunkID
			}

			_, _, err := docs.EditDocChunk(ctx, nil, docs.EditDocChunkInput{
				ChunkID: chunkID, Body: tt.body, AppendBody: tt.appendBody, Position: tt.position,
			})

			after := s.getDocChunk(ctx, ids[1])
			if tt.wantErr != "" {
				require.Error(s.T(), err)
				assert.Contains(s.T(), err.Error(), tt.wantErr)
				assert.Equal(s.T(), before, after)
				s.assertDocChunks(ctx, docID, "a", startBody, "c", "d")
				assert.Equal(s.T(), "foreign", s.getDocChunk(otherCtx, foreignChunk).Body)
				return
			}
			require.NoError(s.T(), err)
			s.assertDocChunks(ctx, docID, tt.want...)
			assert.Equal(s.T(), tt.wantPrevious, after.PreviousBody)
			if tt.bodyEdited {
				assert.True(s.T(), after.UpdatedAt.After(before.UpdatedAt))
			} else {
				assert.Equal(s.T(), before.UpdatedAt, after.UpdatedAt)
			}
		})
	}

	s.Run("second edit keeps only the body it replaced", func() {
		docID := s.createDoc(ctx, "Doc")
		chunkID := s.createDocChunks(ctx, docID, "v1")[0]

		for _, body := range []string{"v2", "v3"} {
			_, _, err := docs.EditDocChunk(ctx, nil, docs.EditDocChunkInput{ChunkID: chunkID, Body: &body})
			require.NoError(s.T(), err)
		}

		chunk := s.getDocChunk(ctx, chunkID)
		assert.Equal(s.T(), "v3", chunk.Body)
		assert.Equal(s.T(), util.Ptr("v2"), chunk.PreviousBody)
	})

	s.Run("move keeps the previous body", func() {
		docID := s.createDoc(ctx, "Doc")
		ids := s.createDocChunks(ctx, docID, "v1", "other")
		_, _, err := docs.EditDocChunk(ctx, nil, docs.EditDocChunkInput{ChunkID: ids[0], Body: util.Ptr("v2")})
		require.NoError(s.T(), err)

		_, out, err := docs.EditDocChunk(ctx, nil, docs.EditDocChunkInput{ChunkID: ids[0], Position: util.Ptr(2)})
		require.NoError(s.T(), err)

		assert.Equal(s.T(), 2, out.Chunk.Position)
		chunk := s.getDocChunk(ctx, ids[0])
		assert.Equal(s.T(), "v2", chunk.Body)
		assert.Equal(s.T(), util.Ptr("v1"), chunk.PreviousBody)
		s.assertDocChunks(ctx, docID, "other", "v2")
	})
}

// --- revert_doc_chunk ---------------------------------------------

func (s *IntegrationTestSuite) TestRevertDocChunk() {
	ctx := s.Context()
	otherCtx := s.otherUserContext(ctx)

	// editedChunk seeds a chunk whose body "new" replaced "old".
	editedChunk := func(ctx context.Context) int64 {
		docID := s.createDoc(ctx, "Doc")
		chunkID := s.createDocChunks(ctx, docID, "old")[0]
		chunk := s.getDocChunk(ctx, chunkID)
		chunk.Body, chunk.PreviousBody = "new", util.Ptr("old")
		require.NoError(s.T(), s.Repo().UpdateDocChunk(ctx, chunk))
		return chunkID
	}

	s.Run("swaps the bodies, a second revert restores the edit", func() {
		chunkID := editedChunk(ctx)
		before := s.getDocChunk(ctx, chunkID)

		_, out, err := docs.RevertDocChunk(ctx, nil, docs.RevertDocChunkInput{ChunkID: chunkID})
		require.NoError(s.T(), err)

		assert.Equal(s.T(), "old", out.Chunk.Body)
		reverted := s.getDocChunk(ctx, chunkID)
		assert.Equal(s.T(), "old", reverted.Body)
		assert.Equal(s.T(), util.Ptr("new"), reverted.PreviousBody)
		assert.True(s.T(), reverted.UpdatedAt.After(before.UpdatedAt))

		_, _, err = docs.RevertDocChunk(ctx, nil, docs.RevertDocChunkInput{ChunkID: chunkID})
		require.NoError(s.T(), err)

		restored := s.getDocChunk(ctx, chunkID)
		assert.Equal(s.T(), "new", restored.Body)
		assert.Equal(s.T(), util.Ptr("old"), restored.PreviousBody)
	})

	neverEdited := s.createDocChunks(ctx, s.createDoc(ctx, "Doc"), "only")[0]
	foreign := editedChunk(otherCtx)

	tests := []struct {
		name    string
		chunkID int64
		wantErr string
	}{
		{name: "chunk with no previous body fails", chunkID: neverEdited, wantErr: "no previous body"},
		{name: "unknown chunk fails", chunkID: -1, wantErr: "chunk not found"},
		{name: "other user's chunk fails", chunkID: foreign, wantErr: "chunk not found"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, _, err := docs.RevertDocChunk(ctx, nil, docs.RevertDocChunkInput{ChunkID: tt.chunkID})

			require.Error(s.T(), err)
			assert.Contains(s.T(), err.Error(), tt.wantErr)
			assert.Equal(s.T(), "only", s.getDocChunk(ctx, neverEdited).Body)
			assert.Equal(s.T(), "new", s.getDocChunk(otherCtx, foreign).Body)
		})
	}
}

// --- delete_doc_chunk ---------------------------------------------

func (s *IntegrationTestSuite) TestDeleteDocChunk() {
	ctx := s.Context()
	otherCtx := s.otherUserContext(ctx)
	foreignDoc := s.createDoc(otherCtx, "Foreign")
	foreignChunk := s.createDocChunks(otherCtx, foreignDoc, "foreign")[0]

	tests := []struct {
		name    string
		index   int    // which of the chunks a, b, c to delete
		chunkID *int64 // overrides index
		want    []string
		wantErr string
	}{
		{name: "first chunk", index: 0, want: []string{"b", "c"}},
		{name: "middle chunk", index: 1, want: []string{"a", "c"}},
		{name: "last chunk", index: 2, want: []string{"a", "b"}},
		{name: "unknown chunk fails", chunkID: util.Ptr(int64(-1)), wantErr: "chunk not found"},
		{name: "other user's chunk fails", chunkID: &foreignChunk, wantErr: "chunk not found"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			docID := s.createDoc(ctx, "Doc")
			ids := s.createDocChunks(ctx, docID, "a", "b", "c")
			chunkID := ids[tt.index]
			if tt.chunkID != nil {
				chunkID = *tt.chunkID
			}

			_, out, err := docs.DeleteDocChunk(ctx, nil, docs.DeleteDocChunkInput{ChunkID: chunkID})

			if tt.wantErr != "" {
				require.Error(s.T(), err)
				assert.Contains(s.T(), err.Error(), tt.wantErr)
				s.assertDocChunks(ctx, docID, "a", "b", "c")
				s.assertDocChunks(otherCtx, foreignDoc, "foreign")
				return
			}
			require.NoError(s.T(), err)
			assert.True(s.T(), out.Success)
			s.assertDocChunks(ctx, docID, tt.want...)

			gone, err := s.Repo().GetDocChunk(ctx, chunkID, gateways.UserIDFromContext(ctx))
			require.NoError(s.T(), err)
			assert.Nil(s.T(), gone)
		})
	}
}
