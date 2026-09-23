# Security policy

## Supported versions

Only the latest tagged release and the master branch receive fixes.

## Reporting

Report vulnerabilities privately via GitHub Security Advisories on the
repository, or by email to the address listed in the module metadata.
Do not open public issues for unpatched vulnerabilities.

Expected response: acknowledgment within a few days, a fix or a
documented decision before public disclosure.

## Scope notes

This library implements a protocol where each peer is trusted by
default but malformed input must never panic or corrupt state.

- `CommandTransport` and the examples execute a caller-supplied binary.
  Selecting the binary is the caller's responsibility, matching how
  ACP clients launch agents.
- Incoming protocol lines are size-capped at 64 MiB.
- All protocol messages decode through typed structures; unknown union
  variants are preserved in `Raw` fields rather than dropped.
- Keep `gosec`, `govulncheck`, and `gitleaks` clean in CI.
