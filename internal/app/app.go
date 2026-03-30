package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/boyzcl/codex-proxy-fix/internal/detect"
	"github.com/boyzcl/codex-proxy-fix/internal/platform"
	"github.com/boyzcl/codex-proxy-fix/internal/platform/common"
	"github.com/boyzcl/codex-proxy-fix/internal/state"
	"github.com/boyzcl/codex-proxy-fix/internal/ui"
)

const (
	exitSuccess                = 0
	exitCodexNotFound          = 10
	exitNoProxy                = 11
	exitPersistentInstallFail  = 12
	exitFallbackInstallFail    = 13
	exitVerificationIncomplete = 14
	exitUnexpected             = 20
)

type options struct {
	Port      int
	CodexPath string
	Yes       bool
	DryRun    bool
	Verbose   bool
	JSON      bool
}

type doctorReport struct {
	Platform          detect.PlatformInfo     `json:"platform"`
	Codex             detect.CodexInstall     `json:"codex"`
	ProxyCandidates   []detect.ProxyCandidate `json:"proxy_candidates"`
	SelectedProxy     string                  `json:"selected_proxy,omitempty"`
	PersistentState   string                  `json:"persistent_state,omitempty"`
	FallbackAvailable bool                    `json:"fallback_available"`
}

func Run(args []string, stdout, stderr io.Writer, version, commit string) int {
	cmd, opts, rest, err := parseArgs(args)
	if err != nil {
		ui.Line(stderr, "error: %v", err)
		printUsage(stderr)
		return exitUnexpected
	}

	switch cmd {
	case "", "help", "--help", "-h":
		printUsage(stdout)
		return exitSuccess
	case "version":
		ui.Line(stdout, "codex-proxy %s (%s)", version, commit)
		return exitSuccess
	case "doctor":
		return runDoctor(opts, stdout, stderr)
	case "status":
		return runStatus(opts, stdout, stderr)
	case "fix":
		return runFix(opts, stdout, stderr)
	case "launch":
		return runLaunch(opts, rest, stdout, stderr)
	case "unset":
		return runUnset(opts, stdout, stderr)
	default:
		ui.Line(stderr, "error: unknown command %q", cmd)
		printUsage(stderr)
		return exitUnexpected
	}
}

func runDoctor(opts options, stdout, stderr io.Writer) int {
	report, code, err := gatherReport(opts)
	if err != nil {
		ui.Line(stderr, "error: %v", err)
		return code
	}
	if opts.JSON {
		if err := ui.PrintJSON(stdout, report); err != nil {
			ui.Line(stderr, "error: %v", err)
			return exitUnexpected
		}
		return exitSuccess
	}
	ui.Line(stdout, "Codex proxy doctor")
	ui.Line(stdout, "")
	ui.Line(stdout, "Platform: %s/%s", report.Platform.OS, report.Platform.Arch)
	if report.Codex.AnyFound() {
		if report.Codex.GUIPath != "" {
			ui.Line(stdout, "Codex GUI: %s", report.Codex.GUIPath)
		}
		if report.Codex.CLIPath != "" {
			ui.Line(stdout, "Codex CLI: %s", report.Codex.CLIPath)
		}
	} else {
		ui.Line(stdout, "Codex: not found")
	}
	ui.Line(stdout, "")
	for i, candidate := range report.ProxyCandidates {
		status := "unverified"
		if candidate.Verified {
			status = "verified"
		} else if candidate.Listening {
			status = "listening only"
		}
		ui.Line(stdout, "%d. %s [%s] score=%d source=%s", i+1, candidate.URL, status, candidate.Score, candidate.Source)
		if opts.Verbose && len(candidate.Errors) > 0 {
			ui.Line(stdout, "   notes: %s", strings.Join(candidate.Errors, "; "))
		}
	}
	if report.SelectedProxy != "" {
		ui.Line(stdout, "")
		ui.Line(stdout, "Best proxy: %s", report.SelectedProxy)
	}
	return code
}

func runStatus(opts options, stdout, stderr io.Writer) int {
	s, err := state.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			ui.Line(stdout, "No codex-proxy state found. Run `codex-proxy fix` first.")
			return exitSuccess
		}
		ui.Line(stderr, "error: %v", err)
		return exitUnexpected
	}
	if opts.JSON {
		if err := ui.PrintJSON(stdout, s); err != nil {
			ui.Line(stderr, "error: %v", err)
			return exitUnexpected
		}
		return exitSuccess
	}
	ui.Line(stdout, "Codex proxy status")
	ui.Line(stdout, "")
	ui.Line(stdout, "Platform: %s", s.Platform)
	ui.Line(stdout, "Selected proxy: %s", s.SelectedProxy)
	ui.Line(stdout, "Codex GUI: %s", fallbackText(s.CodexGUIPath))
	ui.Line(stdout, "Codex CLI: %s", fallbackText(s.CodexCLIPath))
	ui.Line(stdout, "Persistent fix installed: %t", s.PersistentFixInstalled)
	ui.Line(stdout, "Persistent fix verified: %t", s.PersistentFixVerified)
	ui.Line(stdout, "Fallback launcher installed: %t", s.FallbackLauncherInstalled)
	ui.Line(stdout, "Original env snapshot saved: %t", s.OriginalPersistentEnv != nil)
	ui.Line(stdout, "Last fix time: %s", fallbackText(s.LastFixTime))
	if opts.Verbose && len(s.ManagedPaths) > 0 {
		ui.Line(stdout, "Managed paths:")
		for _, path := range s.ManagedPaths {
			ui.Line(stdout, "- %s", path)
		}
	}
	return exitSuccess
}

