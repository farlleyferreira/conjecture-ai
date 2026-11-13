# Proposal

## Why

A pasted CI notification ("Release / semantic-release (push) Failing
after 17s") on a recent merge led to checking the Release workflow's own
history: every single run since `v1.1.0` (2026-11-03, the day PR #12
bumped `semantic-release` from 24.2.9 to 25.0.9) has failed - over 100
consecutive runs, confirmed via `gh run list`. No new version, Git tag,
GitHub Release, or `CHANGELOG.md` entry has been generated for any of the
dozens of real features merged since, despite `openspec/config.yaml`'s
own stated process saying this happens automatically on every merge to
main.

Root cause, confirmed by reproducing the exact failure locally: `.github/
workflows/release.yml` pins Node 20, but `semantic-release@25.0.9`
declares an engine requirement of `^22.14.0 || >=24.10.0` - its own `npm
ci` fails under Node 20's older bundled npm (10.8.2) with `Missing:
conventional-commits-filter@6.0.1 from lock file`, even though
`package-lock.json` and `package.json` are themselves mutually
consistent (confirmed: a clean `npm ci` under Node 24/npm 11.16.0
succeeds with zero changes to either file, and a `--dry-run --no-ci`
`semantic-release` invocation loads every plugin and correctly reaches
its own "not on main" gate).

## What Changes

- `.github/workflows/release.yml`'s `node-version` bumped from `20` to
  `24` - the exact version verified locally to make `npm ci` and
  `semantic-release` both succeed.
- No change to `package.json`/`package-lock.json` - they were already
  correct; only the CI runner's Node version was wrong.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None - a CI configuration fix, not a product behavior change.
`skip_specs: true`.

## Impact

- Affected paths: `.github/workflows/release.yml`.
- No Go code or test changes; `go build`/`go vet` pass unchanged.
- Verified locally: `npm ci` (Node 24.18.0 / npm 11.16.0) succeeds
  cleanly against the committed lock file with zero package changes;
  `npx semantic-release --dry-run --no-ci` loads every configured plugin
  and correctly refuses to publish from a non-`main` branch - the same
  two steps the workflow itself runs, both now proven to work under the
  Node version this change pins.
- Once merged, the very next real merge to `main` is the first real test
  of whether a release actually gets cut again - this change only fixes
  the CI configuration, it does not itself produce a release (nothing to
  release from a non-`main` branch).
