// Command fakeimplement stands in for "docker" plus the Python
// interpreter runImplementLoop always invokes with the fixed command
// ["python3", "impl.py"] - it never actually reads or runs the
// generated impl.py, since this fixture exists only to test the outer
// implement-loop's own round/retry mechanics, not conformance-cycle's
// directed sampling (already covered by internal/conformancecycle's
// own tests).
//
// It counts how many times it has been invoked (via a counter file
// "attempts" in the host directory a real "docker run -v
// HOSTDIR:/workspace" would have mounted from its own "-v" argument),
// and writes a correct or incorrect conjecture_output.json depending
// on whether that count has reached FAKEIMPLEMENT_SUCCEED_AFTER (an
// env var the test controls; 0 or unset means "never succeed").
package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "kill" {
		os.Exit(0)
	}

	var hostDir string
	for i, a := range args {
		if a == "-v" && i+1 < len(args) {
			if host, ok := strings.CutSuffix(args[i+1], ":/workspace"); ok {
				hostDir = host
			}
		}
	}

	attemptsPath := filepath.Join(hostDir, "attempts")
	attempts := 0
	if data, err := os.ReadFile(attemptsPath); err == nil {
		attempts, _ = strconv.Atoi(strings.TrimSpace(string(data)))
	}
	attempts++
	_ = os.WriteFile(attemptsPath, []byte(strconv.Itoa(attempts)), 0o644)

	succeedAfter, _ := strconv.Atoi(os.Getenv("FAKEIMPLEMENT_SUCCEED_AFTER"))

	output := `{"output.result":"99.00"}`
	if succeedAfter > 0 && attempts >= succeedAfter {
		output = `{"output.result":"7.50"}`
	}
	_ = os.WriteFile(filepath.Join(hostDir, "conjecture_output.json"), []byte(output), 0o644)
}
