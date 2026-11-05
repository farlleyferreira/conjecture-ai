package hardwarescheduler

import "testing"

func TestCheckRAM_BelowMinimum(t *testing.T) {
	err := checkRAM(4 * 1024 * 1024 * 1024) // 4 GB
	if err == nil {
		t.Fatal("checkRAM(4 GB) = nil error, want an error")
	}
}

func TestCheckRAM_AtMinimum(t *testing.T) {
	if err := checkRAM(MinimumRAMBytes); err != nil {
		t.Errorf("checkRAM(exactly 6 GB) returned unexpected error: %v", err)
	}
}

func TestCheckRAM_AboveMinimum(t *testing.T) {
	if err := checkRAM(16 * 1024 * 1024 * 1024); err != nil {
		t.Errorf("checkRAM(16 GB) returned unexpected error: %v", err)
	}
}

func TestCheckRAM_Undetermined(t *testing.T) {
	err := checkRAM(0)
	if err == nil {
		t.Fatal("checkRAM(0) = nil error, want an error (undetermined RAM is not assumed sufficient)")
	}
}

func TestCheckMinimumRAM_RunsWithoutPanicking(t *testing.T) {
	// This machine's own RAM is whatever it is; this only proves the
	// real syscall-backed path runs cleanly (never panics) and
	// produces a value checkRAM can judge, not any particular verdict.
	_ = CheckMinimumRAM()
}
