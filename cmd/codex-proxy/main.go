package main

import (
	"fmt"
	"os"

	"github.com/boyzcl/codex-proxy-fix/internal/app"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	code := app.Run(os.Args[1:], os.Stdout, os.Stderr, version, commit)
	if code != 0 {
		os.Exit(code)
	}
	fmt.Fprintln(os.Stdout)
}
