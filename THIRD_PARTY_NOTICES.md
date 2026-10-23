# Third-party notices

Gopad++ application code and original G/++ branding are MIT licensed. Third-party
components retain their licenses; the repository LICENSE does not replace those notices.

## Toolbar line icons

The existing paths in `internal/ui/icons_svg.go` include Feather geometry (for example,
the Save path and its two polylines exactly match Feather's `save.svg`) and adapted Lucide
geometry (for example Folder Open and Copy). Retain the compatible upstream notices for
this derived icon collection. These are not claimed as newly authored Gopad++ artwork.
The baseline does not record the exact importing versions; the notices below were verified
against the official repositories on 2026-10-20. Current snapshots are recorded for traceability,
not presented as the original importing commits.

- [Feather license](docs/notices/feather-LICENSE), official snapshot [`3dc050d97405`](https://github.com/feathericons/feather/blob/3dc050d97405062eba78aa57115c0a15c63abdaa/LICENSE).
- [Lucide license](docs/notices/lucide-LICENSE), official snapshot [`c29c276f1659`](https://github.com/lucide-icons/lucide/blob/c29c276f16590cf017126671535ad8c792be64d8/LICENSE).

Feather uses MIT (Cole Bemis). Lucide uses ISC and retains MIT notices for Feather-derived
icons; both full notice files are bundled with release archives. Simple adapted shapes retain
the upstream attribution as a collection rather than pretending the introducing commit is known.

## Go and libraries

The vendored `third_party/mygo` has its own MIT license and nested notices. Release tooling
collects actual linked-module license, notice, author and patent files from the selected module
versions, plus the Go runtime LICENSE. `dependencies.json` in an archive records those inputs.
No font binary or stock image was added by this polish; social text uses system DejaVu Sans
rendered into project PNG artwork.

AppStream metadata in `packaging/io.github.thanhnam2811.gpad.metainfo.xml` is CC0-1.0.
