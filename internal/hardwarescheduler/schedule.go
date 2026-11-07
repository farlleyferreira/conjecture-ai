package hardwarescheduler

// ShouldDeferHeavyWork reports whether heavy adversarial-SLM or
// background-verification work should be deferred right now, given
// the caller-supplied facts about current load. Measuring
// systemUnderTestLoad and userActive themselves is an OS-level
// concern this package does not have an opinion on - same as
// [CheckMinimumRAM] separates the real RAM reading (memory.TotalMemory)
// from its own pure threshold check ([checkRAM]) - no caller in this
// codebase measures either yet (nothing here runs the adversarial-SLM
// attack loop or a background verification pass from a real
// invocation to have load to measure), so this is only the policy two
// of the specification's own requirements state once real
// measurements exist to feed it:
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
