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

### Phase 1: Native Skeleton & Decoupled Architecture [IN PROGRESS]
- [ ] Initialize `go.mod` with module `gpad` and dependency `github.com/egoist/mygo`.
- [ ] Establish package boundaries:
  * `cmd/gpad`: Application entrypoint.
  * `internal/app`: Lifecycle coordinator and state orchestration.
  * `internal/fileio`: High-throughput, zero-copy file loader.
  * `internal/session`: Instant crash-resilient session draft persistence.
  * `internal/search`: SIMD/parallel adaptive search engine.
  * `internal/ui`: High-density GNOME 46 native compact presentation layer.
- [ ] Verification by `[Agent-4]` and Phase 1 Git Commit.

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

### Phase 4: Bottleneck #3 — Adaptive Search & Replace (`internal/search`)
- [ ] Implement benchmark suite on 10MB text buffer across:
  * Candidate A: Sequential `bytes.Index` (SIMD Go stdlib).
  * Candidate B: Multi-threaded Worker Pool chunk scanner (4 host CPU cores).
  * Candidate C: Precompiled standard `regexp`.
- [ ] Determine crossover point and implement Adaptive Search Dispatcher.
- [ ] Document ADR `[PERF-003]` in `docs/perf-decisions.md`.
- [ ] Race detection audit (`go test -race ./...`), `[Agent-4]` sign-off and Phase 4 Git Commit.

### Phase 5: Compact Native UI on GNOME 46 (`internal/ui`)
- [ ] Compact Tab Bar (28px height, dirty indicator dots, Ctrl+N, Ctrl+W, Ctrl+Tab).
- [ ] Main Code Editor integration via `mygo/ui/editor` with full viewport occupation (0px margin).
- [ ] Micro Status Bar (20px height: line, column, UTF-8, LF, document size, microsecond op latency).
- [ ] Inline Quick Find Bar (`Ctrl+F`) with microsecond search latency metric.
- [ ] Headless frame cost & allocation benchmark `[PERF-004]` (<8.3ms/frame, 0 allocs/frame in steady scroll).
- [ ] Full build `./bin/gpad`, tag `v0.1.0-alpha`, final `[Agent-4]` approval.
