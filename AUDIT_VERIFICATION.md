# GPAD reliability audit verification

Date: 2026-10-20. Baseline: `410a27c`; working branch: `fix/audit-reliability`.
The initial working tree was clean. No commit, push, deployment, release, real document,
or existing GPAD session was used by this investigation. Tests use temporary directories,
synthetic text, and headless UI resources. The EXDEV test uses its own temporary directory
on `/dev/shm`; its file-size fault injection runs in a child process.

## Findings

Baseline references below refer to `git show 410a27c:<path>`; linked references refer to
the patched working tree. A passing test is not proof of real power-loss durability.

| Finding | Verdict | Evidence | Fix | Regression Test | Status |
|---|---|---|---|---|---|
| 1. SaveFile EXDEV, atomicity, permissions, symlinks, durability | Confirmed; existing write/short-write/file-fsync checks were already present | Baseline `internal/fileio/fileio.go:90` stages in global TMPDIR, `:121` renames without directory fsync. Actual baseline reproduction returns `invalid cross-device link`; mode/link regression reports “symlink replaced.” | [SaveFile](internal/fileio/fileio.go#L91): stage beside destination, retain ordinary mode bits, resolve existing symlink targets, reject dangling links/nonregular targets, clean staging files, fsync file and destination directory. | `TestSavePreservesModeAndSymlink`, `TestAuditSaveAcrossTempFilesystem`, `TestAuditSaveFileWriteFailure`, existing save/load tests | Fixed; PASS |
| 2. Ignored shadow writes, partial writes, unsaved-data loss | Confirmed | Baseline `internal/app/app.go:515,532` ignores mkdir/WriteFile errors; stable basename draft paths overwrite prior checkpoints and collide. Failing baseline regression reports “shadow failure reported success.” | [SaveSession](internal/app/app.go#L511): checked private immutable draft generations, durable shadow before WAL, cleanup only prior published references after successful WAL commit. Failed publication preserves old generation. Unknown artifacts retained. | `TestAuditShadowFailurePreservesCheckpoint`, `TestAuditImmutablePrivateShadowGenerations`, existing heavy-draft round trip | Fixed; PASS |
| 3. Socket removal/concurrent startup | Confirmed | Baseline `internal/instance/instance.go:109` unlinks unconditionally; second-owner regression accepts two listeners. Baseline `cmd/gpad/main.go:30` opens session before acquiring listener and continues on listener failure. | [StartListener](internal/instance/instance.go#L103): stable-inode flock, live-socket probe, conservative stale reclaim, non-socket preservation; listener ownership before session creation in `cmd/gpad/main.go`. Shutdown retains flock until session closes; queued acknowledged opens are drained. | `TestListenerRejectsSecondOwner`, `TestListenerConcurrentStartup` (12 simultaneous callers), `TestListenerPreservesRegularFileAndReclaimsStaleSocket`, `TestListenerRetainsOwnershipDuringSessionShutdown`, `TestRequestQueueStartupAndConcurrentDelivery` | Fixed; PASS |
| 4. unsafe.String use-after-unmap | False Positive for the reported ownership hypothesis | `editor.NewBuffer` at `third_party/mygo/ui/editor/buffer.go:122` calls rope.New; `third_party/mygo/internal/rope/rope.go:94` clones every leaf with strings.Clone. App finishes buffer construction before payload.Close. Overwriting/truncating the source after OpenFile leaves editor content intact. | Ownership implementation retained; explanatory comment added to common loader. Separate concurrent external-truncation mmap limitation remains below. | `TestAuditMappedEditorOwnsText` | Verified; PASS |
| 5. WAL fsync, compaction errors, recovery | Confirmed | Baseline `internal/session/session.go:139,140,146` ignores compaction Sync/Close/rename failure; `:173–179` append lacks fsync. Replay stops at invalid tail but later appends behind it become invisible. Baseline recovery test returns old ActiveIndex 1 after saving 2. Corrupt lengths allocate before checking remaining bytes. | [WAL append](internal/session/session.go#L183): checked frame, rollback on write failure, terminal disable on failed rollback, fsync before success. Durable replacement and reopen pathname on compaction outcomes. [Replay](internal/session/session.go#L228): bound lengths by remaining file, propagate real read errors, repair invalid tails at startup. | `TestAppendAfterCorruptTail`, `TestCompactionFailureKeepsRecoverableWAL`, `TestCorruptLengthAndPartialHeaderRecovery`, `TestAuditFailedRollbackDisablesWAL`, existing CRC/truncated-tail tests, `FuzzWALCorruptTail` | Fixed; PASS |
| 6. Search cancellation/stale generations | Confirmed | Baseline `internal/app/app.go:715–724`: stopped timer callback can still start and replace newer cancellation; `:755–763` cancellation check and result publication are separate. No query-generation/document-version check. | [TriggerSearch](internal/app/app.go#L724) and generation worker serialize generation validation, cancellation installation and publication; reject changed document/path/version; tab transitions clear/restart search. Save As/path reads share app lock. | `TestAuditSearchRejectsStaleGeneration` uses blocked engine and explicit stale callback, `TestAuditSearchClearsResultsOnTabSwitch`, existing debounce test, race suite | Fixed; PASS |
| 7. Autosave/session shutdown | Partially Confirmed: missing lifecycle join; no raw WAL close/write data race due to Manager.mu | Baseline `internal/app/app.go:106–131` closes stop channel without joining autosave; concurrent Stop callers can both close it; persistence failures ignored. Background snapshot also reads UI-owned editor.State. | [Stop](internal/app/app.go#L124): sync.Once shutdown, join autosave, invalidate pending searches, final snapshot, then close manager. Cached caret avoids background editor.State reads. IPC lock retained through this lifecycle. | `TestAuditConcurrentStop`, `TestAuditAutosaveDoesNotReadUIState`, `TestListenerRetainsOwnershipDuringSessionShutdown`, race suite | Fixed; PASS |
| 8. Async loading stuck on error | Confirmed | Baseline `internal/app/app.go:384–387` exits worker on error without terminal publication; restore branches similarly omit errors. Document.ReadyCh was never closed. Saving a loading dirty draft could write empty contents over its original. | [loadDocument](internal/app/app.go#L904) publishes success/failure, document methods synchronize publication and release waiters, UI renders failure, all save paths including CloseTab reject loading/error documents. | `TestAuditAsyncLoadFailureCompletes`, `TestAuditDocumentReadySignal`, `TestFailedAsyncLoadRendersTerminalError`, `TestAuditCloseCannotSaveUnloadedDraft`, existing large-file async restore | Fixed; PASS |
| 9. Session content permissions | Confirmed | Baseline `internal/session/session.go:49,54` uses directory 0755/WAL 0644; app shadow directory/file 0755/0644 at `:515,532`. Contents can be read by other local users under usual home-directory traversal permissions. | [NewManager](internal/session/session.go#L48) tightens dedicated session directory to 0700 and WAL to 0600, including existing paths. SaveSession uses 0700 shadow directory and new 0600 drafts. Parent protection covers legacy artifacts even if restore fails. | `TestAuditLegacyDirectoryPrivate`, `TestAppendAfterCorruptTail` checks WAL mode, `TestAuditImmutablePrivateShadowGenerations` checks draft and directory modes | Fixed; PASS |
| 10. SaveSession errors dropped by workflows | Confirmed | Baseline app drops errors at NewTab, CloseTab, tab navigation, file open/save, autosave, Stop; constructor ignores NewManager error. Error-returning file workflows return nil after failing session save (`:469,493`). | [recordSessionError](internal/app/app.go#L873) records/logs failures; compact status displays SessionError. File workflows propagate snapshot errors; void workflows retain observable error and autosave retry. Recovery/manager failure blocks replacing prior checkpoint. | `TestAuditWorkflowReportsSessionFailure`, `TestAuditMissingDraftPreservesWAL`, `TestAuditManagerCreationFailureReported`, existing save workflow tests | Fixed; PASS |

## Closely related defects fixed

- Initial nonempty untitled content was treated as clean and omitted from recovery. A dirty
  document emptied by its user was mistaken for an omitted clean file and loaded old disk
  text. Both baseline failures are reproduced by `TestAuditUntitledAndEmptyDirtyRecovery`.
- Missing/unreadable recovery drafts silently fell back to disk/empty text, allowing the
  next save to destroy the recovery reference. Recovery now reports the error and blocks
  replacement snapshots. `TestAuditMissingDraftPreservesWAL` checks byte-for-byte WAL retention.
- Closing or cleaning a tab deleted its shadow before committing replacement metadata.
  Draft retirement now follows successful checkpoint publication.
- Loader publication raced unlocked document access; background session snapshots queried
  UI-owned editor.State. Document accessors synchronize publication, and the UI updates a
  cached caret used by autosave. UI error/latency reads use synchronized accessors.
- Repeated listener Close previously panicked; accepted handlers were not joined. Shutdown
  is idempotent, joins handlers, and retains exclusive session ownership until persistence closes.

## Validation and evidence

Reproductions were run before fixes where interfaces permitted: shadow failure, untitled
and empty-dirty recovery, readiness signal, mode/symlink replacement, corrupt-tail append,
second listener ownership, and actual EXDEV all failed on baseline production code.
The baseline EXDEV reproduction was additionally rerun in an isolated `/tmp` module after
adding the test; it returned `rename /dev/shm/... /tmp/...: invalid cross-device link`.
Other concurrency/lifecycle cases use deterministic forced interleavings and regressions
against the new lifecycle interfaces; no baseline runtime result is claimed for those.

| Check | Result |
|---|---|
| `go test ./...` | PASS |
| `go test -race ./...` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| Targeted audit tests and IPC/WAL suites | PASS |
| `go test ./internal/session -run '^$' -fuzz FuzzWALCorruptTail -fuzztime=5s -parallel=2` | PASS; 125 executions plus three seeds, about six seconds |
| Linux write-failure injection | PASS: child-only RLIMIT_FSIZE causes save failure; original bytes retained and staging file removed |
| EXDEV regression | PASS on distinct `/dev/shm` and destination device IDs; no skip on this host |
| Headless UI loading/error integration and concurrent IPC queue test | PASS; no native display opened |
| `git diff --check` | PASS |

Five-run baseline/current durable-save and WAL benchmarks, raw samples and variance are in
[docs/perf-decisions.md](docs/perf-decisions.md). BenchmarkAudit harnesses remain in source.
Host reports Go 1.27.2 linux/amd64 and Intel Core i5-7500. The production comparison used
`-bench BenchmarkAudit -benchmem -count=5 -benchtime=200ms`:

| Path | Before | After | Interpretation |
|---|---|---|---|
| SaveFile | 6.19–27.41 ms/op; about 521 B/op, 9 allocs/op | 11.26–13.36 ms/op; about 2495 B/op, 33 allocs/op | Storage variance is large; no throughput improvement claimed |
| WAL snapshot | 0.200–0.217 ms/op; about 28 KB/op, 5 allocs/op | 6.12–9.98 ms/op; about 57 KB/op, 6 allocs/op | Added durable fsync is deliberately more expensive |

Additional relevant five-run benchmark smoke command used `-benchtime=1x`: durable save,
WAL, IPC and shadow-persistence candidates completed. The existing
`BenchmarkApp_HybridSearch/Candidate_A_FullStringExtraction_Search` was stopped after
roughly 90 seconds without finishing its first sample: its repeated whole-suffix
string-to-byte conversion makes this harness resource-intensive. No result is claimed;
production search correctness/race tests passed. It was not weakened or modified.

Separate UI render benchmarks used `-bench BenchmarkUI_Render -benchmem -count=5
-benchtime=100ms`: steady 0.127–0.308 ms/frame (58 allocs), typing 0.743–2.076 ms/frame,
large document 0.333–0.656 ms/frame (84 allocs). Tab switching, which includes synchronous
durable session persistence, measured 6.93–20.82 ms/op and sometimes exceeds 8.3 ms.
These runs overlapped other validation activity and are noisy; no precise before/after UI
ratio is claimed. Zero allocations/frame was already unmet and remains unmet. Durability
was chosen over latency; this performance limitation is explicit rather than hidden.

## Exact changed files

Production:

- `cmd/gpad/main.go`
- `internal/app/app.go`
- `internal/buffer/buffer.go`
- `internal/fileio/fileio.go`
- `internal/instance/instance.go`
- `internal/session/session.go`
- `internal/ui/ui.go`
- `internal/ui/menu.go`
- `internal/ui/menubar.go`
- `internal/ui/toolbar.go`

Tests/benchmarks:

- `cmd/gpad/main_test.go`
- `internal/app/audit_test.go`
- `internal/fileio/fileio_test.go`
- `internal/fileio/audit_failure_test.go`
- `internal/fileio/audit_bench_test.go`
- `internal/instance/instance_test.go`
- `internal/session/session_test.go`
- `internal/session/audit_bench_test.go`
- `internal/session/audit_fuzz_test.go`
- `internal/ui/audit_test.go`

Documentation: `AUDIT_VERIFICATION.md`, `docs/roadmap.md`, `docs/perf-decisions.md`.
No vendored library changes or dependency additions.

## Remaining risks and limits

- The mmap ownership hypothesis is false, but concurrent **external truncation while mapped
  bytes are being copied/searched** can still cause Linux SIGBUS. This pre-existing limitation
  is separate from use-after-unmap; this patch does not claim exhaustive memory safety for
  externally mutating files. Clean-file search also still searches current disk bytes, which
  can differ from the loaded editor when another program changes a file.
- Atomic replacement retains ordinary Unix permission bits; it changes the inode and does
  not preserve ownership, ACLs, extended attributes, special mode bits, or hard-link identity.
  Symlink resolution is suitable for ordinary user files, not a hostile concurrently swapped
  directory tree. A post-rename directory-fsync error means content may already be replaced
  with durability uncertain; the error is returned.
- Successful fsync is the software durability boundary. Actual power cuts, disk/controller
  faults, file/dir fsync failures, close failures and kernel EIO were not physically injected.
  Error propagation and transaction ordering were reviewed; write failure, compaction
  pre-rename failure, read-only WAL rollback failure and corrupt/truncated files were tested.
- Conservative recovery preserves unknown/orphan draft files after failed checkpoint attempts.
  Repeated failures can accumulate private disk artifacts; these are retained for recovery.
  WAL repair keeps the valid prefix; it does not salvage later frames past corruption.
- SessionError is a sticky record of the most recent failure, even after a retry succeeds.
  Large dirty drafts still require full content extraction and durable disk writes; UI paths
  that synchronously persist sessions can miss the 120 Hz target.
- Native GNOME Wayland focus, real GPU frames and compositor integration were not exercised.
  Headless UI coverage passed. The active Wayland session was never restarted or disturbed.
- No public API promises were added for arbitrary concurrent UI mutations. editor.State remains
  UI-owned; callers must use the synchronized app/document methods for background operations.

## Final release-readiness assessment

All ten audit hypotheses have evidence-based verdicts. Confirmed audit defects and the
closely related recovery/shutdown defects above are fixed with regression coverage.
The patch passes build, tests, race detector and vet and is suitable for code review.
Unqualified release readiness is **not claimed**: external mmap truncation, inode metadata
semantics, synchronous persistence latency and native Wayland integration require the
remaining limitations above to be accepted or addressed before release.

Four-role process: Agent-1 architecture handoff → Agent-2 core implementation/benchmarks →
Agent-3 native UI/startup integration → Agent-4 independent full-diff review. Agent-4 found
and required fixes for WAL error handling, shutdown lock lifetime, Save As/search path
synchronization, legacy permissions and saving unloaded drafts. Final sign-off: **APPROVED — Agent-4 QA Gatekeeper**, after independent complete-diff/ADR/report review, targeted race tests, build, vet and diff checks, plus verification of final full-suite passing logs. At audit completion, changes remained uncommitted; publication requires separate user authorization.
