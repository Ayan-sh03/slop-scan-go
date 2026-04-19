package main

import (
	"fmt"
	"os"

	"github.com/modem-dev/slop-scan-go/internal/config"
	"github.com/modem-dev/slop-scan-go/internal/core"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "slop-scan",
		Short: "Deterministic CLI for finding AI-associated slop patterns in Go repositories",
		Long:  `slop-scan is a deterministic CLI tool that detects AI-associated "slop" patterns in Go code repositories.`,
	}
)

var (
	scanPath   string
	jsonOutput bool
	lintOutput bool
)

var scanCmd = &cobra.Command{
	Use:   "scan <path>",
	Short: "Scan a repository for slop patterns",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}

		cfg, err := config.LoadConfig(path)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		registry := core.CreateDefaultRegistry()

		result, err := core.AnalyzeRepository(path, cfg, registry, core.AnalyzeRepositoryOptions{})
		if err != nil {
			return fmt.Errorf("analysis failed: %w", err)
		}

		var outputFormat string
		if lintOutput {
			outputFormat = "lint"
		} else if jsonOutput {
			outputFormat = "json"
		} else {
			outputFormat = "text"
		}

		reporter, err := registry.GetReporter(outputFormat)
		if err != nil {
			return fmt.Errorf("failed to get reporter: %w", err)
		}

		output, err := reporter.Render(*result)
		if err != nil {
			return fmt.Errorf("failed to render report: %w", err)
		}

		fmt.Print(output)
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("slop-scan-go version 0.1.0")
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(versionCmd)

	scanCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	scanCmd.Flags().BoolVar(&lintOutput, "lint", false, "Output in lint format")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
