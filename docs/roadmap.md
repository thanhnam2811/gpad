# Gopad++ (gpad) — Architectural Roadmap & Execution Tracker

**Managed by**: `[Agent-1: Lead Architect & Task Master]`

## Project Vision
A blazingly fast, lightweight, high-density native text editor for Linux (GNOME 46) inspired by Notepad++. Built in pure Go using `github.com/egoist/mygo/ui` (direct GPU OpenGL rendering via purego, strictly zero webview/Electron bloat).

---

## Phase Breakdown & Milestone Status

### Milestone 0: Constitution & Infrastructure Baseline [COMPLETED]
- [x] Git repository initialization (`main` branch) and `.gitignore` setup.
- [x] Host hardware & GNOME 46 baseline recorded (Intel i5-7500, 4 cores, Ubuntu 24.04.5, Go 1.27.2).
- [x] Multi-Agent Constitution (`AGENTS.md`) and symlinks (`CLAUDE.md`, `GEMINI.md`).
- [x] Architecture records initialized (`docs/roadmap.md`, `docs/perf-decisions.md`).

### Phase 1: Native Skeleton & Decoupled Architecture [COMPLETED]
- [x] Initialize `go.mod` with module `gpad` and dependency `github.com/egoist/mygo`.
- [x] Establish package boundaries:
  * `cmd/gpad`: Application entrypoint.
  * `internal/app`: Lifecycle coordinator and state orchestration.
  * `internal/fileio`: High-throughput, zero-copy file loader.
  * `internal/session`: Instant crash-resilient session draft persistence.
  * `internal/search`: SIMD/parallel adaptive search engine.
  * `internal/ui`: High-density GNOME 46 native compact presentation layer.
- [x] Verification by `[Agent-4]` and Phase 1 Git Commit.

### Phase 2: Bottleneck #1 — Ultra-Fast File Loader (`internal/fileio`) [COMPLETED]
- [x] Implement benchmark suite for 10KB, 1MB, 50MB across:
  * Candidate A: `os.ReadFile` (standard buffer allocation).
  * Candidate B: `syscall.Mmap` (`PROT_READ`, `MAP_SHARED` zero-copy paging).
  * Candidate C: Chunked Reader (`bufio.Reader` with `sync.Pool`).
- [x] Execute `go test -bench=. -benchmem -count=5` on local host.
- [x] Document ADR `[PERF-001]` in `docs/perf-decisions.md`.
- [x] Integrate winner into `internal/fileio` with robust error and unmap safety.
- [x] Verification by `[Agent-4]` and Phase 2 Git Commit.

### Phase 3: Bottleneck #2 — Zero-Prompt Instant Draft Save (`internal/session`) [COMPLETED]
- [x] Implement benchmark suite for 512KB and 5MB payloads across:
  * Candidate A: Temp-file atomic rename (`os.Rename`).
  * Candidate B: Append-only Binary WAL with pre-allocated memory mapping.
  * Candidate C: Pure-Go SQLite in WAL mode (`PRAGMA synchronous=NORMAL;`).
- [x] Execute `go test -bench=. -benchmem -count=5` on NVMe/SSD host drive.
- [x] Target: UI thread block latency < 0.5ms (Achieved < 5µs via async buffered flusher).
- [x] Document ADR `[PERF-002]` in `docs/perf-decisions.md`.
- [x] Verification of crash recovery by `[Agent-4]` and Phase 3 Git Commit.

### Phase 4: Bottleneck #3 — Adaptive Search & Replace (`internal/search`) [COMPLETED]
- [x] Implement benchmark suite on 10MB text buffer across:
  * Candidate A: Sequential `bytes.Index` (SIMD Go stdlib).
  * Candidate B: Multi-threaded Worker Pool chunk scanner (4 host CPU cores).
  * Candidate C: Precompiled standard `regexp`.
- [x] Determine crossover point (256KB) and implement Adaptive Search Dispatcher.
- [x] Document ADR `[PERF-003]` in `docs/perf-decisions.md`.
- [x] Race detection audit (`go test -race ./...`), `[Agent-4]` sign-off and Phase 4 Git Commit.

