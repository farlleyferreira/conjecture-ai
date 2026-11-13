// Package decisionledger implements persistence's missing third
// append-only table: "Append-only, hash-chained tables for evidence"
// names evidence, decisions, and configuration history; evidence
// (internal/evidenceledger) and configuration history (internal/
// teamconfig) already exist, decisions did not. A decision records an
// explicit human determination this codebase already asks for -
// "conjecture promote theory"/"conjecture promote law" (promotion) and
// "conjecture sign --confirmed" (a drafted candidate's human review) -
// which, before this package, left no attributable record outside
// stdout.
//
// A decision is not evidence: it is a determination about evidence
// (or about a reviewed candidate), not an observation of a rule's own
// behavior, so this is its own table rather than folded into
// evidenceledger - mixing the two would blur the evidence profile
// promotion itself weighs.
package decisionledger

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

// Decision is one append-only record of a human determination.
type Decision struct {
	// ID is the record's own row id, assigned by [Append]; zero on a
	// record not yet stored.
	ID int64
	// Kind names what kind of determination this is - this package's
	// own three wired producers use "promotion" (conjecture promote
	// theory/law), and "candidate-review" (conjecture sign
	// --confirmed); a caller may use its own kind for a determination
	// this package does not already wire.
	Kind string
	// Subject names what the decision is about - a hypothesis's
	// composite ID for a promotion, a .cjtr file's own path for a
	// candidate review.
	Subject string
	// Determination is the decision itself (e.g. "Theory", "Law",
	// "accepted").
	Determination string
	// Rationale may be empty - a rejected candidate needs none (see
	// [Append]'s own doc comment).
	Rationale string
	// DecidedBy names who made the determination - a person's own
	// name/username, never a tool or process: a decision this codebase
	// asks a *human* for is only meaningfully attributable to one.
	DecidedBy string
	// ChainHash chains this record to the previous record in the
	// whole decisions table (see [persistence.ChainHash]; nil for the
	// table's first record) - one sequential ledger across every kind
	// and subject, not partitioned per subject the way evidence's own
	// chain is per hypothesis: a decision log's own integrity question
	// ("has any decision, of any kind, been tampered with or
	// reordered") is about the whole table, which [Verify] checks in
	// one pass.
	ChainHash []byte
	CreatedAt time.Time
}

// ErrIncompleteDecision is returned by [Append] when Kind, Subject,
// Determination, or DecidedBy is empty - "Append refuses an empty
// kind/subject/determination/decider, so an unattributed decision
// cannot be recorded." Rationale alone may be empty.
var ErrIncompleteDecision = errors.New("decisionledger: kind, subject, determination, and decided by are all required")

// EnsureSchema creates the decisions table if it does not already
// exist. It is safe to call every time a store is opened.
func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS decisions (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			kind           TEXT NOT NULL,
			subject        TEXT NOT NULL,
			determination  TEXT NOT NULL,
			rationale      TEXT NOT NULL,
			decided_by     TEXT NOT NULL,
			chain_hash     BLOB NOT NULL,
			created_at     TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("decisionledger: creating schema: %w", err)
	}
	return nil
}

// decisionPayload is a [Decision]'s own canonical, hashable content -
// every field a caller controls, in a fixed field order (encoding/json
// marshals struct fields in declaration order, never alphabetically),
// so the same decision content always hashes the same way and two
// different decisions never collide by coincidence.
type decisionPayload struct {
	Kind, Subject, Determination, Rationale, DecidedBy, CreatedAt string
}

func payloadOf(d Decision) ([]byte, error) {
	payload, err := json.Marshal(decisionPayload{
		Kind: d.Kind, Subject: d.Subject, Determination: d.Determination,
		Rationale: d.Rationale, DecidedBy: d.DecidedBy,
		CreatedAt: d.CreatedAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		return nil, fmt.Errorf("decisionledger: encoding decision: %w", err)
	}
	return payload, nil
}

// Append writes d as the next append-only record in the decisions
// table, computing ChainHash from the previous record's own chain hash
// (via [persistence.ChainHash]) and d's own content. It sets and
// returns d's ID, ChainHash, and CreatedAt (now, in UTC); d's other
// fields are stored exactly as given.
func Append(db *sql.DB, d Decision) (Decision, error) {
	if d.Kind == "" || d.Subject == "" || d.Determination == "" || d.DecidedBy == "" {
		return Decision{}, ErrIncompleteDecision
	}

	previous, err := latestChainHash(db)
	if err != nil {
		return Decision{}, err
	}

	d.CreatedAt = time.Now().UTC()
	payload, err := payloadOf(d)
	if err != nil {
		return Decision{}, err
	}
	d.ChainHash = persistence.ChainHash(previous, payload)

	res, err := db.Exec(
		`INSERT INTO decisions (kind, subject, determination, rationale, decided_by, chain_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		d.Kind, d.Subject, d.Determination, d.Rationale, d.DecidedBy, d.ChainHash, d.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Decision{}, fmt.Errorf("decisionledger: appending decision: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Decision{}, fmt.Errorf("decisionledger: appending decision: %w", err)
	}
	d.ID = id
	return d, nil
}

func latestChainHash(db *sql.DB) ([]byte, error) {
	var chainHash []byte
	err := db.QueryRow(`SELECT chain_hash FROM decisions ORDER BY id DESC LIMIT 1`).Scan(&chainHash)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("decisionledger: reading previous decision: %w", err)
	default:
		return chainHash, nil
	}
}

// List returns every recorded decision, oldest first.
func List(db *sql.DB) ([]Decision, error) {
	rows, err := db.Query(
		`SELECT id, kind, subject, determination, rationale, decided_by, chain_hash, created_at
		FROM decisions ORDER BY id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("decisionledger: listing decisions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var decisions []Decision
	for rows.Next() {
		var d Decision
		var createdAt string
		if err := rows.Scan(&d.ID, &d.Kind, &d.Subject, &d.Determination, &d.Rationale, &d.DecidedBy, &d.ChainHash, &createdAt); err != nil {
			return nil, fmt.Errorf("decisionledger: listing decisions: %w", err)
		}
		parsed, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("decisionledger: parsing created_at for decision %d: %w", d.ID, err)
		}
		d.CreatedAt = parsed
		decisions = append(decisions, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("decisionledger: listing decisions: %w", err)
	}
	return decisions, nil
}

// Verify recomputes the whole chain from every recorded decision's own
// content and reports whether it still matches each stored ChainHash,
// in order - "a corrupted or reordered history is detectable." A
// decision whose own row was edited after being written, or removed,
// or reordered, breaks the chain at that point and every link after
// it, so Verify's own false result does not say which row was altered -
// only that the ledger as a whole is no longer trustworthy.
func Verify(db *sql.DB) (bool, error) {
	decisions, err := List(db)
	if err != nil {
		return false, err
	}

	var previous []byte
	for _, d := range decisions {
		payload, err := payloadOf(d)
		if err != nil {
			return false, err
		}
		want := persistence.ChainHash(previous, payload)
		if !bytes.Equal(want, d.ChainHash) {
			return false, nil
		}
		previous = d.ChainHash
	}
	return true, nil
}
