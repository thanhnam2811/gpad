# Autonomous Engineering Squad Workflow Standard

**Repository**: `gpad` (Gopad++)  
**Standard Version**: `v1.1`  
**Governing Document**: `AGENTS.md`

This document records the exact operational standard and lifecycle protocol used by the autonomous squad to execute features, performance optimizations, and refactors.

---

## 1. Squad Architecture & Role Responsibilities

Every task or milestone is executed strictly through 4 specialized agent roles:

```
[Agent-1: Lead Architect]
         │  (Architecture decomposition, interface isolation)
         ▼
[Agent-2: Core Systems & Perf Engineer] / [Agent-3: Native UI Specialist]
         │  (Algorithmic candidates, BDD benchmarks, OpenGL ergonomics)
         ▼
[Agent-4: Red-Team Reviewer & QA Gatekeeper]
         │  (Race detector, leak audit, empirical validation)
         ▼
[Atomic Conventional Commit & Progress Report]
```

1. **`[Agent-1: Lead Architect & Task Master]`**:
   - Decomposes feature requests into modular, decoupled tasks.
   - Preserves package boundaries (e.g. `internal/instance`, `internal/keybind`, `internal/buffer`, `internal/ui`).
   - Guarantees zero circular dependencies.

2. **`[Agent-2: Core Systems & Perf Engineer]`**:
   - Develops algorithmic solutions for low-level systems (I/O, IPC, search, buffer).
   - Writes empirical benchmarks inside `*_bench_test.go`.
   - Analyzes latency (`ns/op`), memory footprint (`B/op`), and allocations (`allocs/op`).

3. **`[Agent-3: Native UI & Ergonomics Specialist]`**:
   - Implements high-density UI components (`mygo/ui` GPU OpenGL canvas).
   - Strictly avoids webview/bloat margins (0px viewport margin, 22px menubar, 24px toolbar, 28px tab bar).
   - Ensures frame budgets remain below **8.33 ms/frame** (120Hz display target).

4. **`[Agent-4: Red-Team Reviewer & QA Gatekeeper]`**:
   - Audits all code changes before any Git commit.
   - **Veto Conditions**: Rejects changes if:
     1. Benchmarks are absent or speculative.
     2. `go test -race ./...` exits non-zero.
     3. Memory leaks or unclosed resources exist.
     4. Ergonomics violate High-Density standards.

---

## 2. Phased Task Execution Lifecycle

For every task in a milestone, the squad follows this 5-step loop:

### Step 1: Task Scoping & Decomposition
- Break the requirement into atomic tasks where each task touches one cohesive domain.
- State clearly what issue is being solved and why the previous behavior existed.

### Step 2: Implementation & Decoupled Architecture
- Implement minimal, zero-copy, zero-allocation algorithms where possible.
- Avoid global mutable state; prefer explicit struct dependencies.

### Step 3: Empirical Benchmarking & Race Detection
- Execute 5-count empirical benchmark directly on host hardware:
  ```bash
  go test -bench=. -benchmem -count=5 ./internal/<package>
  ```
- Run the Go race detector across the entire repository:
  ```bash
  go test -race ./...
  ```

### Step 4: Atomic Conventional Commit
- Each task produces exactly one commit.
- The commit subject includes the key empirical metric:
  `feat(<scope>): <summary> [<metric>, PERF-XXX]`
- The body contains audit sign-off:
  `[Reviewed-by: Agent-4 QA Gatekeeper]`

### Step 5: Progress Reporting
- Report concisely to the user/supervisor upon completion of each task:
  * What was implemented.
  * Measured benchmark metrics.
  * Race detector status.
  * Current Git commit hash.

---

## 3. Documentation Governance

- All architectural decisions and comparative benchmark tables must be appended to `docs/perf-decisions.md` under a unique `[PERF-XXX]` identifier.
- The root index table in `docs/perf-decisions.md` must be kept up-to-date with the winning candidate, key metric, and QA sign-off.
