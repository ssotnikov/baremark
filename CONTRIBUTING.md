# Contributing

Thank you for contributing.

This document defines the conventions for branches, commits, pull requests, reviews, and releases. The goal is to keep the repository history consistent, changes easy to review, and collaboration predictable.

---

## General principles

Before making a change:

1. Check whether a related issue or pull request already exists.
2. For significant changes, consider opening or discussing an issue before implementation.
3. Keep each change focused on a single purpose.
4. Follow the project's existing architecture, coding style, and conventions.
5. Never commit credentials, tokens, private keys, personal data, or other sensitive information.

When project-specific instructions conflict with this document, the project-specific instructions take precedence.

---

## Branches

### Naming format

Use:

```text
<type>/<short-description>
```

If the work is associated with an issue or task:

```text
<type>/<issue-id>-<short-description>
```

Examples:

```text
feat/add-user-authentication
feat/42-add-user-authentication
fix/128-handle-empty-response
docs/update-installation-guide
refactor/extract-storage-interface
```

### Branch types

| Prefix | Purpose |
|---|---|
| `feat/` | New functionality |
| `fix/` | Bug fix |
| `docs/` | Documentation changes |
| `refactor/` | Code restructuring without changing external behavior |
| `perf/` | Performance improvement |
| `test/` | Adding or updating tests |
| `build/` | Build system or dependency changes |
| `ci/` | CI/CD configuration |
| `chore/` | Maintenance and other non-product changes |
| `revert/` | Reverting an earlier change |

### Branch naming rules

1. Use lowercase ASCII characters.
2. Use `kebab-case` for the description.
3. Do not use spaces, underscores, or unnecessary punctuation.
4. Keep names concise and descriptive.
5. Describe the result rather than the implementation process.
6. Include an issue or task ID when useful for traceability.
7. Do not include personal names in branch names unless they are part of the domain being changed.

Good:

```text
feat/add-export-command
fix/handle-null-response
fix/142-prevent-duplicate-records
docs/update-api-reference
refactor/extract-cache-provider
perf/reduce-startup-time
ci/add-security-scan
```

Avoid:

```text
new-feature
my-branch
user-fix
fix_stuff
feat/AddNewFeature
feat/non-ascii-description
fix/trying-to-fix-cache
```

Delete branches after they have been merged unless there is a specific reason to retain them.

---

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/) style:

```text
<type>(<optional-scope>): <description>
```

Examples:

```text
feat(auth): add OAuth login
fix(api): handle empty response
docs: update installation instructions
refactor(storage): extract cache interface
perf(parser): reduce allocation count
test(api): add timeout coverage
ci: add dependency audit
```

### Commit types

The recommended types are:

```text
feat
fix
docs
refactor
perf
test
build
ci
chore
revert
```

### Commit message rules

The subject should:

- use the imperative mood where practical;
- be concise and specific;
- describe what the commit changes;
- start with a lowercase letter after the type;
- not end with a period;
- avoid vague descriptions such as `fix bug`, `update code`, or `changes`.

Prefer:

```text
fix(parser): handle empty input
```

over:

```text
fix: fixed some parser stuff
```

### Atomic commits

Each commit should represent one logical change.

Prefer several focused commits over one commit containing unrelated changes.

A commit should ideally be:

- understandable on its own;
- safe to review independently;
- easy to revert;
- free of unrelated formatting or refactoring changes.

Do not mix a feature, unrelated refactoring, dependency updates, and documentation cleanup into the same commit unless they are inseparable parts of the same change.

### Commit body

Use a commit body when the reason for a change is not obvious from the subject. Explain why the change was necessary, important behavior or design decisions, and any compatibility implications or non-obvious trade-offs.

Focus on **why**, not on repeating what is already visible in the diff.

### Issue references

Reference related issues when appropriate:

```text
Refs #123
Fixes #123
Closes #123
```

Use `Fixes` or `Closes` only when merging the change should actually close the issue.

### Breaking changes

Breaking changes must be explicitly identified.

Use either:

```text
feat(api)!: remove deprecated endpoint
```

or a footer:

