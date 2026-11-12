package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/version"
)

const promoteUsage = "usage: conjecture promote theory <file.cjtr> [--mechanism-confirmed] [--different-family-reproduced]\n" +
	"       conjecture promote law <file.cjtr> <hypothesis-id> [--hidden-samples-conformant] [--new-code-versions-conformant]"

// promotionProducedBy is the [evidenceledger.Evidence.ProducedBy]
// every record this command appends carries - an administrative
// promotion decision, not a conformance or attack result, the same
// "who or what produced this result" vocabulary Evidence.ProducedBy
// already uses for a tool run.
const promotionProducedBy = "conjecture promote"

// runPromote implements "conjecture promote <subcommand>" -
// evidence-ledger's own "Thesis promoted to Theory" and "Theory
// promoted to Law" scenarios, wired into the CLI for the first time.
//
// Honesty note: neither promotion is fully derivable from what this
// codebase already tracks. "The mechanism is confirmed" and "a second
// round with a different-family model reproduces the result" (Theory)
// and "conformant against hidden samples" and "across new code
// versions" (Law) are accepted as caller-attested flags - see
// [evidenceledger.PromoteToTheory]'s and [evidenceledger.PromoteToLaw]'s
// own doc comments for why this codebase cannot derive either pair on
// its own. "Every thesis in a phenomenon is confirmed" and "swarm/
// attack robustness checks find no violation" are derived for real,
// from already-recorded evidence ([evidenceledger.AllAtStage],
// [evidenceledger.SwarmAttackRobust]).
func runPromote(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, promoteUsage)
		return 1
	}
	subcommand, rest := args[0], args[1:]
	switch subcommand {
	case "theory":
		return runPromoteTheory(rest, stdout, stderr)
	case "law":
		return runPromoteLaw(rest, stdout, stderr)
	default:
		_, _ = fmt.Fprintln(stderr, promoteUsage)
		return 1
	}
}

