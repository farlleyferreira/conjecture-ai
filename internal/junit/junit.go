// Package junit parses the JUnit XML test-result format: one of the
// "standard result formats" the language-agnostic-contracts capability
// (openspec/specs/language-agnostic-contracts/spec.md) requires conjecture
// to accept so it can read a target system's test results without parsing
// or interpreting that system's own source code.
//
// JUnit XML has no single owning specification; this follows the de facto
// shape every major test runner and CI system produces and consumes
// (Surefire/Failsafe, Jest, pytest, go-junit-report, Jenkins, and others):
// a <testsuites> element containing one or more <testsuite> elements, each
// containing <testcase> elements that may each hold a <failure>, <error>,
// or <skipped> child. A report with exactly one suite commonly omits the
// <testsuites> wrapper and uses <testsuite> as the document's root
// instead; this package accepts both shapes.
package junit

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// TestSuites is a parsed JUnit XML report.
type TestSuites struct {
	Suites []TestSuite
}

// TestSuite is one <testsuite> element.
type TestSuite struct {
	Name                             string
	Tests, Failures, Errors, Skipped int
	Time                             float64
	TestCases                        []TestCase
}

// TestCase is one <testcase> element. At most one of Failure and Error is
// non-nil; Skipped is true when the element has a <skipped> child.
type TestCase struct {
	ClassName, Name string
	Time            float64
	Failure         *Failure
	Error           *Failure
	Skipped         bool
}

// Passed reports whether the test case neither failed, errored, nor was
// skipped.
func (tc TestCase) Passed() bool {
	return tc.Failure == nil && tc.Error == nil && !tc.Skipped
}

// Failure is a <failure> or <error> element's content.
type Failure struct {
	Message, Type, Text string
}

// Parse reads a JUnit XML report from r, accepting either a <testsuites>
// or a bare <testsuite> document root.
func Parse(r io.Reader) (TestSuites, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return TestSuites{}, fmt.Errorf("junit: %w", err)
	}

	root, err := rootElementName(data)
	if err != nil {
		return TestSuites{}, err
	}

	switch root {
	case "testsuites":
		var doc xmlTestSuites
		if err := xml.Unmarshal(data, &doc); err != nil {
			return TestSuites{}, fmt.Errorf("junit: %w", err)
		}
		suites := make([]TestSuite, len(doc.Suites))
		for i, s := range doc.Suites {
			suites[i] = s.toTestSuite()
		}
		return TestSuites{Suites: suites}, nil
	case "testsuite":
		var doc xmlTestSuite
		if err := xml.Unmarshal(data, &doc); err != nil {
			return TestSuites{}, fmt.Errorf("junit: %w", err)
		}
		return TestSuites{Suites: []TestSuite{doc.toTestSuite()}}, nil
	default:
		return TestSuites{}, fmt.Errorf("junit: unrecognized root element %q, want \"testsuites\" or \"testsuite\"", root)
	}
}

// rootElementName returns the document's outermost element name.
func rootElementName(data []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("junit: %w", err)
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local, nil
		}
	}
}

type xmlTestSuites struct {
	XMLName xml.Name       `xml:"testsuites"`
	Suites  []xmlTestSuite `xml:"testsuite"`
}

type xmlTestSuite struct {
	Name      string        `xml:"name,attr"`
	Tests     int           `xml:"tests,attr"`
	Failures  int           `xml:"failures,attr"`
	Errors    int           `xml:"errors,attr"`
	Skipped   int           `xml:"skipped,attr"`
	Time      float64       `xml:"time,attr"`
	TestCases []xmlTestCase `xml:"testcase"`
}

func (s xmlTestSuite) toTestSuite() TestSuite {
	cases := make([]TestCase, len(s.TestCases))
	for i, c := range s.TestCases {
		cases[i] = c.toTestCase()
	}
	return TestSuite{
		Name: s.Name, Tests: s.Tests, Failures: s.Failures, Errors: s.Errors,
		Skipped: s.Skipped, Time: s.Time, TestCases: cases,
	}
}

type xmlTestCase struct {
	ClassName string      `xml:"classname,attr"`
	Name      string      `xml:"name,attr"`
	Time      float64     `xml:"time,attr"`
	Failure   *xmlFailure `xml:"failure"`
	Error     *xmlFailure `xml:"error"`
	Skipped   *struct{}   `xml:"skipped"`
}

func (c xmlTestCase) toTestCase() TestCase {
	tc := TestCase{ClassName: c.ClassName, Name: c.Name, Time: c.Time, Skipped: c.Skipped != nil}
	if c.Failure != nil {
		f := c.Failure.toFailure()
		tc.Failure = &f
	}
	if c.Error != nil {
		f := c.Error.toFailure()
		tc.Error = &f
	}
	return tc
}

type xmlFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Text    string `xml:",chardata"`
}

func (f xmlFailure) toFailure() Failure {
	return Failure{Message: f.Message, Type: f.Type, Text: f.Text}
}
