package main

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/decisionledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

func openSignTestStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestRecordCandidateReviewDecision_Records(t *testing.T) {
	db := openSignTestStore(t)
	if err := recordCandidateReviewDecision(db, "phenomena/shipping/free-shipping.cjtr", "alice"); err != nil {
		t.Fatalf("recordCandidateReviewDecision returned unexpected error: %v", err)
	}

	decisions, err := decisionledger.List(db)
	if err != nil {
		t.Fatalf("decisionledger.List returned unexpected error: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("decisionledger.List returned %d decisions, want 1", len(decisions))
	}
	got := decisions[0]
	if got.Kind != "candidate-review" || got.Subject != "phenomena/shipping/free-shipping.cjtr" || got.Determination != "accepted" || got.DecidedBy != "alice" {
		t.Errorf("recorded decision = %+v, want kind candidate-review, determination accepted, decided by alice", got)
	}
}

func TestRunSign_SignsFileInPlace(t *testing.T) {
	path := writeFixture(t, fixtureCjtr)
	// --confirmed also records a decisionledger entry - swap in a
	// temporary store via the test hook rather than letting it resolve
	// the real per-OS, per-user data directory (e.g. under
	// "$HOME/.local/share/conjecture" on Linux), which this test
	// should not pollute with its own run - see [openDecisionStore]'s
	// own doc comment.
	// runSign closes whatever openDecisionStore returns, so this opens
	// a fresh connection to the same on-disk store each call, rather
	// than handing back one shared *sql.DB a later Close would break.
	storeDir := filepath.Join(t.TempDir(), "data")
	previous := openDecisionStore
	openDecisionStore = func() (*sql.DB, error) { return persistence.OpenStoreAt(storeDir) }
	t.Cleanup(func() { openDecisionStore = previous })

	var stdout, stderr bytes.Buffer
	code := runSign([]string{"--user", "alice", "--confirmed", path}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("runSign exit code = %d, want 0; stdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "alice") {
		t.Errorf("stdout should mention the signing user, got: %s", stdout.String())
	}

	verifyDB, err := persistence.OpenStoreAt(storeDir)
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	defer func() { _ = verifyDB.Close() }()
	decisions, err := decisionledger.List(verifyDB)
	if err != nil {
		t.Fatalf("decisionledger.List returned unexpected error: %v", err)
	}
	if len(decisions) != 1 || decisions[0].Kind != "candidate-review" {
		t.Errorf("decisionledger.List = %+v, want one candidate-review decision", decisions)
	}

	signedContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the signed file: %v", err)
	}
	if !strings.Contains(string(signedContent), "signed:") {
		t.Errorf("signed file should contain a \"signed:\" block, got:\n%s", signedContent)
	}
	if !strings.Contains(string(signedContent), "user: alice") {
		t.Errorf("signed file's \"signed\" block should record the user, got:\n%s", signedContent)
	}

	// A signed, confirmed file's verify run should carry no "unconfirmed"
	// notice.
	var verifyStdout, verifyStderr bytes.Buffer
	verifyCode := runVerify([]string{path}, &verifyStdout, &verifyStderr)
	if verifyCode != 0 {
		t.Errorf("runVerify on the signed file exit code = %d, want 0", verifyCode)
	}
	if strings.Contains(verifyStdout.String(), "NOTICE") {
		t.Errorf("runVerify on a signed, confirmed file should not carry the unconfirmed notice, got:\n%s", verifyStdout.String())
	}
}

func TestRunSign_RefusesWithPendingMarker(t *testing.T) {
	broken := strings.Replace(fixtureCjtr, "then: the shipping fee is 0.00", "then: ???", 1)
	path := writeFixture(t, broken)

	var stdout, stderr bytes.Buffer
	code := runSign([]string{"--user", "alice", path}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runSign exit code = %d, want 1 (an unresolved \"???\" marker should block signing)", code)
	}
	if stderr.String() == "" {
		t.Error("runSign with a pending marker should write an error to stderr")
	}

	// The file itself must be left untouched.
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the file: %v", err)
	}
	if strings.Contains(string(content), "signed:") {
		t.Error("runSign should not have written a \"signed\" block when it refused to sign")
	}
}

func TestRunSign_MissingPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSign([]string{"--user", "alice"}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runSign with no file path exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunSign_MissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSign([]string{"--user", "alice", filepath.Join(t.TempDir(), "nope.cjtr")}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runSign with a missing file exit code = %d, want 1", code)
	}
}
