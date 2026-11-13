# Proposal

## Why

A closed-beta-readiness audit found no `recover()` anywhere in
`cmd/conjecture` or `internal/` (confirmed by grep): an unexpected
panic in any of the 20+ subcommands crashed the whole process with a
raw Go stack trace on stderr. Not a data-loss risk, but a bad look for
a tester or partner who isn't the developer reading it, and explicitly
flagged before releasing a closed beta.

## What Changes

- `cmd/conjecture/main.go`'s command dispatch now recovers from a panic
  in any subcommand, printing one plain `"conjecture: internal error:
  %v"` line to stderr and exiting 1, instead of letting the panic crash
  the process with its own stack trace.
- The recovery itself is pulled into one small, directly testable
  function (`runRecovered`), rather than inlined into `dispatch`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None - the switch statement actually dispatching commands is otherwise
byte-for-byte unchanged, just renamed from being `dispatch`'s own body
to a separate `route` function `runRecovered` wraps. `skip_specs:
true`.

## Impact

- Affected paths: `cmd/conjecture/main.go`, `cmd/conjecture/main_test.go`.
- No command's own behavior changes when it does not panic - same
  output, same exit code, for every existing passing test.
- Verified: `go test ./...` passes; new tests exercise `runRecovered`
  directly with both a panicking and a non-panicking `fn`, confirming
  the panic case reports a plain error line (not a stack trace) and
  exits 1, and the non-panic case passes its own exit code through
  untouched.
- No new dependency.
