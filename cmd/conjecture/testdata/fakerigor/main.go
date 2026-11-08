// Command fakerigor stands in for "docker" in
// rigorlevels_integration_test.go, computing a real shipping-fee
// output from whatever input bindings conjecture actually sent it -
// unlike internal/conformancecycle's own fakeimpl (testdata/fakeimpl),
// which always writes one canned output regardless of input. That
// difference is the point: a bounded/complete-proof test only proves
// something real if each of its many generated cases gets its own
// distinct, correctly computed output, not the same fixed string
// every time.
package main

import (
	"encoding/json"
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
			// Split on the known ":/workspace" suffix, not the first
			// colon: a Windows host path already has one right after
			// its drive letter.
			if host, ok := strings.CutSuffix(args[i+1], ":/workspace"); ok {
				hostDir = host
			}
		}
	}

	inputBytes, err := os.ReadFile(filepath.Join(hostDir, "conjecture_input.json"))
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
	var input map[string]string
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	subtotal, err := strconv.ParseFloat(input["input.discounted_subtotal"], 64)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	// The real rule this fixture mirrors: free shipping above a
	// 100.00 threshold, except for a subscription order.
	fee := "7.50"
	if subtotal >= 100.00 && input["input.order_type"] != "subscription" {
		fee = "0.00"
	}

	output, err := json.Marshal(map[string]string{"output.shipping_fee": fee})
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(hostDir, "conjecture_output.json"), output, 0o644); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}