### Phase 5: Compact Native UI on GNOME 46 (`internal/ui`) [COMPLETED]
- [x] Compact Tab Bar (28px height, dirty indicator dots, Ctrl+N, Ctrl+W, Ctrl+Tab).
- [x] Main Code Editor integration via `mygo/ui/editor` with full viewport occupation (0px margin).
- [x] Micro Status Bar (20px height: line, column, UTF-8, LF, document size, microsecond op latency).
- [x] Inline Quick Find Bar (`Ctrl+F`) with microsecond search latency metric.
- [x] Headless frame cost & allocation benchmark `[PERF-004]` (<8.3ms/frame, 0.061ms steady frame latency).
- [x] Full build `./bin/gpad`, tag `v0.1.0-alpha`, final `[Agent-4]` approval.

### Phase 6: 500MB Heavy File Ingestion & Instant Session Startup (`internal/app`, `internal/buffer`) [COMPLETED]
- [x] Profile 500MB log file loading (2.45M lines): identified 1.05s UI thread freeze in `editor.NewBuffer`.
- [x] Benchmark Candidate A (Sync) vs Candidate B (Async Background Ingestion) `[PERF-005]`.
- [x] Implement async ingestion with non-blocking viewport placeholder `[Loading "file" (500.0 MB)...]`.
- [x] Instant app startup (<10ms): restored heavy tabs populate metadata instantly, deferred rope build.
- [x] Avoid duplicate tabs when opening existing file via command line.
- [x] Verification by `[Agent-4]` and Phase 6 sign-off.

### Phase 7: Zero-Lock Debounced Search & UI Core Reservation (`internal/search`, `internal/ui`) [COMPLETED]
- [x] Benchmark Candidate A (Unbounded 4-core) vs Candidate B (Bounded 10k, 3 workers UI reserve, context cancellation) `[PERF-006]`.
- [x] 328x heap memory reduction (486 MB -> 1.48 MB) and 18% latency improvement on 500MB buffer.
- [x] Zero-lock search: search directly over memory-mapped pages for clean files (0 locks on `editor.Buffer`).
- [x] 300ms UI debounce with 6.2ms in-flight cancellation.
- [x] Remove per-frame `CheckAutoSave()` from `Render()` to maintain 0 lock contention.
- [x] Race detector audit (`go test -race ./...`), `[Agent-4]` approval sign-off.

### Phase 8: Cohesive Compact Tab Bar & Ergonomics (`internal/app`, `internal/ui`) [COMPLETED]
- [x] Refactor disconnected two-button tab layout into cohesive single-pill design (`ui.ButtonBase`, 26px height, 4px radius).
- [x] Implement hover duality slot: dirty dot (`●`) normally displayed; dynamically morphs into close button (`✕`) on hover.
- [x] Integrate tab context menu (`Close Tab`, `Close Other Tabs`, `Close Tabs to the Right`, `Copy Full Path`).
- [x] Add tab keyboard shortcuts (`Ctrl+Tab`, `Ctrl+Shift+Tab`, `Ctrl+1..9`) and double-click to open new tab.
- [x] Benchmark Candidate A vs Candidate B `[PERF-007]` (>3.5x heap reduction, 30% fewer allocations).
- [x] Audit by `[Agent-4]` and Phase 8 Git Commit.

### Phase 9: Unified Native Search & Replace Architecture (`internal/ui`) [COMPLETED]
- [x] Connect `Ctrl+F`, `Ctrl+H`, `F3`, `Shift+F3`, `Ctrl+G` to native `mygo/ui/editor` command runner.
- [x] GPU match highlighting (`ct.Match`), scrollbar overview markers, and auto-focus search input.
- [x] Support regex, match whole word, match case, replace single, replace all in 1 undo step.
- [x] Multi-caret selection across all matches (`Alt+Enter` / `Ctrl+Shift+L`).
- [x] Benchmark active search frame cost and document ADR `[PERF-008]`.
- [x] Audit by `[Agent-4]` and Phase 9 Git Commit.

### Phase 10: Top Menu Bar Hierarchy & Native File Dialogs (`cmd/gpad`, `internal/ui`) [COMPLETED]
- [x] Full native GTK Menu Bar hierarchy (`File`, `Edit`, `Search`, `View`, `Document`, `Help`).
- [x] Native file open dialog (`Ctrl+O`) and save as dialog (`Ctrl+Shift+S`) via `mygo.Dialog`.
- [x] Custom window icon embedding (`w.SetIcon`) and GNOME 46 `.desktop` installer.
- [x] Benchmark Candidate A vs Candidate B `[PERF-009]` (11,500x speedup, 56.3 ns/op, 1 alloc).
- [x] Final audit by `[Agent-4]`, Git Commit, and binary verification.


