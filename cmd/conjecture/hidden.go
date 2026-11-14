package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/conformancecycle"
	"github.com/farlleyferreira/conjecture-ai/internal/contextengine"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
	"github.com/farlleyferreira/conjecture-ai/internal/version"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

// defaultHiddenSamples is how many hidden cases "conjecture check"
// draws per hypothesis when --samples is not given. Each case is one
// container run, so this is a trade between how fast a check returns
// and how fast a rule accumulates the cases promotion needs (100 at
// low weight): 20 keeps a check interactive and reaches Thesis in
// five conformant runs. It is this project's own chosen default; the
// specification names no number.
const defaultHiddenSamples = 20

// hiddenRun is what one "conjecture check" invocation knows that
// every hypothesis's hidden sampling needs.
type hiddenRun struct {
	samples        int
	specVersion    string
	codeVersion    string
	producerFamily string
}

// sampleHiddenCases draws run.samples independent random cases for h,
// runs the implementation against each, records the result as a
// structured [evidenceledger.Run] and as a readable evidence record,
// and promotes h to Thesis once its failure-free hidden cases against
// this exact specification and code meet its weight's required count.
//
// It is called only after h's deterministic check came back
// conformant: a rule that already failed has its counterexample, and
// more cases would only repeat it.
//
// It reports failed = true when a hidden case broke the rule.
func sampleHiddenCases(
	ctx context.Context,
	db *sql.DB,
	sb *sandbox.Sandbox,
	image string,
	command []string,
	h cjtr.Hypothesis,
	compositeID string,
	run hiddenRun,
	isLaw bool,
	concepts map[string]cjtr.Concept,
	tables cjtr.Tables,
	stdout io.Writer,
) (failed bool) {
	if run.samples <= 0 || h.ID == "" || !cjtr.ValidSegment(h.ID) {
		return false
	}
	hypothesisID := compositeID + "::" + h.ID

	seed, err := randomSeed()
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR choosing a seed for hidden sampling: %v\n", compositeID, h.Name, err)
		return true
	}
	cases, sampleable, reason, err := conformancecycle.HiddenSamplingInputs(concepts, run.samples, seed)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR: %v\n", compositeID, h.Name, err)
		return true
	}
	if !sampleable {
		_, _ = fmt.Fprintf(stdout, "%s: %s: hidden sampling skipped - %s. This rule cannot accumulate cases toward Thesis until it does.\n", compositeID, h.Name, reason)
		return false
	}

	dir, err := os.MkdirTemp("", "conjecture-check")
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR: %v\n", compositeID, h.Name, err)
		return true
	}
	defer func() { _ = os.RemoveAll(dir) }()
	opts := sandbox.Options{Image: image, Dir: dir, Command: command}

	verdict, counterexample, err := conformancecycle.CheckHypothesisHiddenSampling(ctx, sb, opts, h, concepts, tables, cases)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR running hidden sampling: %v\n", compositeID, h.Name, err)
		return true
	}
	label := fmt.Sprintf("hidden sampling (%d random cases, seed %d)", len(cases), seed)
	_, _ = fmt.Fprintf(stdout, "%s: %s: conformance (%s): %s\n", compositeID, h.Name, label, verdict)
	if counterexample != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s:   counterexample: %s\n", compositeID, h.Name, formatCounterexample(counterexample.Value))
	}
	if verdict == conformancecycle.Inconclusive {
		// A timeout is neither a pass nor a failure: nothing is counted
		// and nothing is reset.
		return false
	}

	conformant := verdict == conformancecycle.Conformant
	if err := evidenceledger.AppendRun(db, evidenceledger.Run{
		HypothesisID: hypothesisID, SpecVersion: run.specVersion, CodeVersion: run.codeVersion,
		Kind: evidenceledger.RunHidden, Cases: len(cases), Conformant: conformant,
		ProducerFamily: run.producerFamily, Seed: seed,
	}); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording the run: %v\n", compositeID, h.Name, err)
		return true
	}

	stage := evidenceledger.Hypothesis
	if previous, found, err := evidenceledger.Latest(db, hypothesisID); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR reading prior evidence: %v\n", compositeID, h.Name, err)
		return true
	} else if found {
		stage = previous.Stage
	}
	evidence := evidenceledger.Evidence{
		HypothesisID: hypothesisID,
		Stage:        stage,
		ProducedBy:   "conjecture check " + version.String(),
		Attacked:     label,
		ValidityHash: evidenceledger.ValidityHash(run.specVersion, run.codeVersion, version.String(), "", ""),
	}

	if !conformant {
		evidence.Stage = evidenceledger.Demote(stage)
		if counterexample != nil {
			evidence.Attacked += "; found counterexample: " + formatCounterexample(counterexample.Value)
		}
		if _, err := evidenceledger.Append(db, evidence); err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording evidence: %v\n", compositeID, h.Name, err)
		}
		if isLaw {
			_ = userdefinedlaws.ResetFailureFreeCases(db, hypothesisID)
			if count, err := userdefinedlaws.RecordCounterevidence(db, hypothesisID); err == nil {
				status := userdefinedlaws.Violated
				if userdefinedlaws.MeetsContestedThreshold(count, lawContestedThreshold) {
					status = userdefinedlaws.Contested
				}
				_ = userdefinedlaws.SetStatus(db, hypothesisID, status)
			}
		}
		_ = workqueue.Add(db, workqueue.Item{ID: hypothesisID, Tier: workqueue.Blocker, Weight: 1, Severity: 1, CreatedAt: time.Now().UTC()})
		return true
	}

	evidence.Survived = "every sampled case conformed"
	if _, err := evidenceledger.Append(db, evidence); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording evidence: %v\n", compositeID, h.Name, err)
		return true
	}

	weight, err := hypothesisWeight(db, hypothesisID, isLaw)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR reading weight: %v\n", compositeID, h.Name, err)
		return true
	}
	runs, err := evidenceledger.Runs(db, hypothesisID)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR reading runs: %v\n", compositeID, h.Name, err)
		return true
	}
	total := evidenceledger.Measure(runs, run.specVersion).FailureFree(run.codeVersion)
	_, _ = fmt.Fprintf(stdout, "%s: %s: %d/%d failure-free hidden cases against this code (%s weight)\n",
		compositeID, h.Name, total, weight.RequiredCases(), weight)

	if stage == evidenceledger.Hypothesis && weight.MeetsPromotionThreshold(total) {
		promotion := evidenceledger.Evidence{
			HypothesisID: hypothesisID,
			Stage:        evidenceledger.Thesis,
			ProducedBy:   evidence.ProducedBy,
			Survived:     fmt.Sprintf("promoted to Thesis: %d failure-free hidden cases at %s weight met the required %d", total, weight, weight.RequiredCases()),
			ValidityHash: evidence.ValidityHash,
		}
		if _, err := evidenceledger.Append(db, promotion); err != nil {
			_, _ = fmt.Fprintf(stdout, "%s: %s: ERROR recording promotion: %v\n", compositeID, h.Name, err)
			return true
		}
		_, _ = fmt.Fprintf(stdout, "%s: %s: promoted to Thesis\n", compositeID, h.Name)
	}
	return false
}

