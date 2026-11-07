package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// loadedFicha is everything loading and parsing a .cjtr file at a real
// path produces, shared by every command that needs more than just its
// raw bytes (today: runVerify and runCheck) - factored out so the exact
// same parsing sequence (and its exact same path-derived area/
// phenomenon/table-root conventions) is never maintained as two
// independent copies.
type loadedFicha struct {
	raw    []byte
	root   *yamlstrict.Node
	signed *cjtr.Signed

	area, phenomenon string
	phenomenonID     string

	concepts map[string]cjtr.Concept
	tables   cjtr.Tables
	rules    []cjtr.Rule
	// fileLaws are "user_defined_laws" declared at the file's top level
	// (sibling to "rules"), scoped to phenomenonID - see
	// [cjtr.UserDefinedLaws]'s own doc comment on that placement.
	fileLaws []cjtr.Hypothesis
}

// loadFicha reads, parses, and validates the .cjtr file at path,
// following the cjtr-language spec's own path convention (area is its
// directory, phenomenon is its filename) and the project's own
// invented table-path convention (a table concept's path is relative
// to the conjecture tree's root, three directories above
// phenomena/<area>/<phenomenon>.cjtr - see the original comment this
// was factored out of for why).
func loadFicha(path string) (loadedFicha, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return loadedFicha{}, err
	}

	root, err := yamlstrict.Load(bytes.NewReader(raw))
	if err != nil {
		return loadedFicha{}, err
	}

	version, err := cjtr.FormatVersion(root)
	if err != nil {
		return loadedFicha{}, err
	}
	if err := cjtr.CheckSupported(version); err != nil {
		return loadedFicha{}, err
	}

	signed, err := cjtr.ReadSigned(root)
	if err != nil {
		return loadedFicha{}, err
	}

	area := filepath.Base(filepath.Dir(path))
	phenomenon := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	conjectureRoot := filepath.Dir(filepath.Dir(filepath.Dir(path)))

	concepts, err := cjtr.Concepts(root)
	if err != nil {
		return loadedFicha{}, err
	}

	tables, err := cjtr.LoadTables(concepts, conjectureRoot)
	if err != nil {
		return loadedFicha{}, err
	}

	observation, err := cjtr.ReadObservation(root)
	if err != nil {
		return loadedFicha{}, err
	}

	// "scope" and "simplifications" are declarative documentation with
	// no effect on verification yet, but are still read and validated
	// here so a malformed one is caught rather than silently ignored.
	if _, err := cjtr.ReadScope(root); err != nil {
		return loadedFicha{}, err
	}
	if _, err := cjtr.ReadSimplifications(root); err != nil {
		return loadedFicha{}, err
	}
	if _, err := cjtr.Axioms(root, concepts); err != nil {
		return loadedFicha{}, err
	}

	rules, err := cjtr.Rules(root, area, phenomenon, concepts, observation)
	if err != nil {
		return loadedFicha{}, err
	}

	phenomenonID, err := cjtr.ComposeID(area, phenomenon)
	if err != nil {
		return loadedFicha{}, err
	}
	fileLaws, err := cjtr.UserDefinedLaws(root, concepts)
	if err != nil {
		return loadedFicha{}, err
	}

	return loadedFicha{
		raw: raw, root: root, signed: signed,
		area: area, phenomenon: phenomenon, phenomenonID: phenomenonID,
		concepts: concepts, tables: tables, rules: rules, fileLaws: fileLaws,
	}, nil
}
