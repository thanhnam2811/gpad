# Building and checking release candidates

The source version is `0.1.0-dev` in `internal/buildinfo`. No published GitHub releases were present when checked on 2026-10-20. Ubuntu 24.04 Linux amd64 is the native validation target. Linux arm64 is experimental and cross-compilation only; it has no native runtime validation.

Use the exact Go version in `go.mod`, Python 3, GNU tar/coreutils and gzip. Runtime Linux libraries/display requirements are documented in README. Release binaries disable the mygo inspector and use `CGO_ENABLED=0`; this does not remove their dynamically loaded native GUI libraries.

From a clean source checkout:

```sh
go mod verify
go build ./...
go test ./...
go test -race ./...
go vet ./...
VERSION=0.1.0-dev scripts/release.sh
scripts/verify-release.sh dist/gpad-0.1.0-dev-linux-amd64.tar.gz
```

The script creates a tarball, its `.sha256` sidecar and `SHA256SUMS` in `dist`. It bundles desktop/AppStream metadata, hicolor icons, project and linked-module license notices, dependency inventory and build identity. It never publishes, installs on the host or modifies user sessions. A sidecar checksum detects accidental damage; it is not a signature and cannot authenticate an untrusted download.

`VERSION`, `OUTPUT_DIR`, `SOURCE_DATE_EPOCH` and `GOARCH` are optional inputs. The default timestamp is the Git commit timestamp, not wall-clock time. Do not put credentials or whitespace into version identifiers. Build identity includes version, source commit, fixed build date and Go version. A dirty source tree gets a `-dirty` commit suffix and explicit source-state metadata. Non-development versions require a clean tree; `ALLOW_DIRTY=1 VERSION=0.1.0-rc.1` permits an explicitly local candidate. Never distribute a dirty build as a clean release.

For an experimental arm64 cross-build:

```sh
GOARCH=arm64 OUTPUT_DIR=/tmp/gpad-arm64 scripts/release.sh
```

Verify reproducibility from two independent clean checkouts of the same commit, with the same Go toolchain, module cache inputs, target and environment:

```sh
OUTPUT_DIR=/tmp/gpad-first scripts/release.sh
OUTPUT_DIR=/tmp/gpad-second scripts/release.sh
cmp /tmp/gpad-first/SHA256SUMS /tmp/gpad-second/SHA256SUMS
```

Run the two build commands in the corresponding checkouts. Archives normalize entry order, ownership, timestamps, and gzip headers. Go builds omit source paths, VCS working-tree metadata and random build IDs. Embedded dependency notices omit absolute paths.

Extract a verified archive, then install under an explicit prefix:

```sh
tar -xzf dist/gpad-0.1.0-dev-linux-amd64.tar.gz
PREFIX="$HOME/.local" ./gpad-0.1.0-dev-linux-amd64/install.sh
```

Packaging tools can stage safely with `DESTDIR=/tmp/gpad-stage PREFIX=/usr ./gpad-0.1.0-dev-linux-amd64/install.sh`. The verifier validates archive member types/paths and ELF architecture, checks the version in a timeout-bound private HOME/XDG environment without inherited display or session bus, then stages to temporary DESTDIR and compares the installed executable. A stale pre-CLI binary must fail verification rather than forward to a running user instance. Installation does not require or perform a running-session restart. It does not run host icon or desktop cache refreshes automatically.

The manual Actions workflow builds artifacts for download, compares repeated archives, verifies them and uploads workflow artifacts only. It has read-only repository permissions; it does not create tags, GitHub releases or deployments. All action revisions were resolved through the official GitHub repository API on 2026-10-20. `ci.yml` runs verification on Ubuntu 24.04 with the exact Go version from `go.mod`. Future publication requires a separate authorized release decision and native GUI smoke tests.

## Local tooling evidence (2026-10-20)

Two independent frozen source copies produced byte-identical amd64 archives using Go 1.27.2; the measured candidate checksum was `75a3b25891adc54fba16a72152d17e18e212c6665f0b0c1b1e98d951de206aa2`. This is evidence for that source snapshot, not a checksum for a later release. The snapshots carried an honest `-dirty` source identifier because the changes have not been committed. Later installer/documentation additions require a final archive comparison before distribution. Native ARM64 execution was not tested; its cross-build passed.

Desktop and AppStream validators passed. Shell syntax checks, asset checks, isolated DESTDIR installation and dirty-version rejection passed. An isolated test reinstalled over a running synthetic executable and verified that the existing process survived, replacement bytes were installed and staging files were cleaned. The archive verifier must be rerun against the final binary after CLI version integration. No host installation or publication was performed.

## Uninstall

Close GPAD first. Remove only these GPAD paths beneath the prefix used for installation
(for a user install, normally `~/.local`; for a staged package, the package manager owns them):

- `bin/gpad`
- `share/applications/gpad.desktop`
- `share/metainfo/io.github.thanhnam2811.gpad.metainfo.xml`
- `share/icons/hicolor/{16,24,32,48,64,128,256,512}x{same-size}/apps/gpad.png`
- `share/icons/hicolor/scalable/apps/gpad.svg`
- `share/icons/hicolor/symbolic/apps/gpad-symbolic.svg`
- `share/doc/gpad` and `share/licenses/gpad`

In the icon list, each directory repeats the same size, for example `16x16` and `128x128`.
Do not remove shared parent directories or other applications' icon files. An explicit
`gpad --install-desktop` can place a launcher under a separate XDG_DATA_HOME; remove its
`applications/gpad.desktop` and `icons/hicolor/.../gpad` assets there only if you installed them.
**Retain `~/.config/gpad/session` and all your documents.** Neither installation nor uninstall
requires deleting recovery data or changing default MIME associations.

## First pre-release publication

The maintainer authorized publication of `v0.1.0-rc.1` as a pre-release on 2026-10-23.
Publish only after the reviewed changes are committed, `main` is synchronized, hosted CI
passes for that commit and two clean-source builds produce identical artifacts. Use the
versioned notes in `docs/releases/0.1.0-rc.1.md`; upload only the validated amd64 archive and
its checksum files. Do not upload earlier `-dirty` previews or label the candidate stable.
The manual artifact workflow remains non-publishing; publication is a separate explicit step.
