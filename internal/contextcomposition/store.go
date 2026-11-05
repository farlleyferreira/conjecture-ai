package contextcomposition

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Version is one snapshot of a context's content, numbered
// sequentially per context starting at 1 - the "v4", "v7" tags the
// specification's own worked examples use (`conjecture context open
// checkout@v4`, `conjecture context diff checkout@v4 checkout@v7`).
type Version struct {
	ContextName string
	Seq         int
	Content     string
	CreatedAt   time.Time
}

// Tag renders v's sequence number in the specification's own "v<N>"
// form.
func (v Version) Tag() string {
	return fmt.Sprintf("v%d", v.Seq)
}

// VersionRef is a parsed "<name>" or "<name>@v<N>" reference, the
// argument shape every worked example in the product proposal uses for
// `conjecture context open`/`diff` (e.g. "checkout@v4"). Seq is 0 when
// no version was given - "latest", the form `resume` and `restore`'s
// own worked examples use ("checkout", "old-checkout", with no "@").
type VersionRef struct {
	Name string
	Seq  int
}

// ParseVersionRef parses s into a VersionRef.
func ParseVersionRef(s string) (VersionRef, error) {
	name, tag, hasVersion := strings.Cut(s, "@")
	if name == "" {
		return VersionRef{}, fmt.Errorf("contextcomposition: %q has no context name before \"@\"", s)
	}
	if !hasVersion {
		return VersionRef{Name: name}, nil
	}

	seq, err := parseVersionTag(tag)
	if err != nil {
		return VersionRef{}, err
	}
	return VersionRef{Name: name, Seq: seq}, nil
}

func parseVersionTag(tag string) (int, error) {
	digits, ok := strings.CutPrefix(tag, "v")
	if ok {
		if seq, err := strconv.Atoi(digits); err == nil && seq >= 1 {
			return seq, nil
		}
	}
	return 0, fmt.Errorf("contextcomposition: %q is not a recognized version tag (want \"v<N>\")", tag)
}

// EnsureSchema creates the contexts and context_versions tables if
// they do not already exist. It is safe to call every time a store is
// opened.
//
// This is stored here, in the same local, per-repository SQLite
// database [persistence.OpenStore] opens for every other piece of
// non-git-tracked state, for the same reason knowledge-bases
// registrations are: persistence's own "Git-tracked specification
// files" requirement names a closed list that does not mention a
// verification context, and its "Everything else stays local-only"
// requirement is written to cover exactly what that list excludes.
func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS contexts (
			name          TEXT PRIMARY KEY,
			state         TEXT NOT NULL,
			superseded_by TEXT NOT NULL,
			created_at    TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS context_versions (
			context_name TEXT NOT NULL,
			seq          INTEGER NOT NULL,
			content      TEXT NOT NULL,
			created_at   TEXT NOT NULL,
			PRIMARY KEY (context_name, seq)
		);
	`)
	if err != nil {
		return fmt.Errorf("contextcomposition: creating schema: %w", err)
	}
	return nil
}

// ErrContextNotFound is returned by [GetContext] and every function
// built on it when name names no registered context.
var ErrContextNotFound = errors.New("contextcomposition: context not found")

// CreateContext registers a new, Active context named name. It fails
// if name is already registered - a caller wanting to replace one
// archives it first (see [ArchiveContext]), rather than overwriting an
// existing registration silently.
func CreateContext(db *sql.DB, name string) (Context, error) {
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO contexts (name, state, superseded_by, created_at) VALUES (?, ?, ?, ?)`,
		name, Active.String(), "", now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Context{}, fmt.Errorf("contextcomposition: creating context %q: %w", name, err)
	}
	return Context{ID: name, State: Active, CreatedAt: now}, nil
}

// GetContext returns the registered context named name, or
// [ErrContextNotFound] if there is none.
func GetContext(db *sql.DB, name string) (Context, error) {
	var (
		state, supersededBy, createdAt string
	)
	err := db.QueryRow(
		`SELECT state, superseded_by, created_at FROM contexts WHERE name = ?`, name,
	).Scan(&state, &supersededBy, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Context{}, fmt.Errorf("%w: %q", ErrContextNotFound, name)
	}
	if err != nil {
		return Context{}, fmt.Errorf("contextcomposition: getting context %q: %w", name, err)
	}

	parsedState, err := ParseLifecycleState(state)
	if err != nil {
		return Context{}, fmt.Errorf("contextcomposition: %w", err)
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Context{}, fmt.Errorf("contextcomposition: parsing created_at for %q: %w", name, err)
	}

	return Context{ID: name, State: parsedState, SupersededBy: supersededBy, CreatedAt: parsedCreatedAt}, nil
}

