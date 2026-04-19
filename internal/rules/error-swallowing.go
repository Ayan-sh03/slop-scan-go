package rules

import (
	"fmt"

	"github.com/modem-dev/slop-scan-go/internal/facts"
	"github.com/modem-dev/slop-scan-go/internal/types"
)

type ErrorSwallowingRule struct{}

func (r *ErrorSwallowingRule) ID() string {
	return "defensive.error-swallowing"
}

func (r *ErrorSwallowingRule) Scope() types.Scope {
	return types.ScopeFile
}

func (r *ErrorSwallowingRule) Requires() []string {
	return []string{"file.tryCatchSummaries"}
}

func (r *ErrorSwallowingRule) Supports(ctx types.ProviderContext) bool {
	return ctx.Scope == types.ScopeFile && ctx.File != nil
}

func (r *ErrorSwallowingRule) Family() string {
	return "defensive"
}

func (r *ErrorSwallowingRule) Severity() string {
	return "strong"
}

func (r *ErrorSwallowingRule) Evaluate(ctx types.ProviderContext) ([]types.RuleFinding, error) {
	summaries := ctx.Runtime.Store.GetFileFact(ctx.File.Path, "file.tryCatchSummaries")
	if summaries == nil {
		return []types.RuleFinding{}, nil
	}

	summaryList, ok := summaries.([]facts.TryCatchSummary)
	if !ok {
		return []types.RuleFinding{}, nil
	}

	flagged := filterErrorSwallowingSummaries(summaryList)
	if len(flagged) == 0 {
		return []types.RuleFinding{}, nil
	}

	evidence := make([]string, len(flagged))
	locations := make([]types.FindingLocation, len(flagged))
	totalScore := 0.0

	for i, s := range flagged {
		boundary := ""
		if len(s.BoundaryCategories) > 0 {
			boundary = s.BoundaryCategories[0]
		}
		evidence[i] = fmt.Sprintf("line %d: catch logs only, boundary=%s", s.Line, boundary)
		locations[i] = types.FindingLocation{
			Path: ctx.File.Path,
			Line: s.Line,
		}
		totalScore += scoreTryCatch(s)
	}

	if totalScore > 8 {
		totalScore = 8
	}

	return []types.RuleFinding{
		{
			Finding: types.Finding{
				RuleID:   "defensive.error-swallowing",
				Family:   "defensive",
				Severity: "strong",
				Scope:    types.ScopeFile,
				Path:     ctx.File.Path,
				Message:  fmt.Sprintf("Found %d log-and-continue catch block%s", len(flagged), pluralize(len(flagged))),
				Evidence: evidence,
				Score:    totalScore,
				Locations: locations,
			},
		},
	}, nil
}

func filterErrorSwallowingSummaries(summaries []facts.TryCatchSummary) []facts.TryCatchSummary {
	flagged := make([]facts.TryCatchSummary, 0)
	for _, s := range summaries {
		if s.HasErrorCheck && s.TryStatementCount <= 2 && s.CatchLogsOnly {
			flagged = append(flagged, s)
		}
	}
	return flagged
}

func scoreTryCatch(s facts.TryCatchSummary) float64 {
	score := 1.0

	if len(s.BoundaryCategories) > 0 {
		score += 0.5
	}

	if s.CatchLogsOnly {
		score += 1.0
	}

	if !s.CatchHasDefaultReturn && !s.CatchThrowsGeneric {
		score += 0.5
	}

	return score
}

func NewErrorSwallowingRule() *ErrorSwallowingRule {
	return &ErrorSwallowingRule{}
}
