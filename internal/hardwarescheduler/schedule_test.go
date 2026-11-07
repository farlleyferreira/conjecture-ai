package hardwarescheduler

import "testing"

func TestShouldDeferHeavyWork(t *testing.T) {
	cases := []struct {
		name                            string
		systemUnderTestLoad, userActive bool
		want                            bool
	}{
		{"neither", false, false, false},
		{"system under test load only", true, false, true},
		{"user active only", false, true, true},
		{"both", true, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ShouldDeferHeavyWork(c.systemUnderTestLoad, c.userActive)
			if got != c.want {
				t.Errorf("ShouldDeferHeavyWork(%v, %v) = %v, want %v", c.systemUnderTestLoad, c.userActive, got, c.want)
			}
		})
	}
}
