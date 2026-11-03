# Contributing

## Commit messages

This project follows [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).
Every commit message (and every PR title, which becomes the squash commit message)
must match:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Common `type`s:

| Type       | Release impact                | Use for                                   |
|------------|--------------------------------|--------------------------------------------|
| `feat`     | minor (`0.X.0`)                | a new feature                              |
| `fix`      | patch (`0.0.X`)                | a bug fix                                  |
| `perf`     | patch                           | a performance improvement                  |
| `refactor` | none                            | code change that is not a fix or a feature |
| `docs`     | none                            | documentation only                         |
| `test`     | none                            | adding or fixing tests                     |
| `build`    | none                            | build system or dependencies               |
| `ci`       | none                            | CI configuration                           |
| `chore`    | none                            | maintenance, no production code change     |

Breaking changes: add `!` after the type/scope (e.g. `feat!: ...`) or a
`BREAKING CHANGE:` footer. This triggers a major release (`X.0.0`).

Examples:

```
feat(api): add endpoint to list conjectures
fix(parser): handle empty input without crashing
docs: update setup instructions in README
feat!: drop support for Node 18

BREAKING CHANGE: Node 20+ is now required.
```

Commit messages are validated locally via a `commit-msg` git hook
([commitlint](https://commitlint.js.org/)). In CI, only the **PR title** is
validated (it becomes the squash commit on `main`, which is what
semantic-release reads) — individual commits are not linted in CI, so
automated tools (e.g. Dependabot) can still open PRs without their own
commit messages being conventional.

## Versioning & releases

This project uses [Semantic Versioning](https://semver.org/) and
[semantic-release](https://semantic-release.gitbook.io/). Versions, Git tags,
GitHub releases and `CHANGELOG.md` are generated automatically from commit
history when a pull request is merged into `main` — never bump the version
or edit `CHANGELOG.md` by hand.

## Pull requests

- Branch off `main`, open a PR back into `main`.
- PRs are merged using **squash merge only**; the PR title must itself be a
  valid Conventional Commit, since it becomes the commit message on `main`.
- Every PR must pass the quality & security pipeline (lint/tests, CodeQL,
  secret scanning, dependency review) before it can be merged.

## Local setup

```bash
npm install
```

This installs the git hooks (via `husky`) that validate commit messages
before they are created.