// ListContexts returns every registered context, oldest first - the
// "conjecture context list" command.
func ListContexts(db *sql.DB) ([]Context, error) {
	rows, err := db.Query(`SELECT name, state, superseded_by, created_at FROM contexts ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("contextcomposition: listing contexts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var contexts []Context
	for rows.Next() {
		var c Context
		var state, createdAt string
		if err := rows.Scan(&c.ID, &state, &c.SupersededBy, &createdAt); err != nil {
			return nil, fmt.Errorf("contextcomposition: listing contexts: %w", err)
		}
		parsedState, err := ParseLifecycleState(state)
		if err != nil {
			return nil, fmt.Errorf("contextcomposition: %w", err)
		}
		c.State = parsedState
		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("contextcomposition: parsing created_at for %q: %w", c.ID, err)
		}
		c.CreatedAt = parsedCreatedAt
		contexts = append(contexts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("contextcomposition: listing contexts: %w", err)
	}
	return contexts, nil
}

// AddVersion appends a new version of name's content, numbered one
// past whatever the highest existing sequence number for name is (1
// for its first version). It does not check that name is registered;
// a caller already holding a [Context] from [CreateContext] or
// [GetContext] knows that.
func AddVersion(db *sql.DB, name, content string) (Version, error) {
	var maxSeq sql.NullInt64
	if err := db.QueryRow(`SELECT MAX(seq) FROM context_versions WHERE context_name = ?`, name).Scan(&maxSeq); err != nil {
		return Version{}, fmt.Errorf("contextcomposition: adding a version to %q: %w", name, err)
	}
	seq := int(maxSeq.Int64) + 1

	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO context_versions (context_name, seq, content, created_at) VALUES (?, ?, ?, ?)`,
		name, seq, content, now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Version{}, fmt.Errorf("contextcomposition: adding a version to %q: %w", name, err)
	}
	return Version{ContextName: name, Seq: seq, Content: content, CreatedAt: now}, nil
}

// ErrVersionNotFound is returned by [VersionAt] and every function
// built on it when a context has no version at the asked-for sequence
// number (including "no version at all").
var ErrVersionNotFound = errors.New("contextcomposition: version not found")

// VersionAt returns name's version numbered seq, or
// [ErrVersionNotFound] if there is none.
func VersionAt(db *sql.DB, name string, seq int) (Version, error) {
	var content, createdAt string
	err := db.QueryRow(
		`SELECT content, created_at FROM context_versions WHERE context_name = ? AND seq = ?`, name, seq,
	).Scan(&content, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Version{}, fmt.Errorf("%w: %q@v%d", ErrVersionNotFound, name, seq)
	}
	if err != nil {
		return Version{}, fmt.Errorf("contextcomposition: getting %q@v%d: %w", name, seq, err)
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Version{}, fmt.Errorf("contextcomposition: parsing created_at for %q@v%d: %w", name, seq, err)
	}
	return Version{ContextName: name, Seq: seq, Content: content, CreatedAt: parsedCreatedAt}, nil
}

// LatestVersion returns name's highest-numbered version, or
// [ErrVersionNotFound] if it has none yet.
func LatestVersion(db *sql.DB, name string) (Version, error) {
	var seq sql.NullInt64
	if err := db.QueryRow(`SELECT MAX(seq) FROM context_versions WHERE context_name = ?`, name).Scan(&seq); err != nil {
		return Version{}, fmt.Errorf("contextcomposition: getting latest version of %q: %w", name, err)
	}
	if !seq.Valid {
		return Version{}, fmt.Errorf("%w: %q has no versions", ErrVersionNotFound, name)
	}
	return VersionAt(db, name, int(seq.Int64))
}

// Resume returns name's latest version - `conjecture context resume`:
// "the system restores that context's state for continued use."
func Resume(db *sql.DB, name string) (Version, error) {
	return LatestVersion(db, name)
}

// Open returns the specific version ref names - `conjecture context
// open`. Unlike Resume, Open always requires an explicit version:
// every one of the specification's own worked examples for "open"
// names one ("checkout@v4"), none of them bare, which Resume's own
// example already is.
func Open(db *sql.DB, ref VersionRef) (Version, error) {
	if ref.Seq == 0 {
		return Version{}, fmt.Errorf("contextcomposition: open requires an explicit version (e.g. %q@v1), got %q with none", ref.Name, ref.Name)
	}
	return VersionAt(db, ref.Name, ref.Seq)
}

// ArchiveContext persists [Supersede]'s result for name - `conjecture
// context`'s own archive-not-delete rule, made durable rather than
// just an in-memory transformation.
func ArchiveContext(db *sql.DB, name, supersededBy string) (Context, error) {
	ctx, err := GetContext(db, name)
	if err != nil {
		return Context{}, err
	}
	archived := Supersede(ctx, supersededBy)
	if _, err := db.Exec(
		`UPDATE contexts SET state = ?, superseded_by = ? WHERE name = ?`,
		archived.State.String(), archived.SupersededBy, name,
	); err != nil {
		return Context{}, fmt.Errorf("contextcomposition: archiving %q: %w", name, err)
	}
	return archived, nil
}

// Restore brings an archived context back to Active - `conjecture
// context restore`: "contexts are archived, never deleted... remains
// available via... restore." It fails if name is not currently
// Archived, rather than silently no-opping on an already-Active
// context.
func Restore(db *sql.DB, name string) (Context, error) {
	ctx, err := GetContext(db, name)
	if err != nil {
		return Context{}, err
	}
	if ctx.State != Archived {
		return Context{}, fmt.Errorf("contextcomposition: %q is not archived, nothing to restore", name)
	}

	ctx.State = Active
	ctx.SupersededBy = ""
	if _, err := db.Exec(
		`UPDATE contexts SET state = ?, superseded_by = ? WHERE name = ?`,
		Active.String(), "", name,
	); err != nil {
		return Context{}, fmt.Errorf("contextcomposition: restoring %q: %w", name, err)
	}
	return ctx, nil
}