func runFix(opts options, stdout, stderr io.Writer) int {
	report, code, err := gatherReport(opts)
	if err != nil {
		ui.Line(stderr, "error: %v", err)
		return code
	}
	if !report.Codex.AnyFound() {
		ui.Line(stderr, "error: Codex installation not found")
		return exitCodexNotFound
	}
	if report.SelectedProxy == "" {
		ui.Line(stderr, "error: no usable local HTTP proxy detected")
		return exitNoProxy
	}

	proxyEnv := common.BuildProxyEnv(report.SelectedProxy, os.Getenv("NO_PROXY"))
	result, err := platform.Install(proxyEnv, report.Codex, opts.DryRun)
	if err != nil {
		ui.Line(stderr, "error: failed to install platform fix: %v", err)
		return exitPersistentInstallFail
	}
	s := &state.State{
		Platform:                  runtime.GOOS,
		SelectedProxy:             report.SelectedProxy,
		CodexGUIPath:              report.Codex.GUIPath,
		CodexCLIPath:              report.Codex.CLIPath,
		OriginalPersistentEnv:     result.OriginalEnv,
		PersistentFixInstalled:    result.PersistentInstalled,
		PersistentFixVerified:     result.PersistentVerified,
		FallbackLauncherInstalled: result.FallbackInstalled,
		ManagedPaths:              result.ManagedPaths,
	}
	if !opts.DryRun {
		if err := state.Save(s); err != nil {
			ui.Line(stderr, "error: failed to save state: %v", err)
			return exitUnexpected
		}
	}
	ui.Line(stdout, "Codex proxy fix completed.")
	ui.Line(stdout, "")
	ui.Line(stdout, "Detected:")
	ui.Line(stdout, "- OS: %s/%s", report.Platform.OS, report.Platform.Arch)
	ui.Line(stdout, "- Codex: %s", fallbackText(primaryCodexPath(report.Codex)))
	ui.Line(stdout, "- HTTP proxy: %s", report.SelectedProxy)
	ui.Line(stdout, "")
	ui.Line(stdout, "Installed:")
	ui.Line(stdout, "- Persistent fix: %t", result.PersistentInstalled)
	ui.Line(stdout, "- Persistent verified: %t", result.PersistentVerified)
	ui.Line(stdout, "- Fallback launcher: %t", result.FallbackInstalled)
	for _, note := range result.Notes {
		ui.Line(stdout, "- Note: %s", note)
	}
	ui.Line(stdout, "")
	ui.Line(stdout, "Next:")
	ui.Line(stdout, "- You can try launching Codex normally from the app icon.")
	ui.Line(stdout, "- If reconnecting still appears, run: codex-proxy launch")
	if opts.DryRun {
		ui.Line(stdout, "- Dry-run mode was enabled, so no files were written.")
	}
	if !result.PersistentInstalled {
		return exitPersistentInstallFail
	}
	if !result.PersistentVerified {
		return exitVerificationIncomplete
	}
	if !result.FallbackInstalled {
		return exitFallbackInstallFail
	}
	return exitSuccess
}

func runLaunch(opts options, extraArgs []string, stdout, stderr io.Writer) int {
	s, err := state.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		ui.Line(stderr, "error: %v", err)
		return exitUnexpected
	}
	var proxyURL string
	var codex detect.CodexInstall
	if s != nil {
		proxyURL = s.SelectedProxy
		codex = detect.CodexInstall{GUIPath: s.CodexGUIPath, CLIPath: s.CodexCLIPath}
	}
	if proxyURL == "" || !codex.AnyFound() {
		report, code, err := gatherReport(opts)
		if err != nil {
			ui.Line(stderr, "error: %v", err)
			return code
		}
		if report.SelectedProxy == "" {
			ui.Line(stderr, "error: no usable proxy found")
			return exitNoProxy
		}
		if !report.Codex.AnyFound() {
			ui.Line(stderr, "error: Codex installation not found")
			return exitCodexNotFound
		}
		proxyURL = report.SelectedProxy
		codex = report.Codex
	}
	target := primaryCodexPath(codex)
	if target == "" {
		ui.Line(stderr, "error: Codex installation not found")
		return exitCodexNotFound
	}
	envs := common.BuildProxyEnv(proxyURL, os.Getenv("NO_PROXY"))
	cmd := exec.Command(target, extraArgs...)
	cmd.Env = append(os.Environ(),
		"HTTP_PROXY="+envs.HTTPProxy,
		"HTTPS_PROXY="+envs.HTTPSProxy,
		"ALL_PROXY="+envs.ALLProxy,
		"NO_PROXY="+envs.NOProxy,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		ui.Line(stderr, "error: failed to launch Codex: %v", err)
		return exitUnexpected
	}
	ui.Line(stdout, "Launched Codex with proxy %s", proxyURL)
	return exitSuccess
}

