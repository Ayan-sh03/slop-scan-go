package rules

import (
	"fmt"

	"github.com/modem-dev/slop-scan-go/internal/facts"
	"github.com/modem-dev/slop-scan-go/internal/types"
)

type IgnoredErrorRule struct{}

func (r *IgnoredErrorRule) ID() string {
	return "defensive.ignored-error"
}

func (r *IgnoredErrorRule) Scope() types.Scope {
	return types.ScopeFile
}

func (r *IgnoredErrorRule) Requires() []string {
	return []string{"file.tryCatchSummaries"}
}

func (r *IgnoredErrorRule) Supports(ctx types.ProviderContext) bool {
	return ctx.Scope == types.ScopeFile && ctx.File != nil
}

func (r *IgnoredErrorRule) Family() string {
	return "defensive"
}

func (r *IgnoredErrorRule) Severity() string {
	return "strong"
}

func (r *IgnoredErrorRule) Evaluate(ctx types.ProviderContext) ([]types.RuleFinding, error) {
	summaries := ctx.Runtime.Store.GetFileFact(ctx.File.Path, "file.tryCatchSummaries")
	if summaries == nil {
		return []types.RuleFinding{}, nil
	}

	summaryList, ok := summaries.([]facts.TryCatchSummary)
	if !ok {
		return []types.RuleFinding{}, nil
	}

	flagged := filterIgnoredErrors(summaryList)
	if len(flagged) == 0 {
		return []types.RuleFinding{}, nil
	}

	evidence := make([]string, len(flagged))
	locations := make([]types.FindingLocation, len(flagged))

	for i, s := range flagged {
		evidence[i] = fmt.Sprintf("line %d: error check with empty body", s.Line)
		locations[i] = types.FindingLocation{
			Path: ctx.File.Path,
			Line: s.Line,
		}
	}

	return []types.RuleFinding{
		{
			Finding: types.Finding{
				RuleID:    "defensive.ignored-error",
				Family:    "defensive",
				Severity:  "strong",
				Scope:     types.ScopeFile,
				Path:      ctx.File.Path,
				Message:   fmt.Sprintf("Found %d ignored error check%s", len(flagged), pluralize(len(flagged))),
				Evidence:  evidence,
				Score:     float64(len(flagged)) * 2.0,
				Locations: locations,
			},
		},
	}, nil
}

func filterIgnoredErrors(summaries []facts.TryCatchSummary) []facts.TryCatchSummary {
	flagged := make([]facts.TryCatchSummary, 0)
	for _, s := range summaries {
		if s.HasErrorCheck && s.CatchIsEmpty {
			flagged = append(flagged, s)
		}
	}
	return flagged
}

func NewIgnoredErrorRule() *IgnoredErrorRule {
	return &IgnoredErrorRule{}
}
