package specjudge

import (
	"fmt"
	"strings"
)

// ConsistencyFinding turns an SMT satisfiability check's own unsat
// result into a Finding for Judge - spec-judge's "Consistency check
// via solver": "The judge SHALL check statements within the provable
// subset for logical contradictions using an SMT solver, and SHALL
// report any contradiction found between two or more statements."
//
// Like [FalsifiabilityFinding], this does not import or run the
// solver itself (internal/expressioncore, which this package does not
// depend on): a caller already has unsat (whether the statements
// named by statementNames can all hold at once) once it has built and
// checked the query itself.
func ConsistencyFinding(statementNames []string, unsat bool) *Finding {
	if !unsat {
		return nil
	}
	return &Finding{
		Severity: Rejecting,
		Message:  fmt.Sprintf("contradiction: %s cannot all hold at once", strings.Join(statementNames, ", ")),
	}
}
