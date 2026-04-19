package reporters

import (
	"encoding/json"

	"github.com/modem-dev/slop-scan-go/internal/types"
)

type JSONReporter struct{}

func (r *JSONReporter) ID() string {
	return "json"
}

func (r *JSONReporter) Render(result types.AnalysisResult) (string, error) {
	output := map[string]any{
		"metadata": map[string]any{
			"schemaVersion": 1,
			"tool": map[string]any{
				"name":    "slop-scan-go",
				"version": "0.1.0",
			},
			"analyzedAt": result.AnalyzedAt,
		},
		"summary": map[string]any{
			"fileCount":         result.Summary.FileCount,
			"directoryCount":    result.Summary.DirectoryCount,
			"findingCount":      result.Summary.FindingCount,
			"repoScore":         result.RepoScore,
			"physicalLineCount": result.Summary.PhysicalLineCount,
			"logicalLineCount":  result.Summary.LogicalLineCount,
			"functionCount":    result.Summary.FunctionCount,
		},
		"files": result.Files,
		"findings": result.Findings,
		"fileScores": result.FileScores,
		"directoryScores": result.DirectoryScores,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func NewJSONReporter() *JSONReporter {
	return &JSONReporter{}
}
