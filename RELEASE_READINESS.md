# Release readiness — Gopad++ 0.1.0-rc.1

**Disposition: release candidate, not a stable production release.**
Validation completed on 2026-10-23, following the implementation/review begun on 2026-10-20.
The initial polish branch was based on `0af66fb`; the initial working tree was clean and
all prior reliability commits remain intact. The maintainer subsequently authorized commit,
fast-forward publication to `main` and the first GitHub pre-release, `v0.1.0-rc.1`.

The original local validation below used a source tree explicitly stamped `-dirty`. Published
artifacts must be rebuilt from the clean tagged commit, independently reproduced and verified;
local preview archives must not be uploaded. Their fixed build timestamp is the source commit
timestamp, not the screenshot/test date. A checksum proves byte equality against its sidecar,
not publisher identity. Artifacts are unsigned. Native Wayland/GPU and the stable-release gates
below remain open; publication as a pre-release does not claim those gates have passed.

## What changed

- Retained the G/++ identity and simplified glow/decorations into a scalable cyan/mint mark.
  SVG sources, eight PNG sizes, symbolic icon, social artwork and a light/dark review sheet
  are documented in [BRANDING.md](BRANDING.md). Embedded and packaging copies are checked.
- Replaced stale README performance promises with measured limits and practical build,
  installation, usage and contribution instructions. Added changelog, security guidance,
  issue/PR templates, versioned candidate notes and third-party icon notices.
- Unified CLI/About build identity in `internal/buildinfo`. `--help` and `--version` avoid
  session/IPC initialization. Errors exit nonzero. Launcher integration is an explicit
  `--install-desktop` action, respects XDG_DATA_HOME and quotes executable paths.
- GNOME light/default preferences now select light mode. About uses the canonical icon and
  actual platform. Advertised global Open/Save As/Quit shortcuts work with menus closed.
  Save/discard confirmation validates document identity; failed saves retain the document.
- Added desktop/AppStream metadata, a prefix/DESTDIR installer, deterministic archive tooling,
  checksums, dependency/license inventory and an isolated verifier. Replacing a running binary
  stages a new inode; a directory named `bin/gpad` is rejected rather than treated as a folder.
- CI reads Go from go.mod, runs build/test/race/vet/dependency and metadata/asset/package checks.
  Actions are pinned to verified commit SHAs, permissions are read-only, checkout credentials
  are not persisted, and jobs have timeouts. Manual artifact builds never create a release.

No new Go dependency or vendored framework modification was introduced. Audited core files
under `internal/{app,buffer,fileio,instance,search,session}` have no diff from the baseline.
Normal single-instance ownership and joined autosave/shutdown ordering remain unchanged.

## Evidence and executed checks

Host: Ubuntu 24.04.5 LTS, Linux 7.0.0-38-generic, Intel Core i5-7500, Go 1.27.2 linux/amd64.
Tests use temporary files/sockets/sessions. Actual native capture uses a private display and bus.

| Check | Actual result |
|---|---|
| `go build ./...` | PASS |
| `go test -count=1 ./...` | PASS, fresh uncached execution including prior audit regressions |
| `go test -race -count=1 ./...` | PASS, fresh uncached execution |
| `go vet ./...` | PASS |
| `go mod verify` | PASS |
| `python3 scripts/check-assets.py` | PASS: PNG dimensions, embedded copy equality, SVG resources, desktop identity |
| `python3 scripts/test-release.py` | PASS: checksum damage, absolute/parent paths, symlink archive, directory install target, running-binary replacement |
| `bash -n scripts/{install,release,verify-release}.sh` | PASS |
| `desktop-file-validate packaging/gpad.desktop` | PASS |
| `appstreamcli validate --no-net packaging/io.github.thanhnam2811.gpad.metainfo.xml` | PASS; network screenshot checks deliberately not claimed |
| Child CLI/desktop/UI regressions | PASS, including no session writes for info/error modes, escaped paths, light theme, stale modal identity and save failure retention |
| Native GUI smoke, dark and light | PASS; screenshots and per-run evidence below |
| Linux amd64 candidate verifier | PASS: checksum, safe archive paths/types, ELF target, isolated version probe, staged install and executable comparison |
| Repeated candidate builds | PASS: both archives and checksum manifests are byte-identical under `cmp`; final digest is outside this bundled report in `dist/SHA256SUMS` |
| Linux arm64 | CGO-disabled Linux arm64 cross-build PASS, ELF AArch64 confirmed; **not executed** |
| `git diff --check` | PASS |

The final archive build/verification commands, also used for the deliverable, are:

```sh
ALLOW_DIRTY=1 VERSION=0.1.0-rc.1 OUTPUT_DIR=dist scripts/release.sh
ALLOW_DIRTY=1 VERSION=0.1.0-rc.1 OUTPUT_DIR=/tmp/gpad-polish-repeat scripts/release.sh
cmp dist/SHA256SUMS /tmp/gpad-polish-repeat/SHA256SUMS
scripts/verify-release.sh dist/gpad-0.1.0-rc.1-linux-amd64.tar.gz
```

The same source inputs, Go version, architecture, environment and fixed SOURCE_DATE_EPOCH
are required. Archive order, ownership, modes, mtime and gzip headers are normalized.
A preliminary two-independent-source-copy comparison is recorded in [PERF-017](docs/perf-decisions.md).
The final delivery uses repeat builds of the final working tree, not a claim that dirty trees
are distributable clean tagged releases. Commit/clean-checkout reproducibility remains a
publication gate. No checksum is embedded into its own archive's report.

