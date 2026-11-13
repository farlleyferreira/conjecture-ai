# Proposal

## Why

The first real release-please run (#173's own follow-up, PR #174
"chore(main): release 1.2.0") surfaced one more gap: `pr-title.yml`'s
"Conventional Commit title" check - one of the 4 branch-protection-
required status checks - never ran at all for that PR. Unlike `ci.yml`/
`dependency-review.yml` (triggered on `pull_request`, which land as
`action_required` and just need one manual approval for a PR authored by
the default `GITHUB_TOKEN`), `pr-title.yml` is triggered on
`pull_request_target`, which did not fire at all for this bot-authored
PR - leaving a required check permanently missing, which means the PR
could never satisfy "4 of 4 required status checks" and could never be
merged, with no bypass available (`enforce_admins: true`).

## What Changes

- `.github/workflows/pr-title.yml`'s trigger changed from
  `pull_request_target` to `pull_request` - the same event type the
  other two required checks already use, which is already confirmed to
  at least *register* (even if needing one manual "Approve and run")
  for a PR opened by `GITHUB_TOKEN`, unlike `pull_request_target`, which
  apparently does not register at all in that case.
- No change to what the check validates (still
  `amannn/action-semantic-pull-request@v6` checking the PR title is a
  valid Conventional Commit) or its permissions.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None - a CI trigger fix, not a product behavior change. `skip_specs:
true`.

## Impact

- Affected paths: `.github/workflows/pr-title.yml`.
- No Go code or test changes; `go build`/`go vet` unaffected.
- This PR itself is authored by a human (not `GITHUB_TOKEN`), so its own
  "Conventional Commit title" check still runs via the old
  `pull_request_target` trigger as usual - the fix only changes behavior
  for the next bot-authored PR (release-please's).
- Real verification is whether PR #174 (already open) picks up this
  fixed trigger once this merges and release-please next updates that
  PR - watched directly, not assumed.
