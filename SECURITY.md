# Security and sensitive data

Gopad++ is in early development. There is no supported stable release series or security
response SLA yet. The current maintenance target is the `main` branch.

If a report could expose document contents, credentials or a practical exploit, do not post
those details in a public issue. Use GitHub's private vulnerability-reporting option if the
repository offers it. If it is unavailable, open a minimal issue requesting a private contact
channel without including the sensitive details. No private-reporting service is promised here.

For an ordinary bug, include a synthetic reproduction, build version and relevant OS/display
information. Do not attach your actual session WAL, shadow drafts, clipboard contents or
personal documents; these files can contain complete unsaved text.

Known data-integrity and memory-mapping limits are documented in
[AUDIT_VERIFICATION.md](AUDIT_VERIFICATION.md) and [RELEASE_READINESS.md](RELEASE_READINESS.md).
Recovery snapshots supplement ordinary backups. Do not delete session files while GPAD runs.
