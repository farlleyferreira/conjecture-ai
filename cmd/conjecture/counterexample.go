package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// formatCounterexample renders a counterexample's bindings for a
// person to read: the inputs that were fed in, then the output the
// implementation actually produced, each as name=value with the
// "input."/"output." prefix dropped and names sorted so the same
// counterexample always prints the same way -
//
//	fare_class=economy-basic, hours_before_departure=30.00 -> change_fee=0.00
//
// instead of Go's own map syntax ("map[input.fare_class:...]"), which
// is an implementation detail of this program, not something its
// reader should have to parse. Any value that is not a flat map of
// bindings is printed as Go prints it, unchanged - a shape this
// function does not recognize is still shown, never dropped.
func formatCounterexample(value any) string {
	// Reflection, not a type switch: the value conformance-cycle hands
	// back is a named map type (cjtr.Bindings), which a switch on
	// map[string]string does not match.
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return fmt.Sprint(value)
	}
	bindings := make(map[string]string, rv.Len())
	for iter := rv.MapRange(); iter.Next(); {
		bindings[iter.Key().String()] = fmt.Sprint(iter.Value().Interface())
	}

	var inputs, outputs, other []string
	for k, val := range bindings {
		switch {
		case strings.HasPrefix(k, "input."):
			inputs = append(inputs, strings.TrimPrefix(k, "input.")+"="+val)
		case strings.HasPrefix(k, "output."):
			outputs = append(outputs, strings.TrimPrefix(k, "output.")+"="+val)
		default:
			other = append(other, k+"="+val)
		}
	}
	sort.Strings(inputs)
	sort.Strings(outputs)
	sort.Strings(other)
	inputs = append(inputs, other...)

	switch {
	case len(outputs) == 0:
		return strings.Join(inputs, ", ")
	case len(inputs) == 0:
		return "-> " + strings.Join(outputs, ", ")
	default:
		return strings.Join(inputs, ", ") + " -> " + strings.Join(outputs, ", ")
	}
}

// indentOutput joins a crashed implementation's stdout and stderr,
// trimmed and indented, for printing under the line that reports the
// crash. Output with nothing in it says so rather than printing a
// blank block.
func indentOutput(stdout, stderr []byte) string {
	text := strings.TrimSpace(strings.TrimSpace(string(stdout)) + "\n" + strings.TrimSpace(string(stderr)))
	if text == "" {
		return "    (no output)"
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "    " + strings.TrimRight(line, "\r")
	}
	return strings.Join(lines, "\n")
}
