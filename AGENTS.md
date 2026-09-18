# AGENTS.md — BareMark

BareMark is a fast, lightweight native Windows Markdown viewer/editor written in Go.

This file contains operational instructions for AI coding agents working on the project. Product requirements are defined by the project specification; this file defines how an agent should work with the repository.

## Project priorities

Preserve these properties unless the specification is explicitly changed:

- fast startup and low memory usage;
- native Win32 UI;
- Windows AMD64 and ARM64 targets;
- one self-contained executable per target architecture;
- no CGO;
- no Electron, Chromium, WebView, .NET runtime, Java, Node.js, or other heavy runtime;
- no automatic network activity or telemetry;
- secure handling of untrusted Markdown;
- responsive UI even for large Markdown files.

## Read before changing code

Before implementing a task, read the relevant project documents:

1. `AGENTS.md`;
2. the current BareMark product/technical specification;
3. `go-win32-gui-spec.md` for Win32, rendering, DPI, theme, accessibility, and resource-lifetime rules;
4. the current development plan, if present;
5. `CONTRIBUTING.md`;
6. `docs/DECISIONS.md`;
7. existing code and tests in the area being changed.

The product specification is authoritative for product behavior and architecture.

Do not silently weaken or reinterpret a requirement to make implementation easier. If an implementation requires a material architectural deviation, stop at the decision point and record/propose an architectural decision first.

## Dependencies

The default rule is **zero third-party runtime dependencies**.

Use:

- Go standard library;
- direct WinAPI calls through `syscall.NewLazyDLL`;
- Windows system DLLs available on the supported Windows baseline.

Do not use CGO.

Do not add a third-party Go module merely for convenience. If a specification requirement cannot reasonably be implemented with the standard library and direct WinAPI, do not add a dependency automatically. Document the need, alternatives, security/licensing/performance impact, and obtain an explicit architectural decision before proceeding.

Build-time developer tools are not runtime dependencies, but they must not become requirements for running the final application.

## Platform separation

Windows-specific source must use Windows build constraints, for example:

```go
//go:build windows
```

Keep platform-independent logic free from WinAPI dependencies whenever practical so it can be tested on Windows, Linux, and WSL.

Typical candidates for platform-independent tests include:

- Markdown parsing/model logic;
- encoding and EOL handling;
- search;
- security/path/URI validation;
- settings validation;
- layout calculations that do not require HWND/HDC;
- document state machines;
- save/conflict decision logic.

The application itself is Windows-only. Linux/WSL is a **cross-build and test host**, not a supported runtime target.

## Build, run, and test

**After every code change, perform a build appropriate for the current host.**

Project build inputs:

- current application version: `./version.go`;
- Windows build script: `./build.ps1`;
- Linux/WSL cross-build configuration: `./Makefile`;
- build output directory: `./dist`.

Target architectures are:

```text
windows/amd64
windows/arm64
```

Do not introduce 32-bit Windows targets unless the specification is explicitly changed.

Expected versioned artifacts are:

```text
dist/baremark-{version}-amd64.exe
dist/baremark-{version}-arm64.exe
```

If the canonical build scripts use different artifact names, follow the scripts/specification and update this file rather than creating a parallel naming convention.

### Windows

```powershell
.\build.ps1
```

### Linux / WSL

```bash
make
```

Both canonical build entry points run `go test ./...`, `go vet -unsafeptr=false ./...`, and the Windows-target vet check with the same vet flag before compiling. Any failed check must stop the build.

Before release, also run:

```text
govulncheck ./...
```

On Windows, `./build.ps1 -Vulncheck` runs this scan before the build; on Linux/WSL, use `make vulncheck` separately. The scanner is a developer tool, not an application runtime dependency.

Use repository-provided scripts as the canonical build path. Do not duplicate build logic in ad-hoc commands when a script already exists.

Do not claim that a build, test, benchmark, fuzz run, or manual check passed unless it was actually executed successfully.

## Development workflow

For each task:

1. inspect the relevant specification, code, and tests;
2. check the working tree before editing;
3. define the smallest observable behavior required by the task;
4. add or update a failing test first where practical;
5. implement the smallest coherent change;
6. run focused tests while iterating;
7. run the required repository-wide checks before finishing;
8. review the final diff for unrelated changes, temporary diagnostics, generated files, and scope creep;
9. update documentation or `docs/DECISIONS.md` when required.

Keep the repository buildable after each logical change set.

Do not combine unrelated refactoring with a feature or bug fix.

When fixing a regression, add a regression test whenever technically practical.

## Go coding rules

