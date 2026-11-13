package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
)

const statusUsage = "usage: conjecture status <file.cjtr>"

// StatusEntry is one judged hypothesis or user-defined law's own
// current state, aggregated read-only from the evidence ledger (and,
// for a law, user-defined-laws' own status table) - never written
// here. ID is never recorded for a hypothesis with no declared "id"
// (see check.go's own recordEvidence), so no entry exists for one;
// that is reported as a Note on [Status] instead of a zero-value
// entry a caller could mistake for "checked, but empty."
type StatusEntry struct {
	ID         string `json:"id"`
	IsLaw      bool   `json:"is_law"`
	Checked    bool   `json:"checked"`
	Stage      string `json:"stage,omitempty"`
	Attacked   string `json:"attacked,omitempty"`
	Survived   string `json:"survived,omitempty"`
	Open       string `json:"open,omitempty"`
	Unchecked  string `json:"unchecked,omitempty"`
	ProducedBy string `json:"produced_by,omitempty"`
	CheckedAt  string `json:"checked_at,omitempty"`
	// LawStatus is empty when IsLaw is false - a plain hypothesis has
	// no such status ([userdefinedlaws] only tracks laws).
	LawStatus string `json:"law_status,omitempty"`
}

// Status is a specification file's own aggregated lifecycle state -
// signed (the file's own "signed" block, the same one "conjecture
// sign" writes), then checked and, for a law, promoted (both read
// from the local store check.go's own recordEvidence already writes
// to). This is this project's own answer to the gap "o que falta pra
// tornar o uso fluido" named: today this state is scattered across
// three different reads (the file itself, "read_evidence" per
// hypothesis id, nothing at all for a law's own status); this is one
// read that aggregates all three, so an agent does not have to
// already know a file's every hypothesis id to ask "where does this
// specification stand."
type Status struct {
	PhenomenonID string        `json:"phenomenon_id"`
	Signed       bool          `json:"signed"`
	SignedBy     string        `json:"signed_by,omitempty"`
	SignedTool   string        `json:"signed_tool,omitempty"`
	SignedDate   string        `json:"signed_date,omitempty"`
	Confirmed    bool          `json:"confirmed"`
	Entries      []StatusEntry `json:"entries"`
	// Notes carries anything this read could not fully resolve (a
	// rule body that failed to parse into hypotheses/laws, a
	// hypothesis with no declared "id") - disclosed rather than
	// silently dropped, the same honesty discipline every other
	// command in this codebase already follows.
	Notes []string `json:"notes,omitempty"`
}

