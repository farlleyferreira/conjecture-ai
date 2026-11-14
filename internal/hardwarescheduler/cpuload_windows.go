package hardwarescheduler

import (
	"fmt"
	"syscall"
	"unsafe"
)

var procGetSystemTimes = syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemTimes")

// CurrentCPUSample reads the machine's current cumulative CPU time on
// Windows through kernel32's own GetSystemTimes - go-osstat, which
// cpuload_read.go uses everywhere else, has no Windows reading at all
// (its own cpu.Get returns "not implemented" there), so before this
// file existed "conjecture watch" could never tell a busy Windows
// machine from an idle one and simply never deferred.
//
// GetSystemTimes reports three cumulative counters, each a FILETIME
// (100-nanosecond units since boot, summed across every processor):
// idle time, kernel time and user time. Kernel time already includes
// idle time, so total = kernel + user, with no separate idle term to
// add. Two samples some real time apart are needed to compute
// busyness ([BusyFraction]), exactly as on every other OS.
func CurrentCPUSample() (CPUSample, error) {
	var idle, kernel, user syscall.Filetime
	r, _, err := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r == 0 {
		return CPUSample{}, fmt.Errorf("hardwarescheduler: GetSystemTimes: %w", err)
	}
	return CPUSample{
		Idle:  filetimeTicks(idle),
		Total: filetimeTicks(kernel) + filetimeTicks(user),
	}, nil
}

// filetimeTicks returns ft as one 64-bit count of 100-nanosecond
// units.
func filetimeTicks(ft syscall.Filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}
