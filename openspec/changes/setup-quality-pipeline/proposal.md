# Proposal

## Why

The repository was created empty (LICENSE + README only) with no agreed
convention for commit history, versioning, release notes, or merge quality
gates. Without this, history quickly becomes inconsistent, versions are
bumped by hand/inconsistently, and nothing stops a regression or a leaked
secret from landing on `main`. This needs to be in place before any
application code is added.

## What Changes

- Adopt [Conventional Commits](https://www.conventionalcommits.org/) for all
  commit messages and PR titles, enforced locally (husky + commitlint) and in
  CI (`commitlint-github-action`, `action-semantic-pull-request`).
- Adopt [Semantic Versioning](https://semver.org/) with versions, Git tags,
  GitHub Releases and `CHANGELOG.md` generated automatically by
  `semantic-release` from commit history on every merge to `main`.
- Require PRs into `main` to go through squash merge only, so the PR title
  (a Conventional Commit) becomes the single commit on `main` that
  `semantic-release` analyzes.
- Add a quality & security pipeline that every PR into `main` must pass:
  commit message / PR title linting, CodeQL static analysis (default setup,
  auto language detection), secret scanning (`gitleaks` in CI, plus GitHub's
  native secret scanning/push protection), and a dependency review check.
- Add Dependabot for GitHub Actions and npm dependency updates.
- Adopt OpenSpec (this directory) as the place technical decisions are
  recorded going forward: every change that involves a non-obvious technical
  decision gets a `design.md`, and archived changes remain in
  `openspec/changes/archive/` as permanent, discoverable documentation.

## Capabilities

### New Capabilities

(none - this change only sets up repository tooling/process, it does not add
an application capability)

### Modified Capabilities

(none)

This is a pure tooling/process change with no spec-level system behavior;
see `.openspec.yaml` (`skip_specs: true`).

## Impact

- Affected paths: `package.json`, `.releaserc.json`, `commitlint.config.js`,
  `.husky/`, `CHANGELOG.md`, `CONTRIBUTING.md`, `.github/workflows/*`,
  `.github/dependabot.yml`, repository branch protection and merge settings.
- New dev-only dependency on Node.js/npm for repo tooling (semantic-release,
  commitlint, husky, OpenSpec CLI), independent of whatever stack the
  application itself ends up using.
- Every future PR into `main` must pass the new required checks before it can
  be merged.
