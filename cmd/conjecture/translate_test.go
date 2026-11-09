package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunTranslate_WrongArgCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runTranslate(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runTranslate with no arguments: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunTranslate_MissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runTranslate([]string{"this-file-does-not-exist.cjtr"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runTranslate with a missing file: exit code = %d, want 1", code)
	}
}

func TestRunTranslate_PrintsPTCommentForEveryStatement(t *testing.T) {
	path := writeCheckFixture(t)

	var stdout, stderr bytes.Buffer
	code := runTranslate([]string{path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runTranslate returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "demo-area::demo::shipping-rule: fee matches: always:") {
		t.Errorf("stdout should label the universal statement \"always\", got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "# pt:") {
		t.Errorf("stdout should contain a generated \"# pt:\" comment, got:\n%s", stdout.String())
	}
}

func TestRunTranslate_GivenAndOtherwiseBothTranslated(t *testing.T) {
	path := writeFixture(t, fixtureCjtr)

	var stdout, stderr bytes.Buffer
	code := runTranslate([]string{path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runTranslate returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "above the threshold shipping is free: given:") {
		t.Errorf("stdout should translate the \"given\" clause too, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "above the threshold shipping is free: otherwise:") {
		t.Errorf("stdout should translate the \"otherwise\" clause too, got:\n%s", stdout.String())
	}
}
