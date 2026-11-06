package cjtr

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
	"gopkg.in/yaml.v3"
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
	withoutSigned, err := excludeSignedBlock(raw, root)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(withoutSigned)
	return sum[:], nil
}

// signedBlockLines locates the top-level "signed" pair's own line and
// the line the next top-level pair starts at (0 if "signed" is the
// last pair, meaning "to EOF"), or (0, 0) if root has no "signed" pair
// at all.
func signedBlockLines(root *yamlstrict.Node) (signedLine, nextLine int) {
	for i, p := range root.Pairs {
		if p.Key == "signed" {
			if i+1 < len(root.Pairs) {
				return p.Line, root.Pairs[i+1].Line
			}
			return p.Line, 0
		}
	}
	return 0, 0
}

// StripSignedBlock returns raw with its top-level "signed" block
// removed, if it has one (raw unchanged otherwise). Unlike [Sign], it
// does not require raw be otherwise signable (no pending-marker check)
// - it exists for a caller that must guarantee no signed block
// survives regardless of why one might be present, such as llm-
// orchestration's trust boundary stripping a "signed" block an LLM's
// draft output included despite being told not to ("no LLM... SHALL
// be able to... sign a .cjtr file on behalf of a different user").
//
// If raw does not parse as a strict-profile YAML document at all,
// StripSignedBlock returns raw unchanged rather than erroring: a
// caller that also validates the file (as conjecture verify always
// does) will surface that parse failure on its own, with a clearer
// message than this function could give out of context.
func StripSignedBlock(raw []byte) ([]byte, error) {
	root, err := yamlstrict.Load(bytes.NewReader(raw))
	if err != nil {
		return raw, nil
	}
	return excludeSignedBlock(raw, root)
}

// excludeSignedBlock returns raw with its top-level "signed" block's
// own lines removed (raw unchanged if it has none).
func excludeSignedBlock(raw []byte, root *yamlstrict.Node) ([]byte, error) {
	if root.Kind != yamlstrict.MappingKind {
		return nil, fmt.Errorf("cjtr: document root must be a mapping")
	}

	signedLine, nextLine := signedBlockLines(root)
	if signedLine == 0 {
		return raw, nil
	}

	var kept bytes.Buffer
	for i, line := range splitLinesKeepingTerminators(raw) {
		lineNum := i + 1
		if lineNum >= signedLine && (nextLine == 0 || lineNum < nextLine) {
			continue // inside the "signed" block - excluded
		}
		kept.Write(line)
	}
	return kept.Bytes(), nil
}

// splitLinesKeepingTerminators splits raw into lines, each line's own
// "\n" kept at its end (so re-concatenating a subset of the returned
// lines reproduces exactly that subset's original bytes).
func splitLinesKeepingTerminators(raw []byte) [][]byte {
	return bytes.SplitAfter(raw, []byte("\n"))
}

// PendingMarker is one occurrence of an unresolved "???" marker
// (spec-authoring spec, "Pending markers block signature": "The system
// SHALL represent an open question inside a .cjtr file as a ???
// marker").
type PendingMarker struct {
	Line int
	Text string
}

// FindPendingMarkers scans raw for every "???" marker occurrence, by
// line. This scans the file's raw text rather than its parsed
// structure, and does not try to tell a reserved marker apart from a
// literal "???" that happens to appear inside, say, a human-written
// description: "???" is a token the closed .cjtr vocabulary reserves
// for exactly this purpose, the same way any other word outside that
// vocabulary is rejected elsewhere in this package rather than given
// the benefit of the doubt, so this does not add a special case for it
// appearing to mean something else.
func FindPendingMarkers(raw []byte) []PendingMarker {
	var markers []PendingMarker
	for i, line := range bytes.Split(raw, []byte("\n")) {
		if bytes.Contains(line, []byte("???")) {
			markers = append(markers, PendingMarker{Line: i + 1, Text: string(bytes.TrimSpace(line))})
		}
	}
	return markers
}

// CheckSignable reports an error naming every unresolved "???" marker
// in raw, or nil if there is none - "A file containing any unresolved
// ??? marker SHALL NOT be eligible for signature," checked ahead of
// time rather than only once a signing attempt is already underway.
func CheckSignable(raw []byte) error {
	markers := FindPendingMarkers(raw)
	if len(markers) == 0 {
		return nil
	}

	lines := make([]string, len(markers))
	for i, m := range markers {
		lines[i] = fmt.Sprintf("line %d: %s", m.Line, m.Text)
	}
	return fmt.Errorf("cjtr: %d unresolved \"???\" marker(s) block signing:\n%s", len(markers), strings.Join(lines, "\n"))
}

// Sign returns raw with its "signed" block (if any) replaced by a new
// one recording user, tool, date, and confirmed, and whose hash is
// raw's current [ContentHash] (computed before any block is removed or
// added, so re-signing hashes only what was already there, exactly as
// a first signing would). The new block is always written at the
// file's end, whether or not an old one existed elsewhere in it: the
// specification gives no requirement that a re-signed file keep its
// "signed" block in the same place.
//
// Sign refuses - returning an error, with raw itself unchanged - when
// [CheckSignable] finds an unresolved "???" marker.
func Sign(raw []byte, root *yamlstrict.Node, user, tool, date string, confirmed bool) ([]byte, error) {
	if err := CheckSignable(raw); err != nil {
		return nil, err
	}

	hash, err := ContentHash(raw, root)
	if err != nil {
		return nil, err
	}

	withoutSigned, err := excludeSignedBlock(raw, root)
	if err != nil {
		return nil, err
	}

	block, err := yaml.Marshal(signedYAML{
		User: user, Tool: tool, Date: date,
		Hash: fmt.Sprintf("%x", hash), Confirmed: confirmed,
	})
	if err != nil {
		return nil, fmt.Errorf("cjtr: encoding \"signed\" block: %w", err)
	}

	var out bytes.Buffer
	out.Write(withoutSigned)
	if out.Len() > 0 && out.Bytes()[out.Len()-1] != '\n' {
		out.WriteByte('\n')
	}
	out.WriteString("signed:\n")
	for _, line := range bytes.Split(bytes.TrimRight(block, "\n"), []byte("\n")) {
		out.WriteString("  ")
		out.Write(line)
		out.WriteByte('\n')
	}

	return out.Bytes(), nil
}

// signedYAML is [Signed]'s on-disk shape, marshaled with gopkg.in/
// yaml.v3 (already this project's one YAML-writing dependency) rather
// than hand-built strings, so a user- or tool-supplied value that
// happens to need YAML quoting or escaping (a colon, a quote, …) is
// still written out as valid YAML.
type signedYAML struct {
	User      string `yaml:"user"`
	Tool      string `yaml:"tool"`
	Date      string `yaml:"date"`
	Hash      string `yaml:"hash"`
	Confirmed bool   `yaml:"confirmed"`
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
