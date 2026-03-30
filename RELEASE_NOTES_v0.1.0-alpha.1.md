# codex-proxy-fix v0.1.0-alpha.1

## Title

`v0.1.0-alpha.1` macOS-first alpha release

## Summary

This is the first public alpha release of `codex-proxy-fix`.

The goal of this release is simple:

- help Codex reliably use an existing local HTTP proxy
- reduce local `Reconnecting...` issues caused by proxy inheritance or local network path problems

This release should be presented as:

> macOS-first alpha. Windows and Linux are in progress and not yet officially tested.

## What's Included

- working CLI commands:
  - `codex-proxy fix`
  - `codex-proxy doctor`
  - `codex-proxy status`
  - `codex-proxy launch`
  - `codex-proxy unset`
- macOS local proxy detection and verification
- macOS Codex installation detection
- macOS best-effort persistent fix with `LaunchAgent + launchctl setenv`
- fallback launcher generation
- local state tracking
- rollback snapshot support for persistent proxy env values
- release packaging artifacts for macOS, Linux, and Windows

## Validation Completed

The following were verified locally during this release-prep pass:

- `go test ./...`
- local build
- `doctor`
- real `fix`
- real `status`
- real `unset`
- real reinstall with `fix`
- release packaging output under `dist/`

## Known Limits

- This release is only locally validated on `macOS`.
- Windows and Linux code paths exist but should still be treated as unverified.
- This tool does not solve OpenAI server-side incidents or model backend instability.
- The macOS persistent fix is best-effort because GUI app env inheritance is platform-dependent.

## Recommended GitHub Release Body

```md
## codex-proxy-fix v0.1.0-alpha.1

First public alpha release.

`codex-proxy-fix` helps Codex reliably use an existing local HTTP proxy and reduce `Reconnecting...` issues caused by local proxy inheritance or local network path problems.

### Included in this release

- `fix`, `doctor`, `status`, `launch`, `unset`
- macOS-first working implementation
- local proxy detection and verification
- persistent best-effort macOS fix
- fallback launcher
- rollback snapshot support
- packaged release artifacts with checksums

### Support status

- macOS: alpha, locally validated
- Windows: in progress, unverified
- Linux: in progress, unverified

### Important limits

- This tool does not create internet access on its own.
- It does not fix OpenAI server incidents or unstable model backends.
- The persistent macOS fix is best-effort because GUI environment inheritance is platform-dependent.
```
