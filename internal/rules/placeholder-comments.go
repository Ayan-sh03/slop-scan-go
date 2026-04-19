package rules

import (
	"fmt"

	"github.com/modem-dev/slop-scan-go/internal/facts"
	"github.com/modem-dev/slop-scan-go/internal/types"
)

type PlaceholderCommentsRule struct{}

func (r *PlaceholderCommentsRule) ID() string {
	return "comments.placeholder-comments"
}

func (r *PlaceholderCommentsRule) Scope() types.Scope {
	return types.ScopeFile
}

func (r *PlaceholderCommentsRule) Requires() []string {
	return []string{"file.comments"}
}

func (r *PlaceholderCommentsRule) Supports(ctx types.ProviderContext) bool {
	return ctx.Scope == types.ScopeFile && ctx.File != nil
}

func (r *PlaceholderCommentsRule) Family() string {
	return "comments"
}

func (r *PlaceholderCommentsRule) Severity() string {
	return "weak"
}

func (r *PlaceholderCommentsRule) Evaluate(ctx types.ProviderContext) ([]types.RuleFinding, error) {
	comments := ctx.Runtime.Store.GetFileFact(ctx.File.Path, "file.comments")
	if comments == nil {
		return []types.RuleFinding{}, nil
	}

	commentList, ok := comments.([]facts.CommentsSummary)
	if !ok {
		return []types.RuleFinding{}, nil
	}

	flagged := filterPlaceholderComments(commentList)
	if len(flagged) == 0 {
		return []types.RuleFinding{}, nil
	}

	evidence := make([]string, len(flagged))
	locations := make([]types.FindingLocation, len(flagged))
	byType := make(map[string]int)

	for i, c := range flagged {
		evidence[i] = fmt.Sprintf("%s", c.Text)
		locations[i] = types.FindingLocation{
			Path: ctx.File.Path,
			Line: 0,
		}
		byType[c.PlaceholderType]++
	}

	typeBreakdown := make([]string, 0, len(byType))
	for t, count := range byType {
		typeBreakdown = append(typeBreakdown, fmt.Sprintf("%s: %d", t, count))
	}

	return []types.RuleFinding{
		{
			Finding: types.Finding{
				RuleID:    "comments.placeholder-comments",
				Family:    "comments",
				Severity:  "weak",
				Scope:     types.ScopeFile,
				Path:      ctx.File.Path,
				Message:   fmt.Sprintf("Found %d placeholder comment%s (%s)", len(flagged), pluralize(len(flagged)), joinStrings(typeBreakdown, ", ")),
				Evidence:  evidence,
				Score:     float64(len(flagged)) * 0.25,
				Locations: locations,
			},
		},
	}, nil
}

func filterPlaceholderComments(comments []facts.CommentsSummary) []facts.CommentsSummary {
	flagged := make([]facts.CommentsSummary, 0)
	for _, c := range comments {
		if c.IsPlaceholder {
			flagged = append(flagged, c)
		}
	}
	return flagged
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func NewPlaceholderCommentsRule() *PlaceholderCommentsRule {
	return &PlaceholderCommentsRule{}
}
