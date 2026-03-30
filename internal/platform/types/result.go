package types

import "github.com/boyzcl/codex-proxy-fix/internal/state"

type InstallResult struct {
	PersistentInstalled bool
	PersistentVerified  bool
	FallbackInstalled   bool
	ManagedPaths        []string
	Notes               []string
	OriginalEnv         *state.EnvSnapshot
}
