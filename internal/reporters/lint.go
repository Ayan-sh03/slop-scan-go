package reporters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/modem-dev/slop-scan-go/internal/types"
)

type LintReporter struct{}

func (r *LintReporter) ID() string {
	return "lint"
}

func (r *LintReporter) Render(result types.AnalysisResult) (string, error) {
	var builder strings.Builder

	if len(result.Findings) == 0 {
		return "", nil
	}

	type lintFinding struct {
		Path     string
		Line     int
		Severity string
		Message  string
		RuleID   string
	}

	findings := make([]lintFinding, 0)
	for _, f := range result.Findings {
		for _, loc := range f.Locations {
			findings = append(findings, lintFinding{
				Path:     loc.Path,
				Line:     loc.Line,
				Severity: f.Severity,
				Message:  f.Message,
				RuleID:   f.RuleID,
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Line < findings[j].Line
	})

	for _, f := range findings {
		severity := strings.ToUpper(f.Severity)
		builder.WriteString(fmt.Sprintf("%s  %s:%d  %s  [%s]\n", severity, f.Path, f.Line, f.Message, f.RuleID))
	}

	return builder.String(), nil
}

func NewLintReporter() *LintReporter {
	return &LintReporter{}
}
