package rules

import (
	"fmt"

	"github.com/modem-dev/slop-scan-go/internal/facts"
	"github.com/modem-dev/slop-scan-go/internal/types"
)

type ErrorObscuringRule struct{}

func (r *ErrorObscuringRule) ID() string {
	return "defensive.error-obscuring"
}

func (r *ErrorObscuringRule) Scope() types.Scope {
	return types.ScopeFile
}

func (r *ErrorObscuringRule) Requires() []string {
	return []string{"file.tryCatchSummaries"}
}

func (r *ErrorObscuringRule) Supports(ctx types.ProviderContext) bool {
	return ctx.Scope == types.ScopeFile && ctx.File != nil
}

func (r *ErrorObscuringRule) Family() string {
	return "defensive"
}

func (r *ErrorObscuringRule) Severity() string {
	return "medium"
}

func (r *ErrorObscuringRule) Evaluate(ctx types.ProviderContext) ([]types.RuleFinding, error) {
	summaries := ctx.Runtime.Store.GetFileFact(ctx.File.Path, "file.tryCatchSummaries")
	if summaries == nil {
		return []types.RuleFinding{}, nil
	}

	summaryList, ok := summaries.([]facts.TryCatchSummary)
	if !ok {
		return []types.RuleFinding{}, nil
	}

	flagged := filterErrorObscuringSummaries(summaryList)
	if len(flagged) == 0 {
		return []types.RuleFinding{}, nil
	}

	evidence := make([]string, len(flagged))
	locations := make([]types.FindingLocation, len(flagged))
	totalScore := 0.0

	for i, s := range flagged {
		evidence[i] = fmt.Sprintf("line %d: returns default on error", s.Line)
		locations[i] = types.FindingLocation{
			Path: ctx.File.Path,
			Line: s.Line,
		}
		totalScore += 1.5
	}

	return []types.RuleFinding{
		{
			Finding: types.Finding{
				RuleID:   "defensive.error-obscuring",
				Family:   "defensive",
				Severity: "medium",
				Scope:    types.ScopeFile,
				Path:     ctx.File.Path,
				Message:  fmt.Sprintf("Found %d error-obscuring default return%s", len(flagged), pluralize(len(flagged))),
				Evidence: evidence,
				Score:    totalScore,
				Locations: locations,
			},
		},
	}, nil
}

func filterErrorObscuringSummaries(summaries []facts.TryCatchSummary) []facts.TryCatchSummary {
	flagged := make([]facts.TryCatchSummary, 0)
	for _, s := range summaries {
		if s.HasErrorCheck && s.CatchReturnsDefault && !s.IsFilesystemExistenceProbe {
			flagged = append(flagged, s)
		}
	}
	return flagged
}

func NewErrorObscuringRule() *ErrorObscuringRule {
	return &ErrorObscuringRule{}
}
