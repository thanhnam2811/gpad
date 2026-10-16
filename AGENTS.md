# AGENTS.md — Constitution & Operational Framework for Gopad++ (gpad)

## 1. Hardware & Host Environment Baseline (Observed Reality)
- **OS**: Ubuntu 24.04.5 LTS (Noble Numbat) x86_64, Linux Kernel 7.0.0-38-generic
- **Desktop Environment**: GNOME Shell 46.0 (`org.gnome.desktop.interface color-scheme`: `'prefer-dark'`)
- **CPU**: Intel(R) Core(TM) i5-7500 CPU @ 3.40GHz (4 cores / 4 threads, 6MB L3 Cache, AVX2, BMI2)
- **Memory**: 11 GiB Total RAM (~3.6 GiB Available, 11 GiB Swap)
- **Go Runtime**: Go 1.27.2 linux/amd64
- **UI Platform**: Native GPU UI via `github.com/egoist/mygo/ui` (OpenGL on Linux, purego, strictly zero webview)

---

## 2. Multi-Agent Autonomous Squad Structure & Cross-Review Mechanism

Every feature, optimization, or lifecycle phase must pass through 4 independent roles in strict order:

### `[Agent-1: Lead Architect & Task Master]`
- Decomposes architecture and roadmap milestones in `docs/roadmap.md`.
- Defines decoupled boundaries and clean interfaces between Core systems (`internal/fileio`, `internal/session`, `internal/search`) and Native UI (`internal/ui`).
- Guarantees zero circular dependencies and clean domain isolation.

### `[Agent-2: Core Systems & Perf Engineer]`
- Implements 2-3 algorithmic candidate solutions for every performance-critical path.
- Runs empirical benchmarks directly on this local machine:
  `go test -bench=. -benchmem -count=5`
- Analyzes latency (`ns/op`), memory footprint (`B/op`), and allocations (`allocs/op`).
- Provides comparative analysis and selects the winning candidate with hard evidence.

### `[Agent-3: Native UI & Ergonomics Specialist]`
- Specializes in `github.com/egoist/mygo/ui` on GNOME 46.
- Enforces **High-Density Compact UI** design guidelines:
  * Compact Tab Bar: 28px height, 2-4px padding, dirty dot indicators.
  * Micro Status Bar: 20px height at bottom (`Ln X, Col Y | UTF-8 | LF | Size | Op Latency (µs)`).
  * 0px redundant padding/margins around code editor viewport.
  * Synchronized with GNOME 46 dark/light mode preference (`prefer-dark`).
  * Target render frame budget < 8.3ms/frame (120Hz display capability) with 0 allocations/frame on scroll/typing.

### `[Agent-4: Red-Team Reviewer & QA Gatekeeper]`
- Acts as an independent auditor with absolute veto power.
- **MUST REJECT** code from Agent-2 or Agent-3 if:
  1. An implementation choice was made arbitrarily without empirical benchmark numbers run on this machine.
  2. The corresponding Architecture Decision Record (ADR) has not been documented in `docs/perf-decisions.md`.
  3. Race detector fails (`go test -race ./...` exits non-zero) or memory leaks exist.
  4. UI ergonomics are bloated with excessive whitespace or Electron/Web-style margins.
- Only when `[Agent-4]` signs off with **APPROVED** may changes be committed to Git.

---

## 3. Benchmark-Driven Development (BDD) Discipline

1. **Never guess performance**: For any bottleneck, implement at least 2-3 candidate algorithms inside `*_bench_test.go`.
2. **Local execution**: Execute `go test -bench=. -benchmem -count=5` directly on this host CPU.
3. **Transparent documentation**: Record raw benchmark tables, variance analysis, and selection rationale in `docs/perf-decisions.md`.
4. **Clean codebase**: The production codebase must retain only the winning candidate. Benchmarks remain preserved for regression testing.

---

## 4. Git Workflow & Conventional Commits

- Commit messages must follow the Conventional Commits specification.
- Every performance or bottleneck resolution commit must include empirical benchmark metrics in the subject or summary.
- The commit message body must explicitly include:
  `[Reviewed-by: Agent-4 QA Gatekeeper]`
- Commit criteria: `go build ./...`, `go test -race ./...`, and benchmarks must all PASS before any commit.
