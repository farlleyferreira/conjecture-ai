//go:build !darwin || cgo

package hardwarescheduler

import (
	"github.com/mackerelio/go-osstat/cpu"
)

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
