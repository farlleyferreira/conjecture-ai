package main

import "testing"

func TestFormatCounterexample(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{
			"inputs then outputs, sorted, prefixes dropped",
			map[string]string{"output.change_fee": "0.00", "input.hours_before_departure": "30.00", "input.fare_class": "economy-basic"},
			"fare_class=economy-basic, hours_before_departure=30.00 -> change_fee=0.00",
		},
		{"inputs only", map[string]string{"input.b": "2", "input.a": "1"}, "a=1, b=2"},
		{"outputs only", map[string]string{"output.fee": "0.00"}, "-> fee=0.00"},
		{"unprefixed names are kept whole", map[string]any{"total": 3, "input.a": "1"}, "a=1, total=3"},
		{"an unrecognized shape is printed, not dropped", 42, "42"},
	}
	for _, c := range cases {
		if got := formatCounterexample(c.value); got != c.want {
			t.Errorf("%s: formatCounterexample = %q, want %q", c.name, got, c.want)
		}
	}
}
