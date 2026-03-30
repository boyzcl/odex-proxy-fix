# codex-proxy-fix

Make Codex reliably use your existing local HTTP proxy and reduce `Reconnecting...` issues with one command.

`codex-proxy-fix` is a macOS-first CLI for users whose local proxy is already running, but Codex is not consistently inheriting or using it.

Current position:

- Release stage: `alpha`
- Officially validated here: `macOS`
- Windows/Linux: in progress, not yet officially tested

Quick start:

```bash
codex-proxy doctor --verbose
codex-proxy fix
codex-proxy status
```

If Codex still reconnects when launched normally:

```bash
codex-proxy launch
```

If you want to remove the fix and restore captured env state:

```bash
codex-proxy unset
```

This tool helps when the problem is on the local networking or proxy inheritance side. It does not fix OpenAI server incidents, region-specific outages, or unstable model backends.

## What This Tool Does

The tool focuses on one problem:

- Codex is unstable because it is not consistently using the user's existing local HTTP proxy

It provides:

- `codex-proxy fix`
  - detect Codex
  - detect and verify a local HTTP proxy
  - install a best-effort persistent fix
  - create a fallback launcher
- `codex-proxy doctor`
  - inspect the current environment without changing anything
- `codex-proxy status`
  - show the installed state
- `codex-proxy launch`
  - launch Codex with explicit proxy env as a reliable fallback
- `codex-proxy unset`
  - remove tool-managed files and restore captured env state when possible

## What This Tool Does Not Do

- It does not create internet access on its own.
- It does not fix every possible `Reconnecting...` cause.
- It does not guarantee recovery from server-side incidents or unstable model backends.
- It does not change system-wide proxy settings by default.

## Support Matrix

- `macOS`: alpha, locally verified in this repository
- `Windows`: code path exists, not officially tested
- `Linux`: code path exists, not officially tested

If you publish a GitHub release today, describe it as:

> macOS-first alpha. Windows and Linux are in-progress and unverified.

## How It Works

On macOS, `fix` currently installs:

1. a user-scoped proxy env script under `~/Library/Application Support/codex-proxy-fix/`
2. a user `LaunchAgent` under `~/Library/LaunchAgents/`
3. a fallback launcher script under the same application support directory
4. a local state file that records:
   - detected Codex paths
   - selected proxy
   - managed file paths
   - a snapshot of the original persistent env values before mutation

This means the tool can later remove what it created and restore the previous env snapshot during `unset`.

## Installation

### Option 1: Download a GitHub release artifact

Download the archive for your platform from the release page, then extract it and move the binary somewhere in your `PATH`.

Example for macOS:

```bash
tar -xzf codex-proxy_0.1.0-darwin_arm64.tar.gz
chmod +x codex-proxy
mv codex-proxy /usr/local/bin/codex-proxy
```

### Option 2: Build from source

```bash
go build -o ./bin/codex-proxy ./cmd/codex-proxy
```

## Quick Start

Run diagnosis first:

```bash
codex-proxy doctor --verbose
```

If it detects a valid local proxy, install the fix:

```bash
codex-proxy fix
```

If Codex still reconnects when launched normally, use the explicit fallback:

```bash
codex-proxy launch
```

Check the installed state:

```bash
codex-proxy status
```

Remove the fix and restore captured env state:

```bash
codex-proxy unset
```

## What Gets Written

On macOS, this tool writes files under paths like:

- `~/Library/Application Support/codex-proxy-fix/state.json`
- `~/Library/Application Support/codex-proxy-fix/setenv.sh`
- `~/Library/Application Support/codex-proxy-fix/launch-codex.sh`
- `~/Library/LaunchAgents/com.codexproxyfix.env.plist`

It also updates the current login session proxy env through `launchctl setenv`.

## Rollback

`codex-proxy unset` is intended to:

1. unload the managed LaunchAgent
2. remove tool-managed files
3. restore the previously captured persistent env values when a snapshot exists

If there was no prior value for a managed variable, `unset` removes that variable from the managed persistent environment.

## Known Limits

- The macOS persistent fix is best-effort because GUI app env inheritance is a platform behavior, not something this tool fully controls.
- If OpenAI has a server-side incident or model backend degradation, this tool may not help.
- Windows/Linux implementations exist but should not yet be treated as production-ready.

## Development

Run tests:

```bash
go test ./...
```

Build a release bundle:

```bash
bash scripts/release/build.sh
```

This generates archives and `checksums.txt` under `dist/`.

## Repository Docs

- [Implementation Blueprint](./CODEX_PROXY_FIX_IMPLEMENTATION_BLUEPRINT.md)
- [Release Readiness Plan](./RELEASE_READINESS_PLAN.md)
