package specjudge

import "fmt"

// FalsifiabilityFinding turns one statement's falsifiability result
// (as reported by cjtr.Falsifiable, which this package does not import
// directly - a caller already has that bool once it has called
// cjtr.Falsifiable itself) into a [Finding] for [Judge]: a Rejecting
// finding when falsifiable is false, matching the falsifiability
// requirement's own "SHALL reject" wording, or nil when there is
// nothing to report.
func FalsifiabilityFinding(hypothesisName, statementText string, falsifiable bool) *Finding {
	if falsifiable {
		return nil
	}
	return &Finding{
		Severity: Rejecting,
		Message:  fmt.Sprintf("%s: %q holds for every value its term's own declared domain allows (unfalsifiable)", hypothesisName, statementText),
	}
}
