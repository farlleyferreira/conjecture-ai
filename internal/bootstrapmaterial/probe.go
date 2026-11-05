// Package bootstrapmaterial implements pieces of the bootstrap-
// material capability (openspec/specs/bootstrap-material/spec.md): how
// conjecture bridges to an initial observation harness before a
// stable, first-party harness exists for a given codebase. This first
// slice only implements "Known-answer probes detect a wrong harness" -
// reference-document ingestion and harness promotion both need pieces
// (knowledge-bases' own ingestion flow, and a human-confirmation step)
// this package does not attempt to orchestrate on its own.
package bootstrapmaterial

import (
	"bytes"
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/jsonschema"
)

// Probe is a known-answer probe run against a harness (initial or
// stable). ExpectedAnswer and ExpectedSchema are each checked only
// when non-empty/non-nil: a probe may declare either, both, or - when
// run more than once - rely on reproducibility alone.
type Probe struct {
	Input          string
	Seed           string
	ExpectedAnswer string
	// ExpectedSchema is a JSON Schema document (raw bytes) the probe's
	// output must validate against, via [jsonschema.Validate].
	ExpectedSchema []byte
}

// Result is one observed outcome of running a Probe against a harness.
type Result struct {
	Output []byte
}

// CheckProbe reports whether the harness should be flagged wrong,
// given probe and one or more observed Results from running it -
// "The system SHALL run known-answer probes against any harness... and
// SHALL flag the harness as wrong when a probe's result does not match
// its known answer, when output does not conform to the expected
// schema, or when the same input and seed produce different results
// across runs." Passing more than one Result is how a caller checks
// that last condition: CheckProbe does not run the harness itself, so
// it has no way to produce repeated results on its own.
//
// wrong reports whether any of the three conditions held; reasons
// names every one that did (a harness can fail more than one check at
// once, and the specification's own wording treats each as its own
// reason to distrust the harness, not just the first one found).
func CheckProbe(probe Probe, results []Result) (wrong bool, reasons []string) {
	for i, r := range results {
		if probe.ExpectedAnswer != "" && string(r.Output) != probe.ExpectedAnswer {
			reasons = append(reasons, fmt.Sprintf("result %d: output %q does not match the known answer %q", i, r.Output, probe.ExpectedAnswer))
		}
		if probe.ExpectedSchema != nil {
			if err := jsonschema.Validate(probe.ExpectedSchema, r.Output); err != nil {
				reasons = append(reasons, fmt.Sprintf("result %d: output does not conform to the expected schema: %v", i, err))
			}
		}
	}

	if len(results) > 1 {
		first := results[0].Output
		for i := 1; i < len(results); i++ {
			if !bytes.Equal(results[i].Output, first) {
				reasons = append(reasons, fmt.Sprintf("result %d differs from result 0 for the same input and seed (non-reproducible)", i))
				break // one mismatch already proves non-reproducibility
			}
		}
	}

	return len(reasons) > 0, reasons
}
