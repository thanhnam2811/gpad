# Architecture Decision Records (ADR) & Benchmark Performance Log

**Repository**: `gpad` (Gopad++)  
**Environment Baseline**:  
- CPU: Intel(R) Core(TM) i5-7500 CPU @ 3.40GHz (4 physical cores, 4 threads, 6MB L3 cache, AVX2, BMI2)
- Memory: 11 GiB Total RAM
- OS: Ubuntu 24.04.5 LTS (Noble Numbat), Linux 7.0.0-38-generic x86_64
- Go Runtime: `go1.27.2 linux/amd64`
- Testing Protocol: `go test -bench=. -benchmem -count=5`

---

## Index of Performance Decisions

| ADR ID | Domain | Primary Bottleneck | Winner Candidate | Key Metric | Reviewed By |
|---|---|---|---|---|---|
| `[PERF-001]` | `internal/fileio` | Large File Ingestion (10KB, 1MB, 50MB) | Pending Phase 2 | TBD | `[Agent-4]` |
| `[PERF-002]` | `internal/session` | Instant Notepad++ Zero-Save WAL Draft | Pending Phase 3 | TBD | `[Agent-4]` |
| `[PERF-003]` | `internal/search` | Ultra-Fast Search Crossover & Dispatch | Pending Phase 4 | TBD | `[Agent-4]` |
| `[PERF-004]` | `internal/ui` | GNOME 46 Native Render Frame Budget | Pending Phase 5 | TBD | `[Agent-4]` |

---

*(Records will be populated systematically by `[Agent-2]` and verified by `[Agent-4]` during each execution phase)*.
