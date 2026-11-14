package hardwarescheduler

import (
	"runtime"
	"testing"
)

func TestBusyFraction_ZeroIntervalIsZero(t *testing.T) {
	s := CPUSample{Idle: 100, Total: 500}
	if got := BusyFraction(s, s); got != 0 {
		t.Errorf("BusyFraction(same sample, same sample) = %v, want 0", got)
	}
}

func TestBusyFraction_FullyIdle(t *testing.T) {
	before := CPUSample{Idle: 100, Total: 500}
	after := CPUSample{Idle: 200, Total: 600} // idleDelta == totalDelta
	if got := BusyFraction(before, after); got != 0 {
		t.Errorf("BusyFraction = %v, want 0 (every tick of the interval was idle)", got)
	}
}

func TestBusyFraction_FullyBusy(t *testing.T) {
	before := CPUSample{Idle: 100, Total: 500}
	after := CPUSample{Idle: 100, Total: 600} // idleDelta == 0
	if got := BusyFraction(before, after); got != 1 {
		t.Errorf("BusyFraction = %v, want 1 (no idle ticks in the interval)", got)
	}
}

func TestBusyFraction_HalfBusy(t *testing.T) {
	before := CPUSample{Idle: 100, Total: 500}
	after := CPUSample{Idle: 150, Total: 600} // idleDelta 50, totalDelta 100
	if got := BusyFraction(before, after); got != 0.5 {
		t.Errorf("BusyFraction = %v, want 0.5", got)
	}
}

func TestCurrentCPUSample_ReadsARealSample(t *testing.T) {
	if runtime.GOOS == "darwin" && !cgoEnabled {
		t.Skip("CPU load is not measurable on macOS without CGO")
	}
	s, err := CurrentCPUSample()
	if err != nil {
		t.Fatalf("CurrentCPUSample returned unexpected error: %v", err)
	}
	if s.Total == 0 {
		t.Error("CurrentCPUSample: Total = 0, want a nonzero cumulative CPU time")
	}
}