## Actual native GUI and visual QA

The packaged amd64 executable was launched with:

```sh
python3 scripts/gui-smoke.py /path/to/extracted/bin/gpad /tmp/gpad-capture-dark
python3 scripts/gui-smoke.py /path/to/extracted/bin/gpad /tmp/gpad-capture-light --theme light
```

The helper allocates Xvfb via displayfd, whitelists environment variables, creates its own
HOME/XDG/runtime paths and D-Bus bus with no host service activation directories, and forces
software OpenGL. It uses XTEST directly, never a clipboard. It does not connect to the user's
Wayland display/session bus. Only its own child process is killed for crash recovery.
Earlier harness failures exposed coordinate/timing assumptions; window-origin and mapped-state
checks now precede input. Those failed attempts are not counted as passing GUI runs.

Both final runs verified startup, a 128×128 native window icon, `WM_CLASS=gpad`, keyboard editing
and saving a synthetic file, Find, the actual File menu and About dialog, secondary-process
forwarding, autosaved untitled content, recovery after SIGKILL, and successful Ctrl+Q shutdown.
Captured recovery windows visibly contain `recoveryprivatedraft`; original content is synthetic.
`gui-dark.json` and `gui-light.json` record the candidate executable hash/version and outcomes.
No screenshots were fabricated, drawn over or taken from a personal workspace.

- [Dark editor](assets/screenshots/editor-dark.png) / [Light editor](assets/screenshots/editor-light.png)
- [Find](assets/screenshots/find-dark.png) / [File menu](assets/screenshots/menu-dark.png)
- [Dark About](assets/screenshots/about-dark.png) / [Light About](assets/screenshots/about-light.png)
- [Restored draft](assets/screenshots/recovery-dark.png)
- [Dark evidence](docs/qa/gui-dark.json) / [Light evidence](docs/qa/gui-light.json)
- [Actual-size light/dark icon sheet](assets/branding/icon-preview.png)

PNG variants cover 16–512 physical pixels, including raster choices for 2× slots; SVG scales
without a bitmap dependency. Native 16–128 px icon sheet and About/window integration were
visually inspected on light/dark backgrounds. A real HiDPI compositor session was not tested.

During initial review a preliminary verifier could run a stale pre-CLI binary with inherited
environment. It could therefore forward a literal `--version` and focus request to an existing
instance; no document change was observed. This was corrected before delivery: version probes
now have private HOME/XDG, no inherited display/bus, timeout, expected identity and no-session
assertions. A stale binary is rejected. This is recorded rather than claiming every early
experimental command had the final verifier's isolation guarantees.

## Performance and licensing

No persistence/search algorithm was changed to achieve a score. [PERF-018](docs/perf-decisions.md)
records five-run baseline/current steady-frame samples: final shortcut-enabled code measured
125966–133894 ns/op, 2001 B/op, 58 allocs/op. Historical noisy runs remain documented.
These are headless UI timings, not native scanout/fps measurements or zero-allocation claims.
The real Xvfb run reports llvmpipe/software rendering; hardware GPU behavior is not inferred.

[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) includes exact copied Feather-path evidence and
both Feather MIT and Lucide ISC/MIT notices, verified against official repository snapshots.
The original import commits are unknown and are not invented. Release archives contain root,
Go, linked-module, nested vendored and icon licenses with a stable dependency inventory.
Original project artwork stays MIT; AppStream metadata is CC0-1.0. No font files or stock
images were added. The social preview is ready for a manual maintainer upload, not uploaded.

Metadata and workflow choices were checked against the
[desktop Exec specification](https://specifications.freedesktop.org/desktop-entry/latest/exec-variables.html),
[AppStream upstream metadata](https://www.freedesktop.org/software/appstream/docs/chap-Metadata.html),
and [GitHub Actions secure-use guidance](https://docs.github.com/en/actions/reference/security/secure-use).
Offline validators and local workflow-equivalent checks passed. Hosted CI must pass for the published commit before publishing its pre-release. The local
validation recorded here predates that CI run; inspect the matching commit on GitHub Actions
for the publication-time result.

## Remaining gates before a stable release

- Preserve the [audit limitations](AUDIT_VERIFICATION.md): external mmap truncation can SIGBUS;
  clean-file search can differ from an externally changed editor; inode replacement does not
  preserve ACLs/xattrs/ownership/hard links; directory-fsync failures imply uncertain durability.
- Synchronous durable session writes can miss 8.3 ms frame budgets; zero allocations/frame
  remains unmet. Orphan drafts and sticky persistence error records remain conservative behavior.
- Validate native GNOME Wayland focus/tokens, real GPU drivers, HiDPI/accessibility and a full
  shortcut/dialog/IME matrix. Xvfb with direct test focus cannot establish compositor activation.
- Validate actual power-cut/disk-error behavior, native arm64 execution and distribution packages
  before claiming those environments. No AppImage/Flatpak/DEB support is claimed here.
- Commit reviewed source, rerun clean-checkout reproducibility and hosted CI, review unsigned
  artifact distribution/authentication and authorize publication separately.

Independent review follows Agent-1 architecture → Agent-2 core/release tooling → Agent-3 native
integration → Agent-4 complete-diff review. Review already required fixes for CLI exit codes,
stale modal identity, directory installation targets, unsafe stale-binary probing and ELF
validation. **APPROVED — Agent-4 QA Gatekeeper**, after independent code/report, test and artifact review. Final documentation-only status changes are rebuilt and compared again before delivery. Commit and pre-release publication were separately authorized by the maintainer; the stable-release gates remain unchanged.