```text
BREAKING CHANGE: the legacy endpoint has been removed.
```

Describe the required migration whenever possible.

---

## Pull requests

A pull request should represent one logical change. Avoid combining unrelated features, fixes, refactoring, dependency upgrades, or formatting changes in a single PR.

Smaller, focused PRs are generally easier to review, test, merge, and revert.

### PR title

Use the same Conventional Commits format as commit messages:

```text
<type>(<optional-scope>): <description>
```

Examples:

```text
feat(auth): add OAuth login
fix(api): prevent duplicate requests
docs: update deployment guide
refactor(storage): extract cache provider
```

This is especially important when the repository uses squash merging because the PR title commonly becomes the final commit message.

Do not include an issue ID in the PR title unless the repository explicitly requires it. Reference the issue in the PR description instead.

### PR description

A PR should explain:

```markdown
## Summary

What changed?

## Motivation

Why is this change needed?

## Changes

What are the important implementation details?

## Testing

How was the change tested?

## Related issues

Closes #123
```

Add screenshots, recordings, logs, benchmarks, or other evidence when they materially help reviewers understand or validate the change. For UI changes, before/after screenshots are strongly recommended. For performance changes, include measurements or benchmarks when practical.

### Before opening a PR

Verify that:

- [ ] The change has a single clear purpose.
- [ ] The project builds successfully.
- [ ] Relevant automated tests pass.
- [ ] Linters and formatters pass.
- [ ] New behavior has appropriate tests where practical.
- [ ] Existing tests have been updated if behavior intentionally changed.
- [ ] Temporary debug code and diagnostic output have been removed.
- [ ] No unrelated files or formatting changes are included.
- [ ] Documentation has been updated when necessary.
- [ ] Public APIs and interfaces remain compatible unless a breaking change is intentional.
- [ ] Breaking changes are clearly documented.
- [ ] Dependencies added by the change are necessary and appropriate.
- [ ] Generated files are committed only when required by the project.
- [ ] No credentials, tokens, secrets, private keys, personal data, or confidential information are present.
- [ ] The branch is reasonably up to date with its target branch.
- [ ] The PR description explains what changed, why, and how it was tested.
- [ ] Related issues are referenced.

### Draft PRs

Use a Draft Pull Request when implementation is incomplete, CI is expected to fail temporarily, architectural feedback is needed before completing the implementation, or the PR is being shared for early discussion.

Mark the PR ready for review only when it is in a reviewable state.

---

## Code review

Reviewers should evaluate correctness, security, maintainability, readability, architecture, test coverage, compatibility, performance implications, documentation, and unnecessary complexity.

Comments should focus on the code and the technical decision rather than the author. Authors should address review comments by making the requested change, explaining why a different approach is preferable, or discussing alternatives until the concern is resolved.

Resolve review conversations only after the underlying concern has been addressed. Significant changes made after approval should be re-reviewed when appropriate.

---

## CI and required checks

Do not merge a PR while required CI checks are failing.

Depending on the project, required checks may include build, unit tests, integration tests, linting, formatting, type checking, security scanning, dependency auditing, static analysis, and coverage requirements.

Do not bypass required checks merely to merge faster. If a failing check is unrelated to the PR, document the reason and follow the repository's established exception process.

---

## Security and sensitive information

Never commit passwords, API keys, access tokens, private keys, certificates containing private keys, production credentials, `.env` files containing secrets, confidential infrastructure details, private customer or employee information, or production datasets containing sensitive information.

Before pushing, inspect the diff:

```bash
git diff
git diff --staged
```

If a secret has already been committed, **removing it in a later commit is not sufficient**. Treat the secret as compromised: revoke or rotate it immediately, remove it from the repository, clean the Git history when necessary, and notify the appropriate maintainers or security contact.

Use environment variables, secret managers, CI/CD secrets, or documented local configuration mechanisms instead.

Security vulnerabilities should not be disclosed through a public issue when the repository provides a private vulnerability-reporting mechanism or security contact.

---

## Dependencies

When adding or updating dependencies:

