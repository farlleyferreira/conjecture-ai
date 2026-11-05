package cjtr

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// Signed is a .cjtr file's top-level "signed" block (spec-authoring
// spec, "Signature records authorship without a dedicated key"; the
// specification's field-reference table lists its fields: "usuário,
// ferramenta usada, data, hash do conteúdo e se houve revisão humana
// (confirmed)").
type Signed struct {
	User string
	Tool string
	Date string
	// Hash is the recorded content hash, hex-encoded exactly as
	// written - see [ContentHash].
	Hash string
	// Confirmed reports whether a human reviewed the file before
	// signing ("Confirmed flag gates downstream use": an unconfirmed
	// file's verdicts must carry a visible caveat).
	Confirmed bool
}

// ReadSigned reads a strictly-loaded .cjtr document's top-level
// "signed" block, if present ("" -> nil, nil: an unsigned file is not
// an error here, only somewhere a signature is specifically required).
func ReadSigned(root *yamlstrict.Node) (*Signed, error) {
	node, ok := root.Get("signed")
	if !ok {
		return nil, nil
	}
	if node.Kind != yamlstrict.MappingKind {
		return nil, fmt.Errorf("cjtr: \"signed\" must be a mapping, got %v", node.Kind)
	}

	s := &Signed{}
	if v, ok := scalarField(node, "signed", "user"); ok {
		s.User = v
	}
	if v, ok := scalarField(node, "signed", "tool"); ok {
		s.Tool = v
	}
	if v, ok := scalarField(node, "signed", "date"); ok {
		s.Date = v
	}
	if v, ok := scalarField(node, "signed", "hash"); ok {
		s.Hash = v
	}
	if confirmedNode, ok := node.Get("confirmed"); ok {
		if confirmedNode.Kind != yamlstrict.ScalarKind || (confirmedNode.Value != "true" && confirmedNode.Value != "false") {
			return nil, fmt.Errorf("cjtr: \"signed\": \"confirmed\" must be the scalar true or false")
		}
		s.Confirmed = confirmedNode.Value == "true"
	}

	return s, nil
}

// ContentHash computes a .cjtr file's signature content hash: the
// SHA-256 of the file's raw bytes, excluding its top-level "signed"
// block's own lines ("a content hash of the file excluding the signed
// block itself"). A file with no "signed" block yet hashes in full -
// so that signing it later, by appending a "signed" block afterward,
// reproduces the same hash for everything that came before it.
//
// This operates on raw, as-written bytes rather than the parsed
// structure, because the specification's own invalidation scenario -
// "any part of a .cjtr file outside the signed block changes" - means a
// change this package's own loader would not care about, such as a
// reformatted comment or changed whitespace, must still invalidate the
// signature.
func ContentHash(raw []byte, root *yamlstrict.Node) ([]byte, error) {
	if root.Kind != yamlstrict.MappingKind {
		return nil, fmt.Errorf("cjtr: document root must be a mapping")
	}

	signedLine, nextLine := 0, 0
	for i, p := range root.Pairs {
		if p.Key == "signed" {
			signedLine = p.Line
			if i+1 < len(root.Pairs) {
				nextLine = root.Pairs[i+1].Line
			}
			break
		}
	}

	if signedLine == 0 {
		sum := sha256.Sum256(raw)
		return sum[:], nil
	}

	var kept bytes.Buffer
	for i, line := range splitLinesKeepingTerminators(raw) {
		lineNum := i + 1
		if lineNum >= signedLine && (nextLine == 0 || lineNum < nextLine) {
			continue // inside the "signed" block - excluded from the hash
		}
		kept.Write(line)
	}

	sum := sha256.Sum256(kept.Bytes())
	return sum[:], nil
}

// splitLinesKeepingTerminators splits raw into lines, each line's own
// "\n" kept at its end (so re-concatenating a subset of the returned
// lines reproduces exactly that subset's original bytes).
func splitLinesKeepingTerminators(raw []byte) [][]byte {
	return bytes.SplitAfter(raw, []byte("\n"))
}

// VerifySignature reports whether root's recorded "signed" block's hash
// still matches raw's current content hash (see [ContentHash]) - "any
// part of a .cjtr file outside the signed block changes" -> "the
// recorded signature no longer matches the file's content hash." It
// returns an error if the file has no "signed" block at all, rather
// than treating an unsigned file as either valid or invalid.
func VerifySignature(raw []byte, root *yamlstrict.Node) (bool, error) {
	signed, err := ReadSigned(root)
	if err != nil {
		return false, err
	}
	if signed == nil {
		return false, fmt.Errorf("cjtr: file has no \"signed\" block")
	}

	current, err := ContentHash(raw, root)
	if err != nil {
		return false, err
	}

	return fmt.Sprintf("%x", current) == signed.Hash, nil
}
