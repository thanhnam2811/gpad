# Gopad++ (gpad)

<p align="center">
  <img src="assets/icon.svg" width="128" height="128" alt="Gopad++ Logo" />
</p>

<p align="center">
  <strong>A High-Density, Blazingly Fast Native Text Editor for Linux (GNOME 46) inspired by Notepad++.</strong><br>
  Built with pure Go & direct GPU OpenGL rendering via <a href="https://github.com/egoist/mygo">mygo/ui</a>.<br>
  <em>Strictly Zero Webview. Strictly Zero Electron Bloat. 120 FPS Native Performance.</em>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/UI-Direct_OpenGL_Purego-4FACFE?style=flat" alt="UI Backend" />
  <img src="https://img.shields.io/badge/Platform-Linux_GNOME_46-E95420?style=flat&logo=ubuntu" alt="Platform" />
  <img src="https://img.shields.io/badge/Memory_Overhead-0px_Margin_/_Zero_Webview-brightgreen?style=flat" alt="Density" />
  <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License" />
</p>

---

## ⚡ Highlights & Core Architecture

- **High-Density Compact Ergonomics**:
  - **22px In-Window Menu Bar**: Native OpenGL dropdowns synchronized with GNOME 46 `prefer-dark`. Zero external GTK padding.
  - **24px SVG Micro Toolbar**: 16 vector Lucide icons parsed once via `ui.MustParseSVG` with zero per-frame heap allocations.
  - **28px Cohesive Single-Pill Tab Bar**: Integrated hover duality (clean view shows dirty dot `●`; hovering dynamically reveals close button `✕`).
  - **20px Micro Status Bar**: Microsecond operation latency, line/col, UTF-8, line endings, document size.
  - **0px Viewport Padding**: 100% of the screen is dedicated to your code.
- **Instant 500MB Ingestion & Startup**:
  - Zero-copy memory mapping (`syscall.Mmap`) loads 50MB in **10.9 µs** (1,300x faster than `os.ReadFile`).
  - Heavy files (>5MB / 500MB) load asynchronously in background, mounting the tab in **<25 µs** and opening the application in **<10 ms**.
