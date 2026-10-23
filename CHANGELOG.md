# Changelog

Changes are recorded here before a maintainer creates a versioned release. No stable release
is declared by this file. Release candidates and validation limits are tracked in
[RELEASE_READINESS.md](RELEASE_READINESS.md).

## 0.1.0-rc.1 — 2026-10-23

First public pre-release; not a stable-support commitment.

### Reliability

- Durable same-directory file replacement with symlink/mode handling.
- Private immutable shadow drafts and fsynced WAL recovery checkpoints.
- Serialized single-instance startup and ownership held through session shutdown.
- Autosave shutdown joining, search-generation checks and terminal asynchronous-load errors.
- Recovery fixes for untitled content and deliberately emptied documents.
- Evidence and remaining limitations in [AUDIT_VERIFICATION.md](AUDIT_VERIFICATION.md).

### Project and desktop integration

- Refined G/++ branding, scalable/symbolic icons, raster sizes and social-preview source.
- Current installation/development documentation and contributor/report templates.
- Central build-version metadata, Linux desktop/AppStream packaging and reproducible archives.
- CI build/test/race/vet/dependency checks and manual artifact creation without publication.
- Isolated native GUI smoke/capture tooling and release-readiness evidence.
