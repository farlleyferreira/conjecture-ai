// Command fakedocker stands in for the real "docker" binary in
// sandbox_test.go, so Run's own argument-building, output-capture,
// exit-code, and timeout/kill logic can be exercised without a real
// Docker daemon. It is not built as part of the module's normal build
// (testdata is ignored by the go tool); sandbox_test.go compiles it
// once, on demand, with "go build".
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "kill" {
		if marker := os.Getenv("FAKEDOCKER_KILL_MARKER"); marker != "" && len(args) > 1 {
			_ = os.WriteFile(marker, []byte(args[1]), 0o644)
		}
		os.Exit(0)
	}

	// Simulate "docker run ... IMAGE CMD...": the command portion is
	// whatever comes after the image name, which is whatever argument
	// immediately follows "-w" "/workspace" in the real builder's own
	// argument order.
	joined := strings.Join(args, " ")
	switch {
	case strings.Contains(joined, "SLEEP_FOREVER"):
		time.Sleep(10 * time.Second)
		os.Exit(0)
	case strings.Contains(joined, "FAIL"):
		fmt.Fprintln(os.Stderr, "boom")
		os.Exit(7)
	default:
		fmt.Fprintln(os.Stdout, "hello from sandbox")
		os.Exit(0)
	}
}
