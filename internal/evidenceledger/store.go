package evidenceledger

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"
)

// Evidence is one append-only record of a hypothesis's evidence profile
// (evidence-ledger spec, "Evidence profile per hypothesis": "what was
// attacked, what survived, what remains open, what nobody has checked
// yet, who/what produced each result, and the validity hash of that
// result"). Attacked, Survived, Open, and Unchecked are each free text
// (a caller may encode a structured value, e.g. JSON, into one): the
// specification enumerates these six things a profile records, but
// never gives any of them its own sub-schema, so this does not invent
// one.
type Evidence struct {
	// ID is the record's own row id, assigned by [Append]; zero on a
	// record not yet stored.
	ID int64
	// HypothesisID is the hypothesis's composite ID (area::phenomenon::
	// rule-id::hypothesis-id, per [cjtr.ComposeID]), identifying which
	// hypothesis's append-only history this record belongs to.
	HypothesisID string
	Stage        Stage
	// Theorem reports whether this result is a formally proved
	// statement - see [Stage]'s doc comment: available at any stage,
	// not a position on the maturity scale itself.
	Theorem bool

	Attacked, Survived, Open, Unchecked string
	// ProducedBy names who or what produced this result (a person, an
	// SLM, a specific tool run, …).
	ProducedBy string

	// ValidityHash chains this record to the specification, code,
	// harness, SLM, and operator versions it was produced against (see
	// [ValidityHash]).
	ValidityHash []byte
	// ChainHash chains this record to the previous record in the same
	// hypothesis's history (see [persistence.ChainHash]; nil for
	// previous on a hypothesis's first record).
	ChainHash []byte

	CreatedAt time.Time
}

// EnsureSchema creates the evidence table if it does not already exist.
// It is safe to call every time a store is opened.
func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS evidence (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			hypothesis_id  TEXT NOT NULL,
			stage          TEXT NOT NULL,
			theorem        INTEGER NOT NULL,
			attacked       TEXT NOT NULL,
			survived       TEXT NOT NULL,
			open           TEXT NOT NULL,
			unchecked      TEXT NOT NULL,
			produced_by    TEXT NOT NULL,
			validity_hash  BLOB NOT NULL,
			chain_hash     BLOB NOT NULL,
			created_at     TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS evidence_hypothesis_id ON evidence (hypothesis_id, id);
	`)
	if err != nil {
		return fmt.Errorf("evidenceledger: creating schema: %w", err)
	}
	return nil
}

// Append writes e as the next append-only record in its HypothesisID's
// history, computing ChainHash from the previous record for that same
// hypothesis (or the hash of an empty byte slice, for its first) and
// e's own content - never overwriting a previous record, per the
// persistence spec's "append-only... chained by hash to the previous
// one" requirement. It sets and returns e's ID, ChainHash, and
// CreatedAt (now, in UTC); e's other fields are stored exactly as
// given.
func Append(db *sql.DB, e Evidence) (Evidence, error) {
	previousChainHash, err := latestChainHash(db, e.HypothesisID)
	if err != nil {
		return Evidence{}, err
	}

	e.CreatedAt = time.Now().UTC()
	e.ChainHash = chainHashOf(previousChainHash, e)

	res, err := db.Exec(
		`INSERT INTO evidence
			(hypothesis_id, stage, theorem, attacked, survived, open, unchecked, produced_by, validity_hash, chain_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.HypothesisID, e.Stage.String(), e.Theorem, e.Attacked, e.Survived, e.Open, e.Unchecked, e.ProducedBy,
		e.ValidityHash, e.ChainHash, e.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Evidence{}, fmt.Errorf("evidenceledger: appending evidence: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Evidence{}, fmt.Errorf("evidenceledger: appending evidence: %w", err)
	}
	e.ID = id
	return e, nil
}

// chainHashOf computes an evidence record's chain hash from the
// previous record's chain hash and this record's own content - every
// field a caller controls, so two records with identical content never
// collide with two genuinely different ones by coincidence.
func chainHashOf(previousChainHash []byte, e Evidence) []byte {
	h := sha256.New()
	h.Write(previousChainHash)
	for _, s := range []string{e.HypothesisID, e.Stage.String(), e.Attacked, e.Survived, e.Open, e.Unchecked, e.ProducedBy} {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	if e.Theorem {
		h.Write([]byte{1})
	} else {
		h.Write([]byte{0})
	}
	h.Write(e.ValidityHash)
	return h.Sum(nil)
}

func latestChainHash(db *sql.DB, hypothesisID string) ([]byte, error) {
	var chainHash []byte
	err := db.QueryRow(
		`SELECT chain_hash FROM evidence WHERE hypothesis_id = ? ORDER BY id DESC LIMIT 1`,
		hypothesisID,
	).Scan(&chainHash)
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("evidenceledger: reading previous evidence: %w", err)
	default:
		return chainHash, nil
	}
}

// Latest returns the most recent evidence record for hypothesisID, or
// (Evidence{}, false, nil) if it has none yet.
func Latest(db *sql.DB, hypothesisID string) (Evidence, bool, error) {
	row := db.QueryRow(
		`SELECT id, stage, theorem, attacked, survived, open, unchecked, produced_by, validity_hash, chain_hash, created_at
		 FROM evidence WHERE hypothesis_id = ? ORDER BY id DESC LIMIT 1`,
		hypothesisID,
	)
	e, err := scanEvidence(row, hypothesisID)
	switch {
	case err == sql.ErrNoRows:
		return Evidence{}, false, nil
	case err != nil:
		return Evidence{}, false, err
	default:
		return e, true, nil
	}
}

// History returns every evidence record for hypothesisID, oldest first.
func History(db *sql.DB, hypothesisID string) ([]Evidence, error) {
	rows, err := db.Query(
		`SELECT id, stage, theorem, attacked, survived, open, unchecked, produced_by, validity_hash, chain_hash, created_at
		 FROM evidence WHERE hypothesis_id = ? ORDER BY id ASC`,
		hypothesisID,
	)
	if err != nil {
		return nil, fmt.Errorf("evidenceledger: reading history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var history []Evidence
	for rows.Next() {
		e, err := scanEvidence(rows, hypothesisID)
		if err != nil {
			return nil, fmt.Errorf("evidenceledger: reading history: %w", err)
		}
		history = append(history, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("evidenceledger: reading history: %w", err)
	}
	return history, nil
}

// rowScanner is the common subset of *sql.Row and *sql.Rows this
// package uses, so scanEvidence works for either.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvidence(row rowScanner, hypothesisID string) (Evidence, error) {
	var (
		e         Evidence
		stage     string
		createdAt string
	)
	if err := row.Scan(&e.ID, &stage, &e.Theorem, &e.Attacked, &e.Survived, &e.Open, &e.Unchecked, &e.ProducedBy, &e.ValidityHash, &e.ChainHash, &createdAt); err != nil {
		return Evidence{}, err
	}
	e.HypothesisID = hypothesisID

	parsedStage, err := ParseStage(stage)
	if err != nil {
		return Evidence{}, fmt.Errorf("evidenceledger: %w", err)
	}
	e.Stage = parsedStage

	createdAtTime, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Evidence{}, fmt.Errorf("evidenceledger: parsing created_at: %w", err)
	}
	e.CreatedAt = createdAtTime

	return e, nil
}
