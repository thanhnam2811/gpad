# Contributing to Gopad++ (gpad)

Thank you for your interest in contributing to Gopad++!

Gopad++ follows a strict **Benchmark-Driven Development (BDD)** engineering discipline. We value observable evidence over subjective assumptions.

---

## Code of Engineering Rigor

1. **Never Guess Performance**:
   - Any modification touching I/O, session persistence, buffer mutation, search, or UI rendering MUST be accompanied by an empirical benchmark suite inside `*_bench_test.go`.
   - Implement at least 2-3 candidate solutions and evaluate latency (`ns/op`), heap footprint (`B/op`), and allocation count (`allocs/op`).
   - Run `go test -bench=. -benchmem -count=5` directly on local hardware.
2. **Zero Race Conditions**:
   - `go test -race ./...` must exit with return code `0` before any pull request is opened.
3. **High-Density Compact UI Philosophy**:
   - Gopad++ is strictly **zero-webview and zero-Electron**.
   - No unnecessary whitespace, bloated margins, or chunky web-style paddings.
   - Respect GNOME 46 `prefer-dark` guidelines.
   - Maintain the steady-state render frame budget < 8.33 ms (120 FPS).
4. **Conventional Commits**:
   - Commit messages must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:
     `feat(subsystem): brief summary [metrics, ADR-ID]`
     `fix(subsystem): brief summary`
     `perf(subsystem): brief summary [metrics, ADR-ID]`

---

## Submitting Pull Requests

1. Fork the repository and create your branch from `main`.
2. Ensure all tests pass:
   ```bash
   go test -race ./...
   ```
3. If introducing or changing performance-sensitive paths, document an Architecture Decision Record (ADR) in `docs/perf-decisions.md`.
4. Open a Pull Request with a clear description of the problem, candidate solutions, and benchmark numbers.
