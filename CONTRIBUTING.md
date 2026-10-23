# Contributing to Gopad++

Small, focused improvements are welcome: reproducible bugs, accessibility findings,
documentation, desktop integration, and measured changes to editing behavior.
Discuss larger changes in an issue before investing in an implementation.

## Get started

Use Go as pinned in [go.mod](go.mod), Linux and the runtime libraries in [README](README.md).
The local `third_party/mygo` replacement is part of the source tree; keep its license and
attribution intact. No Node, Bun or frontend build step is needed for GPAD.

```sh
go mod verify
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

Format Go files you change with `gofmt`. Keep changes narrowly scoped and avoid unrelated
formatting. Follow [AGENTS.md](AGENTS.md) and [docs/workflow.md](docs/workflow.md) for the
repository's architecture, automated-agent role sequence and review gates. No commit should
claim a review or benchmark that did not happen.

## Reproduce safely

Use `t.TempDir()`, `app.NewAppWithSessionDir(...)`, synthetic documents and isolated sockets.
Do not call `app.NewApp()` in a test: it uses the current user's session. Join background
workers and call Stop/Close during test cleanup. Never benchmark against real session data.

For graphical checks, use [scripts/gui-smoke.py](scripts/gui-smoke.py). It owns its Xvfb,
D-Bus session, HOME and XDG paths. Do not run focus experiments, copy/paste automation or
scripts that manipulate windows on another person's active desktop. Older diagnostic scripts
may have different isolation assumptions; read them before use.

## Changes and evidence

- Add a regression for a confirmed bug. Explain the failing behavior and the repaired behavior.
- Keep persistence errors visible and preserve fsync, shadow/WAL ordering and shutdown ownership.
  Data integrity takes priority over a faster benchmark.
- For a performance-sensitive algorithm change, retain 2–3 candidate benchmarks, run
  `-benchmem -count=5`, and record the hardware, exact command, raw results, variance and choice
  in [docs/perf-decisions.md](docs/perf-decisions.md). Documentation-only changes do not need
  artificial algorithm benchmarks.
- Keep editor.State on the UI thread. Background work uses synchronized app/document methods.
- Preserve compact native controls and test light/dark presentation. A 120 Hz frame budget and
  zero allocations are goals, not existing guarantees.
- Check new dependencies and assets for license compatibility. Include actual license texts
  in release artifacts; document visual sources in [BRANDING.md](BRANDING.md).

## Pull requests

Branch from `main`, use a Conventional Commit title such as `fix(session): retain failed drafts`,
and describe the concrete problem, final behavior, tests and remaining limits. Link issues where
applicable. Include screenshots for visible changes and measurements for performance changes.
Do not include personal documents, credentials or session WALs in examples or attachments.

Independent review is required before merging. Automated-agent commits also require the
Agent-4 sign-off specified by AGENTS.md. Please keep discussion constructive, explain tradeoffs
and respect other contributors; harassment and disclosure of private information are not welcome.

See [SECURITY.md](SECURITY.md) for sensitive reports and [docs/RELEASING.md](docs/RELEASING.md)
for the separate, manual release process. Contributions are made under the project's MIT license;
third-party material keeps its own compatible license and notices.