func runUnset(opts options, stdout, stderr io.Writer) int {
	s, err := state.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			ui.Line(stdout, "Nothing to unset. No state file found.")
			return exitSuccess
		}
		ui.Line(stderr, "error: %v", err)
		return exitUnexpected
	}
	removed, err := platform.Uninstall(s, opts.DryRun)
	if err != nil {
		ui.Line(stderr, "error: failed to remove managed files: %v", err)
		return exitUnexpected
	}
	if !opts.DryRun {
		if err := state.Delete(); err != nil {
			ui.Line(stderr, "error: failed to remove state: %v", err)
			return exitUnexpected
		}
	}
	ui.Line(stdout, "Removed %d managed path(s).", len(removed))
	if opts.Verbose {
		for _, path := range removed {
			ui.Line(stdout, "- %s", path)
		}
	}
	if opts.DryRun {
		ui.Line(stdout, "Dry-run mode was enabled, so no files were removed.")
	}
	return exitSuccess
}

func gatherReport(opts options) (doctorReport, int, error) {
	report := doctorReport{
		Platform: detect.CurrentPlatform(),
		Codex:    detect.FindCodex(opts.CodexPath),
	}
	candidates := detect.DetectProxyCandidates(detect.ProxyOptions{ExplicitPort: opts.Port})
	report.ProxyCandidates = candidates
	best, ok := detect.BestProxy(candidates)
	if ok {
		report.SelectedProxy = best.URL
	}
	if s, err := state.Load(); err == nil {
		if s.PersistentFixInstalled {
			report.PersistentState = "installed"
		}
		report.FallbackAvailable = s.FallbackLauncherInstalled
	}
	if !report.Codex.AnyFound() {
		return report, exitCodexNotFound, nil
	}
	if report.SelectedProxy == "" {
		return report, exitNoProxy, nil
	}
	return report, exitSuccess, nil
}

func parseArgs(args []string) (string, options, []string, error) {
	var opts options
	if len(args) == 0 {
		return "", opts, nil, nil
	}
	cmd := args[0]
	rest := args[1:]
	var extra []string
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		switch {
		case arg == "--yes":
			opts.Yes = true
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--verbose":
			opts.Verbose = true
		case arg == "--json":
			opts.JSON = true
		case arg == "--port":
			if i+1 >= len(rest) {
				return "", opts, nil, fmt.Errorf("--port requires a value")
			}
			i++
			fmt.Sscanf(rest[i], "%d", &opts.Port)
		case strings.HasPrefix(arg, "--port="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--port="), "%d", &opts.Port)
		case arg == "--codex-path":
			if i+1 >= len(rest) {
				return "", opts, nil, fmt.Errorf("--codex-path requires a value")
			}
			i++
			opts.CodexPath = rest[i]
		case strings.HasPrefix(arg, "--codex-path="):
			opts.CodexPath = strings.TrimPrefix(arg, "--codex-path=")
		case arg == "--":
			extra = append(extra, rest[i+1:]...)
			i = len(rest)
		default:
			extra = append(extra, arg)
		}
	}
	return cmd, opts, extra, nil
}

func printUsage(w io.Writer) {
	ui.Line(w, "Usage: codex-proxy <command> [options]")
	ui.Line(w, "")
	ui.Line(w, "Commands:")
	ui.Line(w, "  fix       Detect and install the proxy fix")
	ui.Line(w, "  doctor    Diagnose Codex and local proxy detection")
	ui.Line(w, "  status    Show the last installed state")
	ui.Line(w, "  launch    Launch Codex with explicit proxy env")
	ui.Line(w, "  unset     Remove files managed by codex-proxy")
	ui.Line(w, "  version   Show tool version")
}

func primaryCodexPath(c detect.CodexInstall) string {
	if c.GUIPath != "" {
		return c.GUIPath
	}
	return c.CLIPath
}

func fallbackText(v string) string {
	if v == "" {
		return "n/a"
	}
	return v
}
