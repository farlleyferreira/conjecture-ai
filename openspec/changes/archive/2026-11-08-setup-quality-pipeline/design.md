# Design

## Context

New, empty repository (`farlleyferreira/conjecture-ai`), public, MIT
licensed. No application stack has been chosen yet. The request was to put
in place, as market-standard as possible: semantic commits, semantic
versioning with an always-up-to-date `CHANGELOG.md`, and a quality +
security pipeline gating every PR into `main` - plus a durable, in-repo
record of technical decisions (this document, via OpenSpec).

## Goals / Non-Goals

**Goals:**
- Fully automatic version bump + changelog + GitHub Release on every merge
  to `main`, derived only from commit history (no manual version edits).
- Make it structurally hard to merge a non-conventional commit or a PR that
  fails lint/tests/security checks into `main`.
- Keep the setup stack-agnostic where the application code itself is
  concerned (no app stack has been chosen yet), while still being concrete
  and enforced today.
- Record decisions where a future contributor (human or AI) would otherwise
  have to reverse-engineer "why is it like this".

**Non-Goals:**
- Choosing the application's language/framework (tracked separately, once
  decided).
- Publishing a package to the npm registry - `package.json` here exists only
  to host repo tooling (semantic-release, commitlint, husky, OpenSpec CLI),
  hence `"private": true`.

## Decisions

1. **Conventional Commits, enforced twice (local + CI).**
   Local enforcement (husky `commit-msg` hook) gives fast feedback; CI
   enforcement (`wagoid/commitlint-github-action` +
   `amannn/action-semantic-pull-request`) is the actual gate, since local
   hooks can be bypassed (`--no-verify`) or skipped (no `npm install`).

2. **Squash merge only, PR title = the commit on `main`.**
   `semantic-release` decides the next version from the commit messages that
   land on `main`. If merge commits or rebase-merge were allowed, the commits
   `semantic-release` sees would be the many small, messy commits from the
   PR branch instead of one clean Conventional Commit. Squash merge with
   "default to PR title" makes the PR title the single source of truth, so
   enforcing the PR title format (not just individual commits) is what
   actually matters for correct version bumps.

3. **`semantic-release` over a manual/CLI tool (`standard-version`, etc.).**
   Chosen so no human has to remember to bump the version or write the
   changelog entry; it runs unattended in CI on every push to `main` and
   produces the tag, GitHub Release, and `CHANGELOG.md` update atomically.
   Trade-off: an extra CI job with `contents: write` permission, and a
   generated `chore(release): x.y.z [skip ci]` commit back onto `main`
   (mitigated by `[skip ci]` to avoid a release loop).

4. **No `@semantic-release/npm` plugin.**
   `package.json` in this repo is tooling-only (`"private": true`), not a
   published package - there is nothing to publish to npm. Keeping the
   plugin out avoids a prepare/publish step that would otherwise require npm
   registry credentials for no benefit.

5. **CodeQL via "default setup" (GitHub-managed), not a hand-written
   workflow.**
   No application language exists yet, so a hand-written CodeQL workflow
   would need a hardcoded language matrix that may not match the stack once
   chosen (and would fail/no-op until then). Default setup auto-detects
   languages as code is added and GitHub keeps the underlying workflow
   up to date, which removes a maintenance burden that has no payoff today.

6. **`gitleaks` in CI in addition to GitHub's native secret scanning.**
   Public repos get GitHub secret scanning automatically, but an explicit
   `gitleaks` CI job (a) runs push-protection-equivalent checks even if
   native secret scanning/push protection is ever reconfigured, and (b) is
   visible as a normal required status check like the rest of the pipeline,
   which is more visible in the PR UI than a security tab alert.

7. **`actions/dependency-review-action` even though there are almost no
   dependencies yet.**
   Cheap to add now (no-op on a PR that doesn't change dependencies),
   becomes useful immediately once an application stack brings in real
   third-party packages, and is the standard GitHub-recommended action for
   this purpose.

8. **OpenSpec as the decision log, not a separate ad-hoc `docs/decisions/`
   ADR folder.**
   OpenSpec was explicitly requested for this project. Its `design.md` per
   change already covers "record the technical decision and why", and
   archived changes (`openspec/changes/archive/<change>/`) stay in the repo
   permanently and are browsable (`openspec view`, `openspec show`) - a
   second, parallel ADR system would just duplicate this with a different
   format. This document is the first example: it records the decisions
   behind the pipeline itself.

## Risks / Trade-offs

- **Dev-tooling supply chain:** `npm install` currently reports moderate/high
  `npm audit` findings inside `semantic-release`'s own transitive
  dependencies (sigstore/npm-internals used for its own provenance tooling),
  not in any runtime code of this project. Tracked via Dependabot; revisit if
  `npm audit` flags something reachable from the actual release code path.
- **Squash-only merge strategy** loses individual commit granularity on
  `main` for PRs with many commits. Accepted: `git log` on the PR branch
  (via the PR link) still has the detail if ever needed, and a clean `main`
  history is what `semantic-release` and `CHANGELOG.md` readers actually
  benefit from.
- **CodeQL default setup** will show zero findings and no check run at all
  until the first language is detected (first source file pushed) - this is
  expected, not a misconfiguration.
- **`CHANGELOG.md`/version bumps happen automatically** - a contributor who
  writes a non-conventional but technically "fix-shaped" commit message may
  get no release, or an unexpected one. Mitigated by enforcing the format at
  the PR-title level (decision 1/2), which is the only place that matters
  for `main`.
