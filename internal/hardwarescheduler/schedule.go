package hardwarescheduler

// ShouldDeferHeavyWork reports whether heavy adversarial-SLM or
// background-verification work should be deferred right now, given
// the caller-supplied facts about current load. Measuring
// systemUnderTestLoad and userActive themselves is an OS-level
// concern this function itself does not have an opinion on - same as
// [CheckMinimumRAM] separates the real RAM reading (memory.TotalMemory)
// from its own pure threshold check ([checkRAM]). userActive is now
// measured for real by this same package (cpuload.go) and wired into
// "conjecture watch"; systemUnderTestLoad still is not (see ram.go's
// own doc comment for why). This is the policy two of the
// specification's own requirements state, regardless of which of its
// two inputs a given caller can actually supply a real measurement
// for yet:
//
//   - "Mutual exclusion between SLM and system-under-test load": "the
//     system defers heavy adversarial SLM work rather than running
//     both simultaneously" when the system under test is under heavy
//     load.
//   - "Heavy work only on an idle machine": "the system does not run
//     heavy background verification work" while the user is actively
//     using the machine.
//
// Both requirements independently demand deferring, so either fact
// being true defers - this package does not invent a priority between
// the two reasons, since the specification gives none and a caller
// deferring work has no need of one either way.
func ShouldDeferHeavyWork(systemUnderTestLoad, userActive bool) bool {
	return systemUnderTestLoad || userActive
}