- Run `gofmt` on changed Go files.
- Prefer simple, idiomatic Go over speculative abstractions.
- Keep package APIs small and explicit.
- Avoid global mutable state unless ownership is required by the Win32 lifecycle and is clearly documented.
- Do not panic for recoverable document or operation errors.
- Return errors with useful context.
- Keep user-facing messages separate from internal diagnostic details.
- Do not create unbounded goroutines, queues, caches, or retry loops.
- Every long-lived goroutine must have clear ownership and termination/cancellation behavior.
- Avoid retaining large document buffers through stale closures or caches.
- Prefer deterministic pure functions for document logic, security policy, layout, theme resolution, and settings validation.
- Keep WinAPI-specific code isolated from domain logic.
- Comments should explain non-obvious reasons and constraints rather than restating code.

Do not leave temporary `TODO`/`HACK` scaffolding in completed work unless it refers to a concrete tracked follow-up.

## Win32 and UI rules

Follow `go-win32-gui-spec.md` for detailed GUI behavior.

At minimum:

- call `runtime.LockOSThread()` before creating thread-affine Windows UI/COM/WinRT state;
- create/destroy HWND and mutate UI state on the UI thread;
- keep file I/O, large Markdown parsing, full-text search, print preparation, and PDF generation off the UI thread;
- never perform expensive work inside `WM_PAINT`;
- marshal background results back to the UI thread through the project's `WM_APP+n` mechanism;
- use revision/generation IDs or equivalent cancellation semantics so stale background results cannot replace newer state;
- explicitly own and release HWND/HDC/HBITMAP/HFONT/HICON and other native resources;
- never delete a GDI object while it is selected into a DC;
- correctly pair `BeginPaint`/`EndPaint`, `GetDC`/`ReleaseDC`, and other WinAPI acquire/release calls;
- implement Per-Monitor DPI Awareness V2 behavior;
- avoid continuous repaint loops when the interface is idle;
- preserve keyboard navigation, visible focus, High Contrast behavior, and accessibility.

If an API is unavailable on the supported Windows baseline, use a safe fallback rather than crashing.

## Large-file performance

Large Markdown files are a core BareMark use case.

Do not:

- create a bitmap for the entire document;
- synchronously parse/layout a large document on the UI thread;
- make unnecessary full copies of document text;
- introduce O(n²) processing over document size;
- retain stale parse/layout/search results.

Use viewport-oriented rendering and a small overscan where applicable.

Performance-sensitive changes must be checked against the project's defined 1 MB, 10 MB, and 50 MB fixtures/targets.

Do not weaken a performance target silently. If a target cannot be met, provide measurements and record/propose an architectural decision.

## File integrity

Data loss is a release-blocking defect.

- Never truncate the source document before a replacement version is safely prepared.
- Save through a temporary file in the same directory and use the safest available Windows replacement operation.
- Preserve supported source encoding and EOL style as defined by the specification.
- Preserve final-newline semantics where required.
- A failed save must leave the original document intact.
- Read-only documents may be edited in memory, but Save must fall back to Save As when required.
- Detect external modification.
- Never silently overwrite a disk version when unsaved local changes conflict with an external change.
- Closing a dirty document must follow the specified Save / Don't Save / Cancel behavior.

Every discovered data-loss or corruption bug should receive a regression test.

## Security

Treat Markdown content, file paths, links, image references, and filenames as untrusted input.

The application must not:

- execute Markdown content;
- execute JavaScript or active HTML;
- automatically invoke dangerous URI schemes;
- automatically access UNC/network resources during rendering;
- automatically download remote images or other document resources;
- launch executable content merely because it is referenced by a Markdown file;
- load DLLs through an unsafe search path influenced by the opened document directory.

Validate or safely handle:

- `javascript:` and other dangerous URI schemes;
- path traversal;
- malformed UTF-8/UTF-16;
- pathological nesting;
- extremely long lines;
- parser bombs;
- huge image dimensions;
- local and relative path resolution.

Never commit passwords, tokens, private keys, credentials, or sensitive local data.

Before release, run `govulncheck ./...` and review the final dependency/runtime surface.

## Assets and branding

Brand resources are source-of-truth assets and must not be regenerated or stylistically modified by the agent unless the task explicitly requests a branding change.

Application/document icons are stored in:

```text
assets/icons/
```

Primary ICO resources:

```text
BareMark-app-light.ico
BareMark-app-dark.ico
BareMark-file-light.ico
BareMark-file-dark.ico
```

Every canonical multi-resolution ICO must contain exactly these image sizes:

```text
16x16
24x24
32x32
48x48
256x256
```

Build validation must fail if a mandatory ICO is missing any required size or contains additional sizes.

