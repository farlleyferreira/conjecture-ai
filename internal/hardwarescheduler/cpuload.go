// cpuload.go measures real CPU busyness - the "machine is busy"
// signal schedule.go's own ShouldDeferHeavyWork needs a caller to
// supply, which nothing in this codebase measured until now (see
// this package's own doc comment).
//
// Honesty note: this is overall system CPU utilization, not literal
// keyboard/mouse input-idle detection ("the user is actively using
// the machine," the specification's own wording for userActive).
// True input-idle detection needs an OS-specific API per platform
// (Windows' GetLastInputInfo, macOS's CoreGraphics event-source
// query, X11's XScreenSaverQueryInfo with no equivalent at all under
// Wayland) - each either CGO or deeply platform-fragmented, unlike
// every other OS-level reading this codebase does (RAM via
// github.com/pbnjay/memory, now CPU via github.com/mackerelio/
// go-osstat, both pure Go, no CGO, covering Linux/macOS/Windows
// identically). Overall CPU load is this package's own chosen proxy
// instead: a genuinely real, cross-platform-uniform signal for "the
// machine is doing significant work right now" - close enough to
// what deferring heavy work is actually protecting against, without
// guessing at a platform-specific mechanism this package cannot
// build consistently across all three supported OSes.
package hardwarescheduler

import (
	"github.com/mackerelio/go-osstat/cpu"
)

// CPUBusyThreshold is this package's own disclosed default for how
// busy counts as "busy" for [ShouldDeferHeavyWork]'s own userActive
// signal - the specification names no concrete threshold, so this is
// this project's own chosen value (the same "disclosed default, not
// a value the specification gives" posture every other fixed
// constant in this codebase already takes).
const CPUBusyThreshold = 0.70

// CPUSample is one snapshot of cumulative CPU time, enough to compute
// busyness between two samples via [BusyFraction]. It mirrors only
// the two fields that computation needs out of go-osstat's own
// cpu.Stats, so this package's own public API does not leak that
// dependency's shape into every caller - a different CPU-stats
// source could replace [CurrentCPUSample]'s own implementation
// without changing anything downstream.
type CPUSample struct {
	Idle, Total uint64
}

// CurrentCPUSample reads the machine's current cumulative CPU time.
// Two samples taken some real time apart are needed to compute
// busyness ([BusyFraction]) - a single sample alone, being a
// cumulative counter since boot, says nothing about current load.
func CurrentCPUSample() (CPUSample, error) {
	s, err := cpu.Get()
	if err != nil {
		return CPUSample{}, err
	}
	return CPUSample{Idle: s.Idle, Total: s.Total}, nil
}

// BusyFraction reports the fraction of CPU time spent busy (not
// idle) between before and after - two samples taken some real time
// apart, earliest first. 0.0 means entirely idle across that
// interval, 1.0 means entirely busy. A zero-length interval (no
// total-time delta at all, e.g. before and after are the same
// sample) reports 0.0 rather than dividing by zero - nothing actually
// observed over no elapsed time is not evidence of busyness.
func BusyFraction(before, after CPUSample) float64 {
	totalDelta := after.Total - before.Total
	if totalDelta == 0 {
		return 0
	}
	idleDelta := after.Idle - before.Idle
	return 1 - float64(idleDelta)/float64(totalDelta)
}
