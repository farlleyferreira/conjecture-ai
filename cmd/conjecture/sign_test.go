package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSign_SignsFileInPlace(t *testing.T) {
	path := writeFixture(t, fixtureCjtr)

	var stdout, stderr bytes.Buffer
	code := runSign([]string{"--user", "alice", "--confirmed", path}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("runSign exit code = %d, want 0; stdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "alice") {
		t.Errorf("stdout should mention the signing user, got: %s", stdout.String())
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