- prefer actively maintained packages;
- avoid unnecessary dependencies;
- review licensing implications where relevant;
- consider security advisories;
- avoid introducing large dependency trees for trivial functionality;
- update lock files together with dependency manifests when applicable.

Keep dependency-only changes separate from unrelated feature changes whenever practical.

---

## Documentation

Update documentation when a change affects installation, configuration, user-visible behavior, command-line options, APIs, environment variables, deployment, compatibility, architecture, or important design decisions.

Code and documentation should describe the same behavior when the PR is merged.

---

## Generated files

Do not manually edit generated files unless the project explicitly requires it.

When generated artifacts are tracked by Git:

1. Modify the source.
2. Run the appropriate generator.
3. Commit the resulting generated files together with the source change.

Avoid committing build output, caches, temporary files, IDE state, or local environment files unless the repository explicitly tracks them.

---

## Versioning

Projects that publish releases should preferably follow [Semantic Versioning](https://semver.org/):

```text
MAJOR.MINOR.PATCH
```

General interpretation:

- **MAJOR** - incompatible or breaking changes;
- **MINOR** - backward-compatible functionality;
- **PATCH** - backward-compatible bug fixes.

Pre-release versions may use identifiers such as:

```text
1.4.0-alpha.1
1.4.0-beta.1
1.4.0-rc.1
```

The repository's release process remains authoritative if it defines different versioning rules.

---

## Releases

Before creating a release:

- ensure required CI checks pass;
- verify the version number;
- update release notes or changelog when used by the project;
- document breaking changes and migration steps;
- verify release artifacts;
- verify that no development-only or sensitive files are included;
- create the release from the intended commit or protected branch.

Release tags should follow one consistent repository-wide convention, for example `v1.2.3` or `1.2.3`. Do not mix tag formats within the same project.

---

## Merge strategy

Follow the merge strategy configured by the repository.

When **squash merge** is used, the PR title should be suitable as the final commit subject, the final commit should follow the repository's commit convention, and temporary or work-in-progress commits do not need to be preserved.

When **rebase merge** is used, individual commits should be clean, meaningful, and independently understandable.

When **merge commits** are used, keep the branch history reasonably clean and avoid unnecessary merge commits from repeatedly merging the target branch into the feature branch.

Do not rewrite the history of shared or protected branches unless explicitly authorized.

---

## Keeping a branch up to date

When required, update your branch from the target branch using the workflow established by the project.

For private feature branches, rebasing is generally appropriate:

```bash
git fetch origin
git rebase origin/main
```

After rebasing a branch that has already been pushed, prefer:

```bash
git push --force-with-lease
```

over:

```bash
git push --force
```

`--force-with-lease` helps prevent accidentally overwriting commits pushed by someone else.

Do not rebase shared branches without coordinating with other contributors.

---

## Repository hygiene

Keep the repository clean:

- do not commit editor-specific files unless intentionally shared;
- do not commit temporary files or caches;
- do not commit local configuration containing machine-specific values;
- avoid unrelated whitespace or formatting changes;
- remove obsolete code rather than leaving large commented-out blocks;
- keep `.gitignore` appropriate for the project's technology stack.

Before committing, review:

```bash
git status
git diff
git diff --staged
```

Before opening a PR, review the complete branch diff against the target branch.

---

## Contributor checklist

Before requesting review:

- [ ] Branch name follows the repository convention.
- [ ] Commits are focused and meaningful.
- [ ] Commit messages follow Conventional Commits.
- [ ] The PR contains one logical change.
- [ ] The PR title follows Conventional Commits.
- [ ] The PR description explains what changed and why.
- [ ] Related issues are linked.
- [ ] Build and required checks pass.
- [ ] Relevant tests pass.
- [ ] New or changed behavior is tested where appropriate.
- [ ] Documentation is up to date.
- [ ] No debug or temporary code remains.
- [ ] No secrets or sensitive information are included.
- [ ] The diff contains no unrelated changes.
- [ ] Breaking changes are explicitly documented.
- [ ] The PR is ready for review.

---

Thank you for helping keep the project maintainable, secure, and easy to contribute to.
