package main

import (
	"fmt"
	"os"

	"github.com/dotcommander/crawler/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// Fatal errors bypass the standard logger: --quiet redirects the
		// logger to io.Discard, and a failing run must still say why
		// (exit code 1 with no diagnostics is unacceptable for pipelines).
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
