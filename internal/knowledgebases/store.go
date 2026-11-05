package knowledgebases

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Base is one registered knowledge base ("CLI-based base attachment":
// "conjecture base add <path> --authority <...> --scope <...>").
type Base struct {
	ID        int64
	Path      string
	Authority Authority
	// Scope is the base's declared scope exactly as the specification's
	// own examples write it: "project" for the whole project, or
	// "area:<name>" for one area (see [InScope]).
	Scope string

	// ContentHash and LastReviewDate are required for a norm-authority
	// base ("Required metadata for norm-authority bases"). Both are
	// empty for a base at any other authority level, where the
	// specification requires neither.
	ContentHash    []byte
	LastReviewDate string

	// Incomplete reports whether a norm-authority base is missing its
	// ContentHash or LastReviewDate - "The system SHALL reject or flag
	// a norm-authority base missing either." This package flags rather
	// than rejects: an incomplete registration is still recorded (so
	// its material is visible and the gap itself is visible), not
	// silently dropped.
	Incomplete bool

	CreatedAt time.Time
}

// EnsureSchema creates the bases table if it does not already exist.
// It is safe to call every time a store is opened.
//
// A registration is stored here, in the same local, per-repository
// SQLite database as [persistence.OpenStore] opens for every other
// piece of non-git-tracked state, rather than as a git-tracked file.
// The specification never says which directly, but persistence's own
// "Git-tracked specification files" requirement names exactly what is
// git-tracked (.cjtr phenomenon files, materialization output,
// settings.cjtr) without mentioning a base registration, and its
// "Everything else stays local-only" requirement is written to cover
// precisely what that list leaves out.
func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS knowledge_bases (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			path             TEXT NOT NULL,
			authority        TEXT NOT NULL,
			scope            TEXT NOT NULL,
			content_hash     BLOB,
			last_review_date TEXT NOT NULL,
			incomplete       INTEGER NOT NULL,
			created_at       TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("knowledgebases: creating schema: %w", err)
	}
	return nil
}

// Add registers b, setting b.Incomplete when it declares Norm
// authority but is missing ContentHash or LastReviewDate (rather than
// rejecting it - see [Base.Incomplete]), and returns the stored record
// with its ID and CreatedAt (now, in UTC) set.
func Add(db *sql.DB, b Base) (Base, error) {
	if b.Authority == Norm && (len(b.ContentHash) == 0 || b.LastReviewDate == "") {
		b.Incomplete = true
	}

	b.CreatedAt = time.Now().UTC()

	res, err := db.Exec(
		`INSERT INTO knowledge_bases
			(path, authority, scope, content_hash, last_review_date, incomplete, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.Path, b.Authority.String(), b.Scope, b.ContentHash, b.LastReviewDate, b.Incomplete, b.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Base{}, fmt.Errorf("knowledgebases: registering base: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Base{}, fmt.Errorf("knowledgebases: registering base: %w", err)
	}
	b.ID = id
	return b, nil
}

// List returns every registered base, in registration order.
func List(db *sql.DB) ([]Base, error) {
	rows, err := db.Query(
		`SELECT id, path, authority, scope, content_hash, last_review_date, incomplete, created_at
		 FROM knowledge_bases ORDER BY id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("knowledgebases: listing bases: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var bases []Base
	for rows.Next() {
		var (
			b         Base
			authority string
			createdAt string
		)
		if err := rows.Scan(&b.ID, &b.Path, &authority, &b.Scope, &b.ContentHash, &b.LastReviewDate, &b.Incomplete, &createdAt); err != nil {
			return nil, fmt.Errorf("knowledgebases: listing bases: %w", err)
		}
		parsedAuthority, err := ParseAuthority(authority)
		if err != nil {
			return nil, fmt.Errorf("knowledgebases: %w", err)
		}
		b.Authority = parsedAuthority
		createdAtTime, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("knowledgebases: parsing created_at: %w", err)
		}
		b.CreatedAt = createdAtTime
		bases = append(bases, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("knowledgebases: listing bases: %w", err)
	}
	return bases, nil
}

// InScope reports whether b's declared scope applies to area
// ("Scoped application"): a "project" scope always applies; an
// "area:<name>" scope applies only when name equals area exactly - the
// specification's own two worked scope forms, "project" and
// "area:shipping", and nothing beyond them.
func InScope(b Base, area string) bool {
	if b.Scope == "project" {
		return true
	}
	if name, ok := strings.CutPrefix(b.Scope, "area:"); ok {
		return name == area
	}
	return false
}
