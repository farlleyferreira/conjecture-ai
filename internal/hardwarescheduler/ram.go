// Package hardwarescheduler implements the hardware-scheduler
// capability's own-machine checks (openspec/specs/hardware-scheduler/
// spec.md): the minimum hardware requirement, and the local
// scheduling policy (see schedule.go) that keep the adversarial SLM,
// the system under test, and heavy background work from competing for
// the same machine at the wrong time. Measuring the facts that policy
// decides over - whether the system under test is currently under
// heavy load, whether the user is actively using the machine - is not
// implemented here: no caller in this codebase runs the adversarial-
// SLM attack loop or a background verification pass from a real
// invocation yet to have real load to measure (see schedule.go's own
// doc comment).
package hardwarescheduler

import (
	"fmt"

	"github.com/pbnjay/memory"
)

// MinimumRAMBytes is the specification's own number: "The system SHALL
// require at least 6 GB of RAM on the machine running conjecture and
// the adversarial SLM."
const MinimumRAMBytes = 6 * 1024 * 1024 * 1024

// CheckMinimumRAM reports an error naming the machine's total system
// RAM when it falls below [MinimumRAMBytes], or when that total could
// not be determined at all (rather than silently treating "unknown"
// as "sufficient") - "Startup checks available RAM": "WHEN conjecture
// starts on a machine with less than 6 GB of RAM THEN the system
// reports that the minimum hardware requirement is not met."
func CheckMinimumRAM() error {
	return checkRAM(memory.TotalMemory())
}

// checkRAM is [CheckMinimumRAM]'s machine-independent half, so a test
// can exercise the threshold logic itself against a chosen value
// instead of whatever RAM happens to be installed on the machine
// running the test.
func checkRAM(totalBytes uint64) error {
	if totalBytes == 0 {
		return fmt.Errorf("hardwarescheduler: could not determine the machine's total system RAM")
	}
	if totalBytes < MinimumRAMBytes {
		return fmt.Errorf(
			"hardwarescheduler: minimum hardware requirement not met: %.1f GB of RAM detected, want at least 6.0 GB",
			float64(totalBytes)/(1024*1024*1024),
		)
	}
	return nil
}
