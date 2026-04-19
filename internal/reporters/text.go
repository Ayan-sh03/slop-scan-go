package reporters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/modem-dev/slop-scan-go/internal/types"
)

type TextReporter struct{}

func (r *TextReporter) ID() string {
	return "text"
}

func (r *TextReporter) Render(result types.AnalysisResult) (string, error) {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("\nSlop Scan Results for %s\n", result.RootDir))
	builder.WriteString(strings.Repeat("=", 60))
	builder.WriteString("\n\n")

	builder.WriteString("Summary\n")
	builder.WriteString(strings.Repeat("-", 40))
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("Files analyzed:      %d\n", result.Summary.FileCount))
	builder.WriteString(fmt.Sprintf("Directories:         %d\n", result.Summary.DirectoryCount))
	builder.WriteString(fmt.Sprintf("Total findings:      %d\n", result.Summary.FindingCount))
	builder.WriteString(fmt.Sprintf("Repo score:          %.2f\n", result.RepoScore))
	builder.WriteString(fmt.Sprintf("Physical lines:      %d\n", result.Summary.PhysicalLineCount))
	builder.WriteString(fmt.Sprintf("Logical lines:       %d\n", result.Summary.LogicalLineCount))
	builder.WriteString(fmt.Sprintf("Functions:           %d\n", result.Summary.FunctionCount))

	if result.Summary.Normalized.HasData {
		builder.WriteString("\nNormalized Metrics\n")
		builder.WriteString(strings.Repeat("-", 40))
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("Score per file:      %.2f\n", result.Summary.Normalized.ScorePerFile))
		builder.WriteString(fmt.Sprintf("Score per KLOC:      %.2f\n", result.Summary.Normalized.ScorePerKloc))
		builder.WriteString(fmt.Sprintf("Score per function:  %.2f\n", result.Summary.Normalized.ScorePerFunction))
		builder.WriteString(fmt.Sprintf("Findings per file:   %.2f\n", result.Summary.Normalized.FindingsPerFile))
		builder.WriteString(fmt.Sprintf("Findings per KLOC:   %.2f\n", result.Summary.Normalized.FindingsPerKloc))
	}

	if len(result.Findings) > 0 {
		builder.WriteString("\n\nFindings by Severity\n")
		builder.WriteString(strings.Repeat("-", 40))
		builder.WriteString("\n")

		findingsByFamily := make(map[string][]types.Finding)
		for _, f := range result.Findings {
			findingsByFamily[f.Family] = append(findingsByFamily[f.Family], f)
		}

		families := make([]string, 0, len(findingsByFamily))
		for family := range findingsByFamily {
			families = append(families, family)
		}
		sort.Strings(families)

		for _, family := range families {
			findings := findingsByFamily[family]
			sort.Slice(findings, func(i, j int) bool {
				severityOrder := map[string]int{"strong": 0, "medium": 1, "weak": 2}
				if severityOrder[findings[i].Severity] != severityOrder[findings[j].Severity] {
					return severityOrder[findings[i].Severity] < severityOrder[findings[j].Severity]
				}
				return findings[i].Score > findings[j].Score
			})

			builder.WriteString(fmt.Sprintf("\n%s (%d findings)\n", strings.ToUpper(family), len(findings)))
			for _, finding := range findings {
				builder.WriteString(fmt.Sprintf("\n  %s  %s  %s\n", finding.Severity, finding.Message, finding.RuleID))
				for _, loc := range finding.Locations {
					builder.WriteString(fmt.Sprintf("    at %s:%d\n", loc.Path, loc.Line))
				}
			}
		}
	}

	if len(result.FileScores) > 0 {
		builder.WriteString("\n\nFile Hotspots (top 10)\n")
		builder.WriteString(strings.Repeat("-", 40))
		builder.WriteString("\n")

		max := 10
		if len(result.FileScores) < max {
			max = len(result.FileScores)
		}

		for i := 0; i < max; i++ {
			score := result.FileScores[i]
			builder.WriteString(fmt.Sprintf("  %6.2f  %s (%d findings)\n", score.Score, score.Path, score.FindingCount))
		}
	}

	if len(result.DirectoryScores) > 0 {
		builder.WriteString("\n\nDirectory Hotspots (top 10)\n")
		builder.WriteString(strings.Repeat("-", 40))
		builder.WriteString("\n")

		max := 10
		if len(result.DirectoryScores) < max {
			max = len(result.DirectoryScores)
		}

		for i := 0; i < max; i++ {
			score := result.DirectoryScores[i]
			builder.WriteString(fmt.Sprintf("  %6.2f  %s (%d findings)\n", score.Score, score.Path, score.FindingCount))
		}
	}

	builder.WriteString("\n")
	return builder.String(), nil
}

func NewTextReporter() *TextReporter {
	return &TextReporter{}
}