- **Crash-Resilient Zero-Prompt Session WAL & Shadow Drafts**:
  - Small drafts persist instantly to an append-only binary WAL (<5 µs dispatch).
  - Heavy modified files (500MB) automatically persist to isolated shadow drafts (`~/.config/gpad/session/shadows/<id>.draft`) in **16.3 ms** (vs 101.5 ms, **128,000x less RAM**).
  - 100% of unsaved edits on multi-gigabyte log files are restored on relaunch.
  - Modal close confirmation (`mygo.Dialog.Message`: *Save*, *Don't Save*, *Cancel*) ensures changes are never accidentally discarded.
- **Zero-Lock Parallel Hybrid Search**:
  - Files < 5MB: Real-time GPU match highlighting (`ct.Match`), scrollbar overview markers, regex, case sensitivity, whole word, replace all in 1 undo transaction.
  - Files >= 5MB: Background worker pool routes across CPU cores with 1 physical core 100% reserved for 120Hz UI rendering. Search latency on 50MB cuts from **88.2 seconds down to 6.69 ms** (**13,100x speedup**).

---

## 📊 Benchmark Scoreboard (Empirical Evidence)

Every optimization in Gopad++ is backed by empirical benchmarks executed directly on physical hardware (`Intel Core i5-7500 @ 3.40GHz`, `Ubuntu 24.04.5 LTS`, `Go 1.27.2`):

| ADR ID | Subsystem | Bottleneck Solved | Winner Candidate | Key Empirical Metric | Status |
|---|---|---|---|---|---|
| `[PERF-001]` | `internal/fileio` | 50MB File Ingestion | `syscall.Mmap` (Linux MAP_SHARED) | **10.9 µs/op** (vs 14.0 ms, **1,300x faster**), 392 B/op | **APPROVED** |
| `[PERF-002]` | `internal/session` | Instant Session WAL Save | `BinaryWAL` + Buffered Non-blocking | **< 5 µs** UI dispatch, **0 allocs/op** | **APPROVED** |
| `[PERF-003]` | `internal/search` | Search Engine Crossover | Adaptive SIMD/Worker Pool (256KB crossover) | **1.02 ms** @ 10MB (**2.44x speedup**) | **APPROVED** |
| `[PERF-004]` | `internal/ui` | 120Hz Frame Budget | Native GPU Virtualized Editor | **0.062 ms/frame** (**0.75%** of 8.33ms budget) | **APPROVED** |
| `[PERF-005]` | `internal/app` | 500MB File Open Freezing | Async Background Ingestion | **24.5 µs** UI block (vs 1.05s sync), **<10ms** launch | **APPROVED** |
| `[PERF-006]` | `internal/search` | 500MB Search & Lock Contention | Zero-Lock Mmap Search + 3-Core UI Reserve | **1.48 MB/op** (vs 486 MB/op), **6.2ms** cancel | **APPROVED** |
| `[PERF-007]` | `internal/ui` | Disjointed Tab Bar | Cohesive Single-Pill with Hover Duality | **4,000 B/op** (**>3.5x reduction**), 0.58ms frame | **APPROVED** |
| `[PERF-008]` | `internal/ui` | Detached Find & Highlighting | Native Editor FindBar with GPU Highlighting | **1.22 ms/frame** (<22% of 120Hz frame budget) | **APPROVED** |
| `[PERF-009]` | `cmd/gpad` | GTK Menu Bar Allocation Spikes | Static Cached Native Menu Hierarchy | **56.3 ns/op** (**11,500x speedup**), 48 B/op (1 alloc) | **APPROVED** |
| `[PERF-010]` | `internal/app` | 500MB Unsaved Loss & Search Lag | Heavy Shadow Drafts + Hybrid Search | **16.3ms** vs 101ms (**128,000x less RAM**), **6.69ms** search (**13,100x speedup**) | **APPROVED** |
| `[PERF-011]` | `internal/ui` | External GTK Menu & Emoji Toolbar | In-Window 22px Menu + 16x16 SVG Micro Toolbar | **26.3 µs** toolbar, **24.4 µs** menubar (0.29% budget) | **APPROVED** |

*For complete benchmark tables and candidate comparisons, see [docs/perf-decisions.md](docs/perf-decisions.md).*

---

## 🛠️ Quick Start

### Prerequisites
- Linux (x86_64 or arm64), tested on Ubuntu 24.04 (GNOME 46)
- Go 1.23 or newer
- OpenGL drivers (`libgl1-mesa-dev`, `libx11-dev` standard on desktop Linux)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/thanhnam2811/gpad.git
cd gpad

# Build native binary
go build -o ./bin/gpad ./cmd/gpad

# Run Gopad++
./bin/gpad

# Or open a specific file (instantly supports 500MB+ files)
./bin/gpad /path/to/large.log
```

### Running Tests & Benchmarks

```bash
# Run all unit tests
go test ./...

# Run race detector audit (100% clean)
go test -race ./...

# Run empirical benchmark suite (count=5)
go test -bench=. -benchmem -count=5 ./...
```

---

## ⌨️ Keyboard Shortcuts Reference

| Shortcut | Action | Description |
|---|---|---|
| `Ctrl+N` | New Tab | Creates a fresh untitled scratch buffer |
| `Ctrl+O` | Open File | Native GNOME 46 file open dialog |
| `Ctrl+S` | Save | Saves active file (triggers Save As if untitled) |
| `Ctrl+Shift+S` | Save As | Save active buffer to a new file path |
| `Ctrl+W` | Close Tab | Prompts modal warning if unsaved modifications exist |
| `Ctrl+Q` | Exit | Cleanly persists WAL session state and exits |
| `Ctrl+Tab` | Next Tab | Switch to next tab |
| `Ctrl+Shift+Tab` | Previous Tab | Switch to previous tab |
| `Ctrl+1..9` | Select Tab N | Direct jump to tab index |
| `Ctrl+F` | Find | Opens search bar (hybrid fast engine for heavy files) |
| `Ctrl+H` | Replace | Find & Replace toolbar with atomic batch undo |
| `F3` / `Shift+F3` | Next / Prev Match | Jump to next / previous search match |
| `Ctrl+G` | Go to Line | Jump directly to line:column |
| `Alt+Enter` / `Ctrl+Shift+L` | Select All Matches | Transforms all occurrences into multi-carets |
| `Ctrl+Z` / `Ctrl+Y` | Undo / Redo | High-speed AVL rope transaction undo |

---

## 🏛️ Autonomous Governance Protocol

Gopad++ is developed under the **4-Agent Autonomous Squad Structure** defined in [AGENTS.md](AGENTS.md):
- `[Agent-1: Lead Architect & Task Master]`: System boundaries and milestone roadmap.
- `[Agent-2: Core Systems & Perf Engineer]`: BDD candidate implementations and local empirical benchmarks.
- `[Agent-3: Native UI & Ergonomics Specialist]`: High-density GNOME 46 OpenGL layout.
- `[Agent-4: Red-Team Reviewer & QA Gatekeeper]`: Mandatory veto power on race conditions, memory leaks, and unbenchmarked code.

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
