# codex-proxy-fix

[中文说明 / Chinese README](./README.zh-CN.md)

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

## Why This Exists

`Reconnecting...` is a symptom, not a single root cause.

In practice, people see this state for several very different reasons:

1. Local proxy inheritance problems
   - A local proxy is already running, but Codex was launched without the right proxy env.
   - This is especially common with GUI app launches on macOS, where shell env and app env can drift apart.

2. Local network path instability
   - The direct route from the machine to the service is unstable.
   - Regular browsing may still work, but long-lived or streaming connections fail more often.

3. Local proxy node or egress IP instability
   - The proxy process is running, but the selected local node, upstream route, or egress IP is unstable.
   - Simple requests may work while Codex still reconnects under sustained session traffic.

4. Server-side or regional incidents
   - OpenAI may be degraded in a region, for a time window, or for a specific model/backend path.
   - In those cases many users see reconnecting at once.

5. Auth/session problems
   - The user session is stale or a request path fails with authorization/session recovery issues.

This tool is intentionally focused on the first three categories, especially the first one.

## What We Think Is Actually Happening

Codex is not just doing one-shot requests. It relies on longer-lived session behavior, streaming, and reconnection semantics.  
That means it is more sensitive than ordinary web browsing to:

- broken proxy inheritance
- unstable egress paths
- DNS/TLS friction
- connection resets on long-lived traffic

So when users say "the internet works, but Codex keeps reconnecting", that is believable.  
The network may be good enough for ordinary browsing while still being too unstable for Codex sessions.

## What This Tool Can Help With

This tool is a good fit when:

- a local HTTP proxy is already running on the machine
- Codex works better when launched from a shell with proxy env than from the app icon
- reconnecting is mostly a machine-specific or network-path-specific problem
- the user needs a repeatable, reversible, low-intrusion fix

More concretely, it helps when the main problem is:

- Codex not inheriting proxy env
- Codex using the wrong local route
- GUI launch behavior differing from terminal launch behavior
- local proxy path instability that improves when traffic is forced through the intended local HTTP proxy

## What This Tool Cannot Solve

This tool does not fix:

- OpenAI server incidents
- region-wide service degradation
- unstable model backends
- account/auth/session issues
- the absence of a real working local proxy

It also does not create internet access on its own.

If the problem is on the service side, this tool may provide no benefit at all.

## What This Tool Actually Does

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

## How The Fix Works

On macOS, `fix` currently installs:

1. a user-scoped proxy env script under `~/Library/Application Support/codex-proxy-fix/`
2. a user `LaunchAgent` under `~/Library/LaunchAgents/`
3. a fallback launcher script under the same application support directory
4. a local state file that records:
   - detected Codex paths
   - selected proxy
   - managed file paths
   - a snapshot of the original persistent env values before mutation

The strategy has two layers:

1. Best-effort persistent fix
   - make normal GUI launches more likely to inherit the intended proxy env

2. Explicit fallback launch
   - if normal app launching still reconnects, `codex-proxy launch` starts Codex with explicit proxy env for that session

This two-layer model exists because GUI environment inheritance is not perfectly deterministic across systems and sessions.

## Why We Chose This Approach

We deliberately chose:

- low intrusion over system-wide proxy rewrites
- reversible env-based integration over patching Codex internals
- a user-level fix over admin-level system modifications

The benefits are:

- easier rollback
- lower risk to the rest of the system
- easier for users to understand what changed
- easier to package as a small CLI utility

## Benefits

- One-command fix path for the most common local proxy inheritance failure mode
- Clear diagnosis before mutation
- Explicit fallback when best-effort persistence is not enough
- Reversible state with captured env snapshot
- No default system-wide proxy rewrite

## Boundaries and Limitations

- The macOS persistent fix is best-effort because GUI app env inheritance is a platform behavior, not something this tool fully controls.
- A local proxy can be detected and still be a poor route for Codex if the upstream node or egress IP is unstable.
- Windows/Linux implementations exist but should not yet be treated as production-ready.
- This tool improves one high-probability class of reconnecting causes. It is not a universal fix for every reconnecting report.

## Support Matrix

- `macOS`: alpha, locally verified in this repository
- `Windows`: code path exists, not officially tested
- `Linux`: code path exists, not officially tested

If you publish a GitHub release today, describe it as:

> macOS-first alpha. Windows and Linux are in-progress and unverified.

## Installation

### Option 1: Download a GitHub release artifact

Download the archive for your platform from the release page, then extract it and move the binary somewhere in your `PATH`.

Example for macOS:

```bash
tar -xzf codex-proxy_0.1.0-alpha.1_darwin_arm64.tar.gz
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

- [Chinese README](./README.zh-CN.md)
- [Implementation Blueprint](./CODEX_PROXY_FIX_IMPLEMENTATION_BLUEPRINT.md)
- [Release Readiness Plan](./RELEASE_READINESS_PLAN.md)
- [Release Notes: v0.1.0-alpha.1](./RELEASE_NOTES_v0.1.0-alpha.1.md)
