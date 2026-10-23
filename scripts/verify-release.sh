#!/usr/bin/env bash
set -euo pipefail
archive=$(realpath -- "${1:?Usage: verify-release.sh ARCHIVE.tar.gz}")
base=$(basename -- "$archive")
(cd "$(dirname -- "$archive")" && sha256sum -c -- "${base%.tar.gz}.sha256")
# Inspect before extraction; reject links, devices, absolute and escaping paths.
python3 - "$archive" <<'PY'
import pathlib, sys, tarfile
with tarfile.open(sys.argv[1], 'r:gz') as archive:
    roots = set()
    for member in archive:
        path = pathlib.PurePosixPath(member.name)
        if path.is_absolute() or '..' in path.parts or not (member.isfile() or member.isdir()):
            raise SystemExit('Unsafe archive member: ' + member.name)
        roots.add(path.parts[0])
    if len(roots) != 1:
        raise SystemExit('Expected exactly one release root')
PY
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
tar --no-same-owner -xzf "$archive" -C "$work"
release="$work/${base%.tar.gz}"
for item in bin/gpad share/licenses/gpad/LICENSE share/licenses/gpad/dependencies.json share/doc/gpad/build-info.txt share/applications/gpad.desktop share/metainfo/io.github.thanhnam2811.gpad.metainfo.xml; do
 [[ -s "$release/$item" ]] || { echo "Missing $item" >&2; exit 1; }
done
file "$release/bin/gpad"
python3 - "$release/bin/gpad" "$base" <<'PYELF'
import pathlib, struct, sys
header = pathlib.Path(sys.argv[1]).read_bytes()[:64]
expected = 62 if sys.argv[2].endswith('-amd64.tar.gz') else 183 if sys.argv[2].endswith('-arm64.tar.gz') else None
if len(header) < 64 or header[:6] != bytes.fromhex('7f454c460201') or expected is None or struct.unpack_from('<H', header, 18)[0] != expected:
    raise SystemExit('Executable is not the expected 64-bit little-endian Linux target')
PYELF
# Execute only a matching local amd64 build; arm64 is an experimental cross-build.
if [[ "$base" == *-amd64.tar.gz && $(uname -m) == x86_64 ]]; then
 # Even an accidentally stale pre-CLI binary must never discover a user session.
 mkdir -m700 "$work/home" "$work/runtime" "$work/config" "$work/data" "$work/cache"
 identity=$(timeout 10s env -i PATH="$PATH" LANG=C.UTF-8 HOME="$work/home" \
   XDG_RUNTIME_DIR="$work/runtime" XDG_CONFIG_HOME="$work/config" \
   XDG_DATA_HOME="$work/data" XDG_CACHE_HOME="$work/cache" \
   "$release/bin/gpad" --version)
 expected=${base#gpad-}
 expected=${expected%-linux-amd64.tar.gz}
 [[ "$identity" == "Gopad++ $expected "* ]] || { echo 'Unexpected --version identity' >&2; exit 1; }
 printf '%s\n' "$identity"
 [[ ! -e "$work/runtime/gpad.sock" && ! -e "$work/home/.config/gpad" ]] || { echo 'Version probe initialized application state' >&2; exit 1; }
fi
DESTDIR="$work/install" PREFIX=/usr "$release/install.sh"
cmp "$release/bin/gpad" "$work/install/usr/bin/gpad"
printf 'Verified %s\n' "$base"
