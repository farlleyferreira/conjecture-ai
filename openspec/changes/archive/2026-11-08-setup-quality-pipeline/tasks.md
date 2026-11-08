# Tasks

## 1. Commit & versioning conventions

- [x] 1.1 Add commitlint (`@commitlint/config-conventional`) + husky
      `commit-msg` hook
- [x] 1.2 Add `semantic-release` config (`.releaserc.json`) with changelog +
      git + github plugins
- [x] 1.3 Add `CHANGELOG.md` (Keep a Changelog header, semantic-release owns
      the entries)
- [x] 1.4 Document the convention in `CONTRIBUTING.md`

## 2. PR quality & security pipeline

- [x] 2.1 `ci.yml`: commitlint over the PR commit range
- [x] 2.2 `ci.yml`: gitleaks secret scan
- [x] 2.3 `ci.yml`: lint/test placeholder job (stack not chosen yet)
- [x] 2.4 `pr-title.yml`: Conventional Commit PR title check
- [x] 2.5 `dependency-review.yml`: dependency review action
- [x] 2.6 Enable CodeQL default setup (auto language detection)
- [x] 2.7 `.github/dependabot.yml` for `github-actions` + `npm`
- [x] 2.8 `release.yml`: `semantic-release` on push to `main`

## 3. Repository configuration

- [x] 3.1 Restrict merge strategy to squash-only, PR title as commit message
- [x] 3.2 Configure branch protection on `main`: required status checks,
      required PR review, no force-push/deletion

## 4. Decision record

- [x] 4.1 Initialize OpenSpec in the repo (`openspec init`)
- [x] 4.2 Write this proposal/design/tasks set documenting the pipeline
      decisions
- [x] 4.3 Archive this change once merged, so it becomes permanent history
      under `openspec/changes/archive/`
