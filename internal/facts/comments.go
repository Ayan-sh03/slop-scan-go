package facts

import (
	"go/ast"
	"strings"
)

type CommentsSummary struct {
	Text            string
	IsPlaceholder   bool
	PlaceholderType string
}

type CommentsFactProvider struct{}

func (p *CommentsFactProvider) ID() string {
	return "fact.file.comments"
}

func (p *CommentsFactProvider) Scope() Scope {
	return ScopeFile
}

func (p *CommentsFactProvider) Requires() []string {
	return []string{"file.ast"}
}

func (p *CommentsFactProvider) Provides() []string {
	return []string{"file.comments"}
}

func (p *CommentsFactProvider) Supports(ctx ProviderContext) bool {
	return ctx.Scope == ScopeFile && ctx.File != nil
}

func (p *CommentsFactProvider) Run(ctx ProviderContext) (map[string]any, error) {
	if ctx.File == nil {
		return map[string]any{"file.comments": []CommentsSummary{}}, nil
	}

	astFact := ctx.Runtime.Store.GetFileFact(ctx.File.Path, "file.ast")
	if astFact == nil {
		return map[string]any{"file.comments": []CommentsSummary{}}, nil
	}

	astData, ok := astFact.(*ASTFact)
	if !ok {
		return map[string]any{"file.comments": []CommentsSummary{}}, nil
	}

	summaries := collectComments(astData.File)
	return map[string]any{
		"file.comments": summaries,
	}, nil
}

func collectComments(file *ast.File) []CommentsSummary {
	summaries := make([]CommentsSummary, 0)

	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if text == "" {
				continue
			}

			isPlaceholder, placeholderType := isPlaceholderComment(text)

			summaries = append(summaries, CommentsSummary{
				Text:            text,
				IsPlaceholder:   isPlaceholder,
				PlaceholderType: placeholderType,
			})
		}
	}

	return summaries
}

func isPlaceholderComment(text string) (bool, string) {
	upperText := strings.ToUpper(text)

	placeholderTypes := []string{
		"TODO",
		"FIXME",
		"XXX",
		"HACK",
		"NOTE",
		"REVIEW",
		"OPTIMIZE",
		"REFACTOR",
	}

	for _, pt := range placeholderTypes {
		if strings.HasPrefix(upperText, pt) || strings.Contains(upperText, " "+pt) {
			return true, pt
		}
	}

	if strings.Contains(upperText, "IMPLEMENT") && strings.Contains(upperText, "LATER") {
		return true, "IMPLEMENT_LATER"
	}

	if strings.Contains(upperText, "COME BACK") {
		return true, "COME_BACK"
	}

	return false, ""
}

func NewCommentsFactProvider() *CommentsFactProvider {
	return &CommentsFactProvider{}
}
