package rules

import (
	"fmt"

	"github.com/modem-dev/slop-scan-go/internal/facts"
	"github.com/modem-dev/slop-scan-go/internal/types"
)

type PassThroughWrappersRule struct{}

func (r *PassThroughWrappersRule) ID() string {
	return "structure.pass-through-wrappers"
}

func (r *PassThroughWrappersRule) Scope() types.Scope {
	return types.ScopeFile
}

func (r *PassThroughWrappersRule) Requires() []string {
	return []string{"file.functionSummaries"}
}

func (r *PassThroughWrappersRule) Supports(ctx types.ProviderContext) bool {
	return ctx.Scope == types.ScopeFile && ctx.File != nil
}

func (r *PassThroughWrappersRule) Family() string {
	return "structure"
}

func (r *PassThroughWrappersRule) Severity() string {
	return "weak"
}

func (r *PassThroughWrappersRule) Evaluate(ctx types.ProviderContext) ([]types.RuleFinding, error) {
	summaries := ctx.Runtime.Store.GetFileFact(ctx.File.Path, "file.functionSummaries")
	if summaries == nil {
		return []types.RuleFinding{}, nil
	}

	summaryList, ok := summaries.([]facts.FunctionSummary)
	if !ok {
		return []types.RuleFinding{}, nil
	}

	flagged := filterPassThroughWrappers(summaryList)
	if len(flagged) == 0 {
		return []types.RuleFinding{}, nil
	}

	evidence := make([]string, len(flagged))
	locations := make([]types.FindingLocation, len(flagged))

	for i, f := range flagged {
		target := f.PassThroughTarget
		if target == "" {
			target = "unknown"
		}
		evidence[i] = fmt.Sprintf("line %d: %s -> %s", f.Line, f.Name, target)
		locations[i] = types.FindingLocation{
			Path: ctx.File.Path,
			Line: f.Line,
		}
	}

	return []types.RuleFinding{
		{
			Finding: types.Finding{
				RuleID:    "structure.pass-through-wrappers",
				Family:    "structure",
				Severity:  "weak",
				Scope:     types.ScopeFile,
				Path:      ctx.File.Path,
				Message:   fmt.Sprintf("Found %d pass-through wrapper function%s", len(flagged), pluralize(len(flagged))),
				Evidence:  evidence,
				Score:     float64(len(flagged)) * 0.5,
				Locations: locations,
			},
		},
	}, nil
}

func filterPassThroughWrappers(summaries []facts.FunctionSummary) []facts.FunctionSummary {
	flagged := make([]facts.FunctionSummary, 0)
	for _, f := range summaries {
		if f.IsPassThroughWrapper && f.StatementCount == 1 {
			flagged = append(flagged, f)
		}
	}
	return flagged
}

func NewPassThroughWrappersRule() *PassThroughWrappersRule {
	return &PassThroughWrappersRule{}
}