// recordDirectedRun records the deterministic check that ran just
// before hidden sampling as a [evidenceledger.Run]: it adds no cases
// toward promotion (the same boundary cases on every run are one
// observation), but a failing one has to put this code version's
// count back to zero, and that only happens if it is on record.
func recordDirectedRun(db *sql.DB, compositeID string, h cjtr.Hypothesis, run hiddenRun, conformant bool) {
	if h.ID == "" || !cjtr.ValidSegment(h.ID) {
		return
	}
	_ = evidenceledger.AppendRun(db, evidenceledger.Run{
		HypothesisID: compositeID + "::" + h.ID, SpecVersion: run.specVersion, CodeVersion: run.codeVersion,
		Kind: evidenceledger.RunDirected, Conformant: conformant, ProducerFamily: run.producerFamily,
	})
}

// hypothesisWeight is the risk weight that sets how many failure-free
// cases hypothesisID needs: a user-defined law's gated weight, the
// weight recorded for it in the context graph ("conjecture graph
// set"), or low when none was ever set.
func hypothesisWeight(db *sql.DB, hypothesisID string, isLaw bool) (evidenceledger.Weight, error) {
	if isLaw {
		return contextengine.GatedWeight(evidenceledger.Low, true, nil)
	}
	axes, found, err := contextengine.GetAxes(db, hypothesisID)
	if err != nil {
		return 0, err
	}
	if found && axes.Weight != nil {
		return *axes.Weight, nil
	}
	return evidenceledger.Low, nil
}

func randomSeed() (uint64, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b[:]), nil
}

// sourceTreeVersion hashes a project directory's files into one code
// version, so evidence is tied to the exact code it was produced
// against: change one byte and the count toward promotion starts
// again for the new version.
//
// Skipped: version-control and dependency directories, conjecture's
// own working directory, and the specifications themselves - editing
// a rule changes the spec version, and must not also look like a new
// implementation.
func sourceTreeVersion(root string) (string, error) {
	skip := map[string]bool{".git": true, ".conjecture": true, "node_modules": true, ".venv": true, "__pycache__": true}
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(root, path)
			if skip[d.Name()] || filepath.ToSlash(rel) == "conjecture" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("hashing source tree: %w", err)
	}
	sort.Strings(files)

	h := sha256.New()
	for _, path := range files {
		rel, _ := filepath.Rel(root, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("hashing source tree: %w", err)
		}
		_, _ = fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), len(data))
		_, _ = h.Write(data)
	}
	return fmt.Sprintf("src:%x", h.Sum(nil)), nil
}

// imageVersion identifies the code inside a container image by the
// image's own content id, so rebuilding an image under the same tag
// still counts as new code. When the runtime cannot be asked (or does
// not know the image yet), the image reference itself is used: weaker
// - a rebuilt tag looks unchanged - but never wrong about two
// different references.
func imageVersion(image string) string {
	out, err := exec.Command(sandbox.Runtime(), "image", "inspect", "--format", "{{.Id}}", image).Output()
	if id := strings.TrimSpace(string(out)); err == nil && id != "" {
		return "image:" + id
	}
	return "image-ref:" + image
}