// runStatus implements "conjecture status <file.cjtr>" - a pure read:
// it never writes to the file's own "signed" block, the evidence
// ledger, or user-defined-laws' own status table, only to stdout.
//
// This command, and its own single-argument shape, is this project's
// own invention: no specification names a verb for reading this
// aggregate back, the same disclosed situation check.go's own doc
// comment already notes for "conjecture check"'s shape.
func runStatus(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, statusUsage)
		return 1
	}
	path := args[0]

	f, err := loadFicha(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	db, err := persistence.OpenStore(repoRoot)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()
	if err := evidenceledger.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	if err := userdefinedlaws.EnsureSchema(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	status, err := computeStatus(db, f)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	printStatus(stdout, status)
	return 0
}

// computeStatus is runStatus's and the MCP "status" tool's own shared
// read: given an already-opened store (schemas already ensured by the
// caller) and an already-loaded file, it never touches the
// filesystem or opens its own connection, so a caller that already
// has both open (newMCPServer's own shared db, in particular) does
// not pay for a second one.
func computeStatus(db *sql.DB, f loadedFicha) (Status, error) {
	out := Status{PhenomenonID: f.phenomenonID}
	if f.signed != nil {
		out.Signed = true
		out.SignedBy = f.signed.User
		out.SignedTool = f.signed.Tool
		out.SignedDate = f.signed.Date
		out.Confirmed = f.signed.Confirmed
	}

	add := func(hyps []cjtr.Hypothesis, compositeID string, isLaw bool) error {
		for _, h := range hyps {
			if h.ID == "" {
				out.Notes = append(out.Notes, fmt.Sprintf("%s: %q has no declared \"id\", nothing recorded against it", compositeID, h.Name))
				continue
			}
			entry, err := statusEntryFor(db, compositeID, h, isLaw)
			if err != nil {
				return err
			}
			out.Entries = append(out.Entries, entry)
		}
		return nil
	}

	if err := add(f.fileLaws, f.phenomenonID, true); err != nil {
		return Status{}, err
	}
	for _, rule := range f.rules {
		hyps, err := cjtr.Hypotheses(rule.Body, f.concepts)
		if err != nil {
			out.Notes = append(out.Notes, fmt.Sprintf("%s: %v", rule.CompositeID, err))
			continue
		}
		laws, err := cjtr.UserDefinedLaws(rule.Body, f.concepts)
		if err != nil {
			out.Notes = append(out.Notes, fmt.Sprintf("%s: %v", rule.CompositeID, err))
			continue
		}
		if err := add(hyps, rule.CompositeID, false); err != nil {
			return Status{}, err
		}
		if err := add(laws, rule.CompositeID, true); err != nil {
			return Status{}, err
		}
	}
	return out, nil
}

func statusEntryFor(db *sql.DB, compositeID string, h cjtr.Hypothesis, isLaw bool) (StatusEntry, error) {
	hypothesisID := compositeID + "::" + h.ID
	entry := StatusEntry{ID: hypothesisID, IsLaw: isLaw}

	latest, found, err := evidenceledger.Latest(db, hypothesisID)
	if err != nil {
		return StatusEntry{}, err
	}
	if found {
		entry.Checked = true
		entry.Stage = latest.Stage.String()
		entry.Attacked = latest.Attacked
		entry.Survived = latest.Survived
		entry.Open = latest.Open
		entry.Unchecked = latest.Unchecked
		entry.ProducedBy = latest.ProducedBy
		entry.CheckedAt = latest.CreatedAt.Format(time.RFC3339)
	}

	if isLaw {
		status, err := userdefinedlaws.GetStatus(db, hypothesisID)
		if err != nil {
			return StatusEntry{}, err
		}
		entry.LawStatus = status.String()
	}
	return entry, nil
}

// printStatus renders s the same way every other command in this
// codebase reports to stdout: one line per fact, plain text, no
// framing - the CLI's own delivery of the same result the MCP
// "status" tool returns as structured JSON (cli-mcp-integration's
// "Verdicts delivered back through the same channel").
func printStatus(stdout io.Writer, s Status) {
	if s.Signed {
		_, _ = fmt.Fprintf(stdout, "%s: signed by %s (tool %s, date %s, confirmed %v)\n", s.PhenomenonID, s.SignedBy, s.SignedTool, s.SignedDate, s.Confirmed)
	} else {
		_, _ = fmt.Fprintf(stdout, "%s: not signed\n", s.PhenomenonID)
	}

	for _, e := range s.Entries {
		if !e.Checked {
			_, _ = fmt.Fprintf(stdout, "%s: never checked\n", e.ID)
			continue
		}
		_, _ = fmt.Fprintf(stdout, "%s: stage=%s", e.ID, e.Stage)
		if e.Attacked != "" {
			_, _ = fmt.Fprintf(stdout, " attacked=%q", e.Attacked)
		}
		if e.Survived != "" {
			_, _ = fmt.Fprintf(stdout, " survived=%q", e.Survived)
		}
		if e.Open != "" {
			_, _ = fmt.Fprintf(stdout, " open=%q", e.Open)
		}
		if e.LawStatus != "" {
			_, _ = fmt.Fprintf(stdout, " law_status=%s", e.LawStatus)
		}
		_, _ = fmt.Fprintln(stdout)
	}

	for _, n := range s.Notes {
		_, _ = fmt.Fprintf(stdout, "NOTE: %s\n", n)
	}
}
