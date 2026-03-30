# Release Readiness Plan

## Goal

Bring `codex-proxy-fix` from a working MVP scaffold to a public GitHub release candidate that is safe enough for external macOS users to try.

The release target for this pass is:

- Public GitHub repository: yes
- Public GitHub pre-release: yes
- Positioning: `macOS alpha`
- Windows/Linux: code in progress, not yet claimed as tested support

## Why This Plan Exists

The project already has:

- a working Go CLI
- real macOS fix flow
- state tracking
- release packaging
- local successful verification on the developer machine

But it is still missing several release-critical qualities:

1. Reversible rollback is incomplete.
2. macOS install/uninstall error handling is too optimistic.
3. User-facing documentation is not yet release-grade.
4. The repository is missing licensing and explicit support boundaries.
5. Release validation is not yet documented as a checklist.

## Release-Critical Gaps

### 1. Reversible Rollback

Problem:

- `fix` overwrites proxy-related environment values without preserving the original user state.
- `unset` removes tool-managed state, but cannot restore the user's previous values safely.

Required outcome:

- Before mutating persistent environment state, capture the original values.
- Store whether each variable originally existed.
- `unset` must restore the original values when possible.
- On platforms where restoration is not supported yet, the README must say so explicitly.

Acceptance criteria:

- A user with pre-existing proxy env can run `fix` and later `unset` without losing their original values.
- State file includes the original env snapshot.

### 2. Safer macOS Install/Uninstall Behavior

Problem:

- `launchctl` operations currently ignore important failures.
- The tool can report success too optimistically.

Required outcome:

- Capture and classify failures for:
  - LaunchAgent bootstrap
  - kickstart
  - session env application
  - uninstall restoration
- Distinguish:
  - persistent install written
  - persistent install registered
  - persistent env verified
  - fallback launcher created

Acceptance criteria:

- macOS `fix` output is truthful when only partial setup succeeded.
- macOS `unset` restores the session env snapshot when available.

### 3. Release-Grade User Documentation

Problem:

- README is still scaffold-oriented.
- External users do not yet have enough information to install, trust, validate, and undo the tool.

Required outcome:

- README must clearly describe:
  - project purpose
  - current support level
  - how `fix`, `doctor`, `status`, `launch`, `unset` work
  - what files are written
  - what gets changed in the user session
  - what rollback does
  - known limits
  - macOS alpha positioning

Acceptance criteria:

- A new GitHub visitor can understand what the tool does and whether they should use it.

### 4. Repository Release Basics

Problem:

- No license file exists.
- No public release-specific checklist exists.

Required outcome:

- Add an open-source license.
- Keep this plan document in-repo as the release checklist and decision record.

Acceptance criteria:

- Repo is legally publishable and operationally understandable.

### 5. Verification and Release Confidence

Problem:

- Build and packaging work, but release readiness is not formalized.

Required outcome:

- Re-run:
  - formatting
  - tests
  - local build
  - doctor
  - fix
  - status
  - release packaging
- Record remaining risks honestly.

Acceptance criteria:

- We can state clearly whether the repository is ready for:
  - public code visibility
  - public alpha release
  - stable release

## Scope for This Pass

### In Scope

1. Add original env snapshot support.
2. Implement reversible restoration for macOS and Windows code paths.
3. Improve macOS install/uninstall truthfulness.
4. Upgrade README to release quality.
5. Add `LICENSE`.
6. Re-verify build/test/package/macOS local behavior.

### Out of Scope

1. Claiming Windows or Linux are production-ready.
2. Full package manager automation for Homebrew/winget/Scoop.
3. Stable `v1.0.0` positioning.
4. GUI app or installer app.

## Execution Order

1. Extend state model to persist original env values.
2. Update platform install/uninstall paths to capture and restore env snapshots.
3. Improve macOS install result reporting and app logic.
4. Upgrade README and add `LICENSE`.
5. Re-run verification.
6. Decide whether the project is:
   - not ready
   - ready for public repo only
   - ready for alpha release

## Release Bar for This Pass

The repository is considered ready for a public `alpha` release only if all of the following are true:

1. macOS `fix` works on the current machine.
2. macOS `status` reflects the installed state accurately.
3. macOS `unset` can safely remove tool-managed files and restore captured env values.
4. README accurately describes support level and limits.
5. License is present.
6. `go test ./...` passes.
7. release packaging completes successfully.

## Expected Final Positioning

If this plan is completed successfully, the project should be described as:

> `codex-proxy-fix` is a macOS-first alpha CLI that helps Codex reliably use an existing local HTTP proxy and reduce reconnecting issues. Windows and Linux code paths exist, but are not yet officially tested.
