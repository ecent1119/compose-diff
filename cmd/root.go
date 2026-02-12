package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	version   = "1.0.0"
	colorMode string
)

var rootCmd = &cobra.Command{
	Use:   "compose-diff",
	Short: "Semantic diff for Docker Compose files",
	Long: color.New(color.FgCyan).Sprint(`
compose-diff - Semantic Docker Compose Diff Tool

`) + `Compare two Docker Compose configurations and see what actually changed:
services, images, environment variables, ports, volumes, and more.

` + color.New(color.FgYellow).Sprint(`Read-only analysis only. No Docker commands executed.
`),
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&colorMode, "color", "auto", "Color output: auto, always, never")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		switch colorMode {
		case "never":
			color.NoColor = true
		case "always":
			color.NoColor = false
		}
	}
}
