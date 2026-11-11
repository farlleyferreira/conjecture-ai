package contextengine

import (
	"database/sql"
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
)

// EnsureSchema creates the context_node table if it does not already
// exist. It is safe to call every time a store is opened.
//
// Queued here, in the same local, per-repository SQLite database
// [persistence.OpenStore] opens for every other piece of run-time-
// derived state (evidence-ledger's, work-queue's, optimization-suite's
// own stores), for the same reason those are: a node's weight,
// relevance, and confidence are the context engine's own computed
// output, not specification text, so they belong outside anything
// git-tracked.
//
// The three axes are three separate nullable columns, never one
// combined column - the storage-layer expression of "the system SHALL
// NOT combine them into a single multiplied score." A node may have
// some axes set and others not yet known; each column tracks that
// independently rather than forcing a placeholder value on an axis
// nothing has reported yet.
func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS context_node (
			node_id    TEXT PRIMARY KEY,
			weight     INTEGER,
			relevance  INTEGER,
			confidence INTEGER
		);
	`)
	if err != nil {
		return fmt.Errorf("contextengine: creating schema: %w", err)
	}
	return nil
}

// Axes is the three independent values [GetAxes] reports for a node.
// Each pointer is nil when that axis has never been set for this node -
// genuinely unknown, not the same as Low.
type Axes struct {
	Weight     *evidenceledger.Weight
	Relevance  *evidenceledger.Weight
	Confidence *evidenceledger.Weight
}

// SetWeight persists weight as nodeID's weight axis, leaving its
// relevance and confidence axes (if any) untouched.
func SetWeight(db *sql.DB, nodeID string, weight evidenceledger.Weight) error {
	return setAxis(db, nodeID, "weight", weight)
}

// SetRelevance persists relevance as nodeID's relevance axis, leaving
// its weight and confidence axes (if any) untouched.
func SetRelevance(db *sql.DB, nodeID string, relevance evidenceledger.Weight) error {
	return setAxis(db, nodeID, "relevance", relevance)
}

// SetConfidence persists confidence as nodeID's confidence axis,
// leaving its weight and relevance axes (if any) untouched.
func SetConfidence(db *sql.DB, nodeID string, confidence evidenceledger.Weight) error {
	return setAxis(db, nodeID, "confidence", confidence)
}

// setAxis upserts a single named column for nodeID. column is always
// one of this file's own three fixed literals, never caller input, so
// building the statement text with it is not a SQL-injection risk.
func setAxis(db *sql.DB, nodeID, column string, value evidenceledger.Weight) error {
	stmt := fmt.Sprintf(
		`INSERT INTO context_node (node_id, %s) VALUES (?, ?)
		 ON CONFLICT(node_id) DO UPDATE SET %s = excluded.%s`,
		column, column, column,
	)
	if _, err := db.Exec(stmt, nodeID, int(value)); err != nil {
		return fmt.Errorf("contextengine: setting %s for node %q: %w", column, nodeID, err)
	}
	return nil
}

// GetAxes reports nodeID's three axes as they currently stand. found
// is false only when nodeID has never had any axis set at all - a node
// with, say, only a weight ever recorded still reports found = true,
// with Relevance and Confidence left nil.
func GetAxes(db *sql.DB, nodeID string) (axes Axes, found bool, err error) {
	var weight, relevance, confidence sql.NullInt64
	err = db.QueryRow(
		`SELECT weight, relevance, confidence FROM context_node WHERE node_id = ?`,
		nodeID,
	).Scan(&weight, &relevance, &confidence)
	if err == sql.ErrNoRows {
		return Axes{}, false, nil
	}
	if err != nil {
		return Axes{}, false, fmt.Errorf("contextengine: reading axes for node %q: %w", nodeID, err)
	}

	if weight.Valid {
		w := evidenceledger.Weight(weight.Int64)
		axes.Weight = &w
	}
	if relevance.Valid {
		r := evidenceledger.Weight(relevance.Int64)
		axes.Relevance = &r
	}
	if confidence.Valid {
		c := evidenceledger.Weight(confidence.Int64)
		axes.Confidence = &c
	}
	return axes, true, nil
}
