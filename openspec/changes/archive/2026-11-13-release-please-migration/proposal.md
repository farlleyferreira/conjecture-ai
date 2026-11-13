# Proposal

## Why

The prior fix (bumping Node 20→24, #172) got `npm ci`/`semantic-release`
itself running again, but surfaced the real, deeper root cause: `main`'s
branch protection requires 4 status checks on every push, and
`semantic-release`'s own `@semantic-release/git` plugin pushes its
release commit (`chore(release): ...`) and tags directly to `main` -
exactly the push branch protection exists to block. This repository is
owned by a personal account, not an organization, so neither classic
protection's push-allowlist nor a Ruleset's `Integration`/`RepositoryRole`
bypass actor can be scoped to just the Actions bot (both confirmed by
directly attempting them against the live repo and reading the exact
GitHub API rejection); the only actor that *can* bypass is a human
identity authenticated via a personal access token, which was explicitly
declined in favor of not introducing a new credential.

Explicit direction: switch to a release tool that works *through* a pull
request instead of a direct push, so branch protection never needs to be
touched at all.

## What Changes

- Replaced `semantic-release` with
  [`release-please`](https://github.com/googleapis/release-please)
  (`googleapis/release-please-action@v4`). On every push to `main`, it
  opens/updates a standing release PR (version bump + `CHANGELOG.md`
  entry + release notes) instead of pushing directly. Merging that PR -
  a normal PR, through the normal required checks - is what creates the
  Git tag and GitHub Release.
- `.github/workflows/release.yml` rewritten: no more Node/npm setup at
  all (`release-please-action` is a self-contained action, not an
  npm-script-driven job) - this also removes the whole class of
  `npm ci`/lockfile-drift risk the prior two fixes were chasing.
- Removed `semantic-release`, `@semantic-release/changelog`,
  `@semantic-release/git` from `package.json`/`package-lock.json`
  (incidentally dropped all 37 vulnerabilities that were in
  `semantic-release`'s own dependency tree - confirmed via `npm audit`
  before/after). Deleted `.releaserc.json`.
- Added `release-please-config.json` (release-type `simple`) and
  `.release-please-manifest.json` (seeded at `1.1.0`, the last real
  tag/release that actually shipped, so the next computed version
  continues from there instead of restarting).
- Updated `CONTRIBUTING.md`, `openspec/config.yaml`, and `CHANGELOG.md`'s
  own footer note to describe the new PR-based flow instead of
  semantic-release's direct-push one.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None - a release-tooling swap, not a product behavior change.
`skip_specs: true`.

## Impact

- Affected paths: `.github/workflows/release.yml`, `package.json`,
  `package-lock.json`, `.releaserc.json` (deleted),
  `release-please-config.json` (new), `.release-please-manifest.json`
  (new), `CONTRIBUTING.md`, `openspec/config.yaml`, `CHANGELOG.md`.
- No Go code or test changes; `go build`/`go vet` unaffected.
- Verified locally: `npm ci` succeeds cleanly against the regenerated
  lock file with the three semantic-release packages removed
  (`npm audit` now reports 0 vulnerabilities, down from 37).
- The real, end-to-end test is the next merge to `main`: this change
  cannot itself prove a release gets cut (that requires release-please's
  own release PR to be opened, then merged) - watch for that PR after
  this merges, and merge it to confirm a tag/release actually appears.
- No change to branch protection/rulesets was needed or made by this
  change itself - the earlier experimental Ruleset (created while
  diagnosing this) is being reverted back to the original classic
  protection separately, since it is no longer needed.