The PE icon group IDs are fixed: `1` = primary dark application icon, `101` = light application icon, `201` = light file icon, `202` = dark file icon, and `32512` = dark `IDI_APPLICATION`. Keep this mapping identical in AMD64/ARM64 and debug/release builds. Verify the embedded image payloads before publishing an EXE.

The application logo/banner is:

```text
assets/app-logo.png
```

Use `assets/app-logo.png` for:

- `README.md`;
- the application About window.

Do not redraw, recolor, rename, replace, or regenerate these files without an explicit request.

Do not convert PNG to ICO at runtime.

Runtime-required resources must be embedded into the executable. The released application must not require the `assets` directory beside the EXE.

Missing mandatory build-time assets must cause a clear build failure.

## Testing expectations

Use TDD where technically meaningful.

Tests should cover applicable behavior such as:

- encoding detection and round-trip;
- CRLF/LF preservation;
- safe-save behavior;
- dirty-state transitions;
- external modification conflicts;
- Markdown-to-internal-model conversion;
- URI and path validation;
- search and replace;
- settings validation;
- layout/hit testing;
- theme resolution;
- pagination.

Maintain fuzz coverage for parser/security-sensitive code where appropriate.

Fuzz tests must not access the network or launch external processes.

Windows-specific behavior that cannot be meaningfully unit-tested should be verified with integration/system/manual checks instead of artificial mocks.

## Architectural decisions

All accepted architectural decisions are append-only in:

```text
docs/DECISIONS.md
```

Each decision must have:

- a unique decision number;
- the date of acceptance;
- a concise title;
- context/problem;
- considered alternatives where relevant;
- the accepted decision;
- rationale;
- consequences/trade-offs.

Recommended heading format:

```markdown
## DEC-0001 — YYYY-MM-DD — Short decision title
```

Never rewrite or delete an accepted decision to make history look current.

If a previous decision is changed:

1. append a new decision;
2. reference the superseded decision number;
3. explain why the earlier decision is being changed;
4. describe the consequences of the new decision.

## Scope control

Do not add functionality outside the current MVP scope merely because it is convenient or common in Markdown editors.

In particular, do not introduce without an explicit specification change:

- MDX;
- WYSIWYG editing;
- plugin execution;
- JavaScript extensions;
- Mermaid/LaTeX;
- cloud sync;
- accounts/authentication;
- collaborative editing;
- Git client features;
- embedded terminal;
- telemetry/analytics;
- auto-update service;
- tray/background resident mode;
- browser-engine-based primary rendering.

Prefer completing and hardening the defined MVP over adding convenience features.

## Git and pull requests

Follow `CONTRIBUTING.md` for branch naming, Conventional Commits, pull requests, reviews, and release workflow.

Keep each change focused on one logical purpose.

Do not create commits, push branches, open or merge pull requests, tag releases, or otherwise modify remote repository state unless the current task explicitly authorizes that action.

Before finishing, inspect:

```text
git status
git diff
git diff --staged
```

Do not include unrelated formatting changes, temporary output, local configuration, caches, build artifacts, or secrets.

When creating commits:

- use Conventional Commits;
- preserve the human developer as the primary author;
- add AI attribution using `Co-authored-by` trailers at the end of the commit message;
- for Claude Code contributions, add:
  `Co-authored-by: Claude <noreply@anthropic.com>`
- for ChatGPT/Codex contributions, add:
  `Co-authored-by: Codex <codex@openai.com>`
- when both Claude Code and ChatGPT/Codex contributed to the same commit, add both trailers;
- separate the trailers from the commit message body with a blank line.

## Completion report

When finishing a task, report:

- what changed;
- important implementation or architectural decisions;
- materially affected files/modules;
- tests/checks actually run and their results;
- Windows/manual checks actually performed;
- performance/security checks when relevant;
- remaining limitations or open questions.

If a required check could not be run, say exactly which check was not run and why.

## Definition of Done

A task is complete only when all applicable conditions are met:

- behavior matches the specification;
- relevant tests are added or updated and pass;
- changed Go code is formatted;
- required builds succeed;
- `go test ./...` passes;
- relevant `go vet` checks pass;
- errors and cancellation paths are handled;
- no new UI-thread blocking operation is introduced;
- native resource lifetime is correct;
- no obvious new GDI/User handle leak is introduced;
- no silent data-loss path is introduced;
- no unintended network activity is introduced;
- no unapproved runtime dependency is introduced;
- documentation and `docs/DECISIONS.md` are updated where required;
- the repository remains buildable.

When in doubt, choose the smallest change that preserves BareMark's core properties: speed, low resource use, native Windows behavior, secure handling of untrusted Markdown, and a self-contained executable.
