#!/usr/bin/env bash
# Build without publishing. Run from a clean checkout for distributable artifacts.
set -euo pipefail
umask 022
root=$(cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
version=${VERSION:-0.1.0-dev}
arch=${GOARCH:-amd64}
case "$version" in *[!A-Za-z0-9._+-]*|'') echo 'Invalid VERSION' >&2; exit 1;; esac
case "$arch" in amd64|arm64) ;; *) echo 'Supported build targets: amd64 and experimental arm64' >&2; exit 1;; esac
commit=$(git rev-parse HEAD)
source_state=clean
if [[ -n $(git status --porcelain --untracked-files=normal) ]]; then
 source_state=dirty
 commit="$commit-dirty"
 if [[ "$version" != *-dev && ${ALLOW_DIRTY:-0} != 1 ]]; then
  echo 'Refusing a non-development version from a dirty tree; use ALLOW_DIRTY=1 only for an explicit local candidate.' >&2
  exit 1
 fi
fi
epoch=${SOURCE_DATE_EPOCH:-$(git show -s --format=%ct HEAD)}
case "$epoch" in *[!0-9]*|'') echo 'Invalid SOURCE_DATE_EPOCH' >&2; exit 1;; esac
build_date=$(date -u -d "@$epoch" +%Y-%m-%dT%H:%M:%SZ)
out=${OUTPUT_DIR:-$root/dist}
mkdir -p "$out"
out=$(cd "$out" && pwd)
name="gpad-${version}-linux-${arch}"
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
stage="$work/$name"
mkdir -p "$stage/bin" "$stage/share/licenses/gpad" "$stage/share/applications" "$stage/share/metainfo" "$stage/share/doc/gpad"
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -tags mygo_noinspector -trimpath -buildvcs=false -ldflags "-buildid= -X gpad/internal/buildinfo.Version=$version -X gpad/internal/buildinfo.Commit=$commit -X gpad/internal/buildinfo.BuildDate=$build_date" -o "$stage/bin/gpad" ./cmd/gpad
install -m644 LICENSE "$stage/share/licenses/gpad/LICENSE"
install -m644 "$(go env GOROOT)/LICENSE" "$stage/share/licenses/gpad/Go-LICENSE"
install -m644 third_party/mygo/LICENSE "$stage/share/licenses/gpad/mygo-LICENSE"
install -m644 THIRD_PARTY_NOTICES.md "$stage/share/licenses/gpad/THIRD_PARTY_NOTICES.md"
mkdir -p "$stage/share/licenses/gpad/docs/notices"
install -m644 docs/notices/feather-LICENSE docs/notices/lucide-LICENSE "$stage/share/licenses/gpad/docs/notices/"
# Include the exact license texts from every selected module, not a guessed SPDX list.
python3 scripts/release-notices.py "$stage/share/licenses/gpad"
install -m644 packaging/gpad.desktop "$stage/share/applications/"
install -m644 packaging/io.github.thanhnam2811.gpad.metainfo.xml "$stage/share/metainfo/"
for size in 16 24 32 48 64 128 256 512; do
 icon_dir="$stage/share/icons/hicolor/${size}x${size}/apps"
 mkdir -p "$icon_dir"
 install -m644 "assets/icons/$size.png" "$icon_dir/gpad.png"
done
mkdir -p "$stage/share/icons/hicolor/scalable/apps" "$stage/share/icons/hicolor/symbolic/apps"
install -m644 assets/icon.svg "$stage/share/icons/hicolor/scalable/apps/gpad.svg"
install -m644 assets/icon-symbolic.svg "$stage/share/icons/hicolor/symbolic/apps/gpad-symbolic.svg"
printf 'version=%s\ncommit=%s\nbuild_date=%s\nsource_date_epoch=%s\ngo=%s\ntarget=linux/%s\nsource_state=%s\n' "$version" "$commit" "$build_date" "$epoch" "$(go env GOVERSION)" "$arch" "$source_state" > "$stage/share/doc/gpad/build-info.txt"
install -m755 scripts/install.sh "$stage/install.sh"
for document in README.md CONTRIBUTING.md AGENTS.md LICENSE CHANGELOG.md BRANDING.md AUDIT_VERIFICATION.md RELEASE_READINESS.md SECURITY.md THIRD_PARTY_NOTICES.md; do
 install -m644 "$document" "$stage/share/doc/gpad/$document"
done
# Preserve the relative documentation/asset layout used by README and branding links.
cp -R docs "$stage/share/doc/gpad/docs"
mkdir -p "$stage/share/doc/gpad/assets"
install -m644 assets/icon.svg assets/icon.png assets/icon-symbolic.svg "$stage/share/doc/gpad/assets/"
for directory in branding icons screenshots; do
 cp -R "assets/$directory" "$stage/share/doc/gpad/assets/$directory"
done
# Stable ownership, ordering, timestamps and gzip header make identical inputs reproducible.
tar --sort=name --mode='u+rwX,go+rX,go-w' --mtime="@$epoch" --owner=0 --group=0 --numeric-owner -C "$work" -cf - "$name" | gzip -n > "$out/$name.tar.gz"
(cd "$out" && sha256sum "$name.tar.gz" > "$name.sha256" && sha256sum -- *.tar.gz > SHA256SUMS)
printf '%s\n' "$out/$name.tar.gz"
