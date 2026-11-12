package knowledgebases

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// EnsureSearchSchema creates the local full-text search index over
// indexed base content, if it does not already exist - "Local
// indexing for retrieval"'s textual-search half ("The system SHALL
// index attached bases locally using both textual search and
// quantized embeddings, without sending base content to a remote
// service"). The quantized-embeddings half is [EnsureEmbeddingSchema]/
// [Embed], its own separate schema and function - see that doc
// comment for what this project's own "quantized embeddings" means.
//
// SQLite's own FTS5 virtual table extension does the indexing and
// ranking itself, already available through modernc.org/sqlite - the
// driver every other store in this codebase already uses - so this
// needs no new dependency. Content is written only into the local
// SQLite file [persistence.OpenStore] already opens, never
// transmitted anywhere, satisfying "Base content stays local" by
// construction rather than by a separate check.
func EnsureSearchSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_base_search USING fts5(
			base_id UNINDEXED, file_path UNINDEXED, content
		);
	`)
	if err != nil {
		return fmt.Errorf("knowledgebases: creating search schema: %w", err)
	}
	return nil
}

// IndexBase reads every regular file at b.Path - b.Path itself if it
// names a single file, or every file under it if a directory, the
// same "a file or a directory" shape [ComputeContentHash] already
// handles - and indexes each one's text content for both local
// full-text search and quantized-embedding search ([storeEmbedding]),
// replacing any previously indexed content and embeddings for this
// base first (so re-running "conjecture base add" on an already-
// registered path never leaves stale entries behind it).
func IndexBase(db *sql.DB, b Base) error {
	if _, err := db.Exec(`DELETE FROM knowledge_base_search WHERE base_id = ?`, b.ID); err != nil {
		return fmt.Errorf("knowledgebases: clearing previous index for base %d: %w", b.ID, err)
	}
	if _, err := db.Exec(`DELETE FROM knowledge_base_embedding WHERE base_id = ?`, b.ID); err != nil {
		return fmt.Errorf("knowledgebases: clearing previous embeddings for base %d: %w", b.ID, err)
	}

	info, err := os.Stat(b.Path)
	if err != nil {
		return fmt.Errorf("knowledgebases: %w", err)
	}

	index := func(path string) error {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := db.Exec(
			`INSERT INTO knowledge_base_search (base_id, file_path, content) VALUES (?, ?, ?)`,
			b.ID, path, string(content),
		); err != nil {
			return err
		}
		return storeEmbedding(db, b.ID, path, content)
	}

	if !info.IsDir() {
		if err := index(b.Path); err != nil {
			return fmt.Errorf("knowledgebases: indexing %q: %w", b.Path, err)
		}
		return nil
	}

	err = filepath.WalkDir(b.Path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		return index(p)
	})
	if err != nil {
		return fmt.Errorf("knowledgebases: indexing %q: %w", b.Path, err)
	}
	return nil
}

// SearchResult is one full-text match against the local index.
type SearchResult struct {
	BaseID   int64
	FilePath string
	Snippet  string
}

// Search returns every indexed file matching query, most relevant
// first (FTS5's own bm25 ranking), with a short snippet of the
// matching context (matched terms wrapped in "[...]").
//
// query is always sent to FTS5 as one quoted literal phrase, with any
// embedded double quote doubled per FTS5's own string-literal escaping
// rule - this package exposes plain text search, not FTS5's full
// query-operator syntax (AND/OR/NOT, column filters, prefix "*"),
// which would otherwise make an ordinary word containing "-" or ":"
// behave unpredictably for a caller who never asked for operators.
func Search(db *sql.DB, query string) ([]SearchResult, error) {
	ftsQuery := `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
	rows, err := db.Query(
		`SELECT base_id, file_path, snippet(knowledge_base_search, 2, '[', ']', '...', 12)
		 FROM knowledge_base_search WHERE knowledge_base_search MATCH ? ORDER BY rank`,
		ftsQuery,
	)
	if err != nil {
		return nil, fmt.Errorf("knowledgebases: searching: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.BaseID, &r.FilePath, &r.Snippet); err != nil {
			return nil, fmt.Errorf("knowledgebases: searching: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("knowledgebases: searching: %w", err)
	}
	return results, nil
}

// SearchInScope is [Search], filtered to only bases [InScope] for
// area - "Scoped application": "the system does not apply that base's
// content when judging or verifying a different area." Search itself
// stays scope-agnostic (a human asking what any attached base says
// about a query is a real, different use this package still serves),
// so this is a second entry point, not a behavior change to Search.
func SearchInScope(db *sql.DB, query, area string) ([]SearchResult, error) {
	results, err := Search(db, query)
	if err != nil {
		return nil, err
	}

	bases, err := List(db)
	if err != nil {
		return nil, err
	}
	inScope := make(map[int64]bool, len(bases))
	for _, b := range bases {
		inScope[b.ID] = InScope(b, area)
	}

	var filtered []SearchResult
	for _, r := range results {
		if inScope[r.BaseID] {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}
