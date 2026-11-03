package junit

import (
	"strings"
	"testing"
)

const wrappedReport = `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="shipping" tests="3" failures="1" errors="1" skipped="1" time="0.045">
    <testcase classname="shipping.FreeAbove" name="passes" time="0.010"/>
    <testcase classname="shipping.FreeAbove" name="fails" time="0.012">
      <failure message="expected 0.00, got 7.50" type="AssertionError">at line 42</failure>
    </testcase>
    <testcase classname="shipping.FreeAbove" name="errors" time="0.005">
      <error message="nil pointer" type="RuntimeError">stack trace</error>
    </testcase>
    <testcase classname="shipping.FreeAbove" name="skipped" time="0.000">
      <skipped/>
    </testcase>
  </testsuite>
</testsuites>`

func TestParse_WrappedTestSuites(t *testing.T) {
	got, err := Parse(strings.NewReader(wrappedReport))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if len(got.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(got.Suites))
	}

	suite := got.Suites[0]
	if suite.Name != "shipping" || suite.Tests != 3 || suite.Failures != 1 || suite.Errors != 1 || suite.Skipped != 1 {
		t.Errorf("suite = %+v", suite)
	}
	if len(suite.TestCases) != 4 {
		t.Fatalf("len(TestCases) = %d, want 4", len(suite.TestCases))
	}

	pass, fail, errCase, skip := suite.TestCases[0], suite.TestCases[1], suite.TestCases[2], suite.TestCases[3]

	if !pass.Passed() {
		t.Error("pass.Passed() = false, want true")
	}

	if fail.Passed() {
		t.Error("fail.Passed() = true, want false")
	}
	if fail.Failure == nil || fail.Failure.Message != "expected 0.00, got 7.50" || fail.Failure.Type != "AssertionError" {
		t.Errorf("fail.Failure = %+v", fail.Failure)
	}
	if strings.TrimSpace(fail.Failure.Text) != "at line 42" {
		t.Errorf("fail.Failure.Text = %q", fail.Failure.Text)
	}

	if errCase.Passed() {
		t.Error("errCase.Passed() = true, want false")
	}
	if errCase.Error == nil || errCase.Error.Message != "nil pointer" {
		t.Errorf("errCase.Error = %+v", errCase.Error)
	}

	if skip.Passed() {
		t.Error("skip.Passed() = true, want false")
	}
	if !skip.Skipped {
		t.Error("skip.Skipped = false, want true")
	}
}

const bareTestSuite = `<?xml version="1.0"?>
<testsuite name="shipping" tests="1">
  <testcase classname="shipping.FreeAbove" name="passes" time="0.010"/>
</testsuite>`

func TestParse_BareTestSuiteRoot(t *testing.T) {
	got, err := Parse(strings.NewReader(bareTestSuite))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if len(got.Suites) != 1 || got.Suites[0].Name != "shipping" {
		t.Errorf("got = %+v", got)
	}
	if len(got.Suites[0].TestCases) != 1 || !got.Suites[0].TestCases[0].Passed() {
		t.Errorf("Suites[0].TestCases = %+v", got.Suites[0].TestCases)
	}
}

func TestParse_MalformedXML(t *testing.T) {
	_, err := Parse(strings.NewReader("<testsuite><not-closed>"))
	if err == nil {
		t.Fatal("Parse with malformed XML = nil error, want an error")
	}
}

func TestParse_UnrecognizedRoot(t *testing.T) {
	_, err := Parse(strings.NewReader("<somethingelse/>"))
	if err == nil {
		t.Fatal("Parse with an unrecognized root element = nil error, want an error")
	}
}

func TestParse_Empty(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	if err == nil {
		t.Fatal("Parse with empty input = nil error, want an error")
	}
}
