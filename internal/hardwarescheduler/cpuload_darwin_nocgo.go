//go:build darwin && !cgo

package hardwarescheduler

import "errors"

// ErrCPULoadUnavailable is what [CurrentCPUSample] returns on macOS
// from a binary built without CGO: go-osstat reads Darwin CPU
// counters through a Mach host call it can only make via CGO, and
// has no pure-Go reading for that one OS (Linux and Windows are
// unaffected). Every caller already treats an unmeasurable load as
// "not busy" (see cmd/conjecture/watch.go shouldDeferThisTick), so
// "conjecture watch" still runs here - it only never defers a scan.
var ErrCPULoadUnavailable = errors.New("hardwarescheduler: CPU load is not measurable on macOS without CGO")

// CurrentCPUSample is the CGO-free macOS stand-in for the real
// reading in cpuload_read.go: it reports [ErrCPULoadUnavailable]
// rather than a made-up sample.
func CurrentCPUSample() (CPUSample, error) {
	return CPUSample{}, ErrCPULoadUnavailable
}
