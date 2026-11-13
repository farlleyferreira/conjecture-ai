package main

import (
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
)

const statusUsage = "usage: conjecture status [file.cjtr]"

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

// runStatus implements "conjecture status [file.cjtr]" - a pure read:
// it never writes to a file's own "signed" block, the evidence
// ledger, or user-defined-laws' own status table, only to stdout.
// With a path, it reports that one file's own aggregated state
// (unchanged behavior). With none, it reports the project-wide
// aggregate instead - see [computeProjectStatus]'s own doc comment
// for why that gap existed and what this closes.
//
// This command, and its own argument shape, is this project's own
// invention: no specification names a verb for reading either
// aggregate back, the same disclosed situation check.go's own doc
// comment already notes for "conjecture check"'s shape.
func runStatus(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		_, _ = fmt.Fprintln(stderr, statusUsage)
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

	if len(args) == 0 {
		summary, err := computeProjectStatus(db, repoRoot)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
			return 1
		}
		printProjectStatus(stdout, summary)
		return 0
	}

	f, err := loadFicha(args[0])
	if err != nil {
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

// ProjectStatusSummary is the project-wide aggregate "status" itself
// cannot give (it only ever reads one file): every ".cjtr" file under
// "conjecture/phenomena/", summed by signed/confirmed state and by
// each judged hypothesis/law's own maturity [evidenceledger.Stage]
// and (for a law) [userdefinedlaws.Status]. Before this, answering
// "how many theses does this project have" meant calling "status" on
// every file by hand and adding the counts yourself - a real,
// disclosed gap this closes.
type ProjectStatusSummary struct {
	Files        int `json:"files"`
	Signed       int `json:"signed"`
	Unsigned     int `json:"unsigned"`
	Confirmed    int `json:"confirmed"`
	NeverChecked int `json:"never_checked"`
	// Stage counts, by [evidenceledger.Stage.String]'s own four
	// values - every *checked* hypothesis/law falls into exactly one.
	Hypothesis int `json:"hypothesis_stage"`
	Thesis     int `json:"thesis_stage"`
	Theory     int `json:"theory_stage"`
	Law        int `json:"law_stage"`
	// LawStatuses counts every user-defined law by its own
	// [userdefinedlaws.Status.String] - a plain hypothesis has none.
	LawStatuses map[string]int `json:"law_statuses,omitempty"`
	// Errors names every file this walk could not fully read (a
	// malformed ".cjtr", a rule body [computeStatus] could not parse)
	// - disclosed rather than silently excluded from the counts above,
	// the same "disclose, don't swallow" posture [Status.Notes]
	// already takes per file.
	Errors []string `json:"errors,omitempty"`
}

// computeProjectStatus walks every ".cjtr" file under
// "conjecture/phenomena/" (cjtr-language's own path convention) and
// sums each one's own [computeStatus] into one [ProjectStatusSummary].
// A repository with no "conjecture/phenomena/" directory yet (before
// "conjecture init", or one that genuinely has no phenomena) is not an
// error - an empty summary is a legitimate, honestly-reportable state.
func computeProjectStatus(db *sql.DB, repoRoot string) (ProjectStatusSummary, error) {
	out := ProjectStatusSummary{LawStatuses: map[string]int{}}
	root := filepath.Join(repoRoot, "conjecture", "phenomena")

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".cjtr" {
			return nil
		}
		out.Files++

		f, loadErr := loadFicha(path)
		if loadErr != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", path, loadErr))
			return nil
		}
		if f.signed != nil {
			out.Signed++
			if f.signed.Confirmed {
				out.Confirmed++
			}
		} else {
			out.Unsigned++
		}

		status, statusErr := computeStatus(db, f)
		if statusErr != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", path, statusErr))
			return nil
		}
		for _, e := range status.Entries {
			if !e.Checked {
				out.NeverChecked++
				continue
			}
			switch e.Stage {
			case evidenceledger.Hypothesis.String():
				out.Hypothesis++
			case evidenceledger.Thesis.String():
				out.Thesis++
			case evidenceledger.Theory.String():
				out.Theory++
			case evidenceledger.Law.String():
				out.Law++
			}
			if e.IsLaw && e.LawStatus != "" {
				out.LawStatuses[e.LawStatus]++
			}
		}
		return nil
	})
	if walkErr != nil {
		return ProjectStatusSummary{}, walkErr
	}
	return out, nil
}

// printProjectStatus renders s the same plain, one-line-per-fact way
// [printStatus] renders a single file's own state.
func printProjectStatus(stdout io.Writer, s ProjectStatusSummary) {
	_, _ = fmt.Fprintf(stdout, "project: %d files (%d signed, %d unsigned; %d confirmed)\n", s.Files, s.Signed, s.Unsigned, s.Confirmed)
	_, _ = fmt.Fprintf(stdout, "stages: hypothesis %d, thesis %d, theory %d, law %d, never checked %d\n", s.Hypothesis, s.Thesis, s.Theory, s.Law, s.NeverChecked)

	if len(s.LawStatuses) > 0 {
		keys := make([]string, 0, len(s.LawStatuses))
		for k := range s.LawStatuses {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = fmt.Sprintf("%s %d", k, s.LawStatuses[k])
		}
		_, _ = fmt.Fprintf(stdout, "law statuses: %s\n", strings.Join(parts, ", "))
	}

	for _, e := range s.Errors {
		_, _ = fmt.Fprintf(stdout, "NOTE: %s\n", e)
	}
}