// runPromoteTheory parses "theory"'s own arguments and loads the real
// file/store - the thin, real-path wrapper around [promoteTheory].
func runPromoteTheory(args []string, stdout, stderr io.Writer) int {
	var path string
	var mechanismConfirmed, differentFamilyReproduced bool
	var positional []string
	for _, arg := range args {
		switch arg {
		case "--mechanism-confirmed":
			mechanismConfirmed = true
		case "--different-family-reproduced":
			differentFamilyReproduced = true
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 1 {
		_, _ = fmt.Fprintln(stderr, promoteUsage)
		return 1
	}
	path = positional[0]

	f, err := loadFicha(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	db, err := openPromoteStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()

	return promoteTheory(db, f, mechanismConfirmed, differentFamilyReproduced, stdout, stderr)
}

// openPromoteStore opens the local, per-repository store, assuming
// the current working directory is the repository root - the same
// assumption every other command reading this store already makes.
func openPromoteStore() (*sql.DB, error) {
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	db, err := persistence.OpenStore(repoRoot)
	if err != nil {
		return nil, err
	}
	if err := evidenceledger.EnsureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// promoteTheory implements "conjecture promote theory"'s own real
// logic against db: gathers every hypothesis/law composite ID in f's
// own phenomenon, checks "every thesis in a phenomenon is confirmed"
// and "swarm/attack robustness checks find no violation" for real
// against recorded evidence, applies [evidenceledger.PromoteToTheory]
// with mechanismConfirmed/differentFamilyReproduced as given, and - if
// every condition holds - appends a Theory-stage evidence record for
// every one of those hypotheses/laws ("promotes the phenomenon's
// theses to Theory," plural).
func promoteTheory(db *sql.DB, f loadedFicha, mechanismConfirmed, differentFamilyReproduced bool, stdout, stderr io.Writer) int {
	ids := allHypothesisIDs(f)
	if len(ids) == 0 {
		_, _ = fmt.Fprintf(stdout, "%s: no hypotheses or laws to promote\n", f.phenomenonID)
		return 0
	}

	allConfirmed, err := evidenceledger.AllAtStage(db, ids, evidenceledger.Thesis)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	swarmRobust := true
	for _, id := range ids {
		robust, err := evidenceledger.SwarmAttackRobust(db, id, adversarialSLMProducedBy)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
			return 1
		}
		if !robust {
			swarmRobust = false
			break
		}
	}

	if !evidenceledger.PromoteToTheory(allConfirmed, mechanismConfirmed, differentFamilyReproduced, swarmRobust) {
		_, _ = fmt.Fprintf(stdout, "%s: not promoted to Theory - all theses confirmed: %v, mechanism confirmed: %v, different-family reproduction: %v, swarm/attack robust: %v\n",
			f.phenomenonID, allConfirmed, mechanismConfirmed, differentFamilyReproduced, swarmRobust)
		return 0
	}

	specVersion := fmt.Sprintf("%x", sha256.Sum256(f.raw))
	validityHash := evidenceledger.ValidityHash(specVersion, "", version.String(), "", "")
	for _, id := range ids {
		evidence := evidenceledger.Evidence{
			HypothesisID: id,
			Stage:        evidenceledger.Theory,
			ProducedBy:   promotionProducedBy,
			Survived:     "promoted to Theory: every thesis confirmed, mechanism confirmed, different-family reproduction, and swarm/attack robustness all held",
			ValidityHash: validityHash,
		}
		if _, err := evidenceledger.Append(db, evidence); err != nil {
			_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
			return 1
		}
	}
	_, _ = fmt.Fprintf(stdout, "%s: promoted %d hypothesis/law(s) to Theory\n", f.phenomenonID, len(ids))
	return 0
}

// allHypothesisIDs returns every hypothesis's and law's own composite
// ID in f - f's own file-level laws, plus every rule's hypotheses and
// user-defined laws - the same compositeID+"::"+h.ID scheme
// [findHypothesisByID] already uses for one. Only a hypothesis/law
// that declares an "id" is included, the same "no id, nothing to
// record" rule [recordEvidence] itself already applies.
func allHypothesisIDs(f loadedFicha) []string {
	var ids []string
	for _, h := range f.fileLaws {
		if h.ID != "" {
			ids = append(ids, f.phenomenonID+"::"+h.ID)
		}
	}
	for _, rule := range f.rules {
		hyps, err := cjtr.Hypotheses(rule.Body, f.concepts)
		if err == nil {
			for _, h := range hyps {
				if h.ID != "" {
					ids = append(ids, rule.CompositeID+"::"+h.ID)
				}
			}
		}
		laws, err := cjtr.UserDefinedLaws(rule.Body, f.concepts)
		if err == nil {
			for _, h := range laws {
				if h.ID != "" {
					ids = append(ids, rule.CompositeID+"::"+h.ID)
				}
			}
		}
	}
	return ids
}

// runPromoteLaw parses "law"'s own arguments and loads the real
// file/store - the thin, real-path wrapper around [promoteLaw].
func runPromoteLaw(args []string, stdout, stderr io.Writer) int {
	var hiddenSamplesConformant, newCodeVersionsConformant bool
	var positional []string
	for _, arg := range args {
		switch arg {
		case "--hidden-samples-conformant":
			hiddenSamplesConformant = true
		case "--new-code-versions-conformant":
			newCodeVersionsConformant = true
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 2 {
		_, _ = fmt.Fprintln(stderr, promoteUsage)
		return 1
	}
	path, hypothesisID := positional[0], positional[1]

	f, err := loadFicha(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	db, err := openPromoteStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()

	return promoteLaw(db, f, hypothesisID, hiddenSamplesConformant, newCodeVersionsConformant, stdout, stderr)
}

// promoteLaw implements "conjecture promote law"'s own real logic
// against db: looks up the named hypothesis's own IsUniversal fact
// for real, applies [evidenceledger.PromoteToLaw] with
// hiddenSamplesConformant/newCodeVersionsConformant as given, and - if
// every condition holds - appends a Law-stage evidence record.
func promoteLaw(db *sql.DB, f loadedFicha, hypothesisID string, hiddenSamplesConformant, newCodeVersionsConformant bool, stdout, stderr io.Writer) int {
	h, compositeID, _, found := findHypothesisByID(f, hypothesisID)
	if !found {
		_, _ = fmt.Fprintf(stderr, "conjecture: no hypothesis with id %q in %s\n", hypothesisID, f.phenomenon)
		return 1
	}

	if !evidenceledger.PromoteToLaw(h.IsUniversal, hiddenSamplesConformant, newCodeVersionsConformant) {
		_, _ = fmt.Fprintf(stdout, "%s: not promoted to Law - universally quantified: %v, conformant against hidden samples: %v, conformant across new code versions: %v\n",
			compositeID, h.IsUniversal, hiddenSamplesConformant, newCodeVersionsConformant)
		return 0
	}

	specVersion := fmt.Sprintf("%x", sha256.Sum256(f.raw))
	evidence := evidenceledger.Evidence{
		HypothesisID: compositeID,
		Stage:        evidenceledger.Law,
		ProducedBy:   promotionProducedBy,
		Survived:     "promoted to Law: universally quantified, conformant against hidden samples and across new code versions",
		ValidityHash: evidenceledger.ValidityHash(specVersion, "", version.String(), "", ""),
	}
	if _, err := evidenceledger.Append(db, evidence); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s: promoted to Law\n", compositeID)
	return 0
}
