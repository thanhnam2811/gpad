#!/usr/bin/env bash
# Run from an extracted release, or pass its directory as the first argument.
set -euo pipefail
source_dir=${1:-$(cd -- "$(dirname -- "$0")" && pwd)}
prefix=${PREFIX:-/usr/local}
destination=${DESTDIR:-}
case "$prefix" in /*) ;; *) echo 'PREFIX must be absolute' >&2; exit 1;; esac
case "$destination" in ''|/*) ;; *) echo 'DESTDIR must be absolute' >&2; exit 1;; esac
for path in "$prefix" "$destination"; do
 case "/$path/" in */../*|*/./*) echo 'Dot path components are not allowed' >&2; exit 1;; esac
done
[[ -f "$source_dir/bin/gpad" && -d "$source_dir/share" ]] || { echo 'Expected extracted release directory' >&2; exit 1; }
# Stage in the destination filesystem: replacing a running executable must
# rename a new inode rather than opening the live inode (ETXTBSY).
bin_dir="$destination$prefix/bin"
install -d "$bin_dir"
staged_binary=$(mktemp "$bin_dir/.gpad-install.XXXXXX")
trap 'rm -f -- "$staged_binary"' EXIT
install -m755 "$source_dir/bin/gpad" "$staged_binary"
mv -fT -- "$staged_binary" "$bin_dir/gpad"
# Release trees contain only directories and ordinary files.
while IFS= read -r -d '' file; do
 relative=${file#"$source_dir/"}
 install -Dm644 "$file" "$destination$prefix/$relative"
done < <(find "$source_dir/share" -type f -print0)
printf 'Installed to %s%s\n' "$destination" "$prefix"
