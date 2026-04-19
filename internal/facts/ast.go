package facts

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sync"
	"time"
)

type ASTFact struct {
	Fset *token.FileSet
	File *ast.File
}

var (
	astCacheMu sync.RWMutex
	astCache   = make(map[string]cachedAST)
)

type cachedAST struct {
	text      string
	modTime   time.Time
	sourceFile *ASTFact
}

const maxASTCacheEntries = 500

func cacheAST(filePath string, text string, modTime time.Time, astFact *ASTFact) {
	astCacheMu.Lock()
	defer astCacheMu.Unlock()

	if _, exists := astCache[filePath]; !exists && len(astCache) >= maxASTCacheEntries {
		var oldestKey string
		var oldestTime time.Time
		for key, cached := range astCache {
			if oldestKey == "" || cached.modTime.Before(oldestTime) {
				oldestKey = key
				oldestTime = cached.modTime
			}
		}
		if oldestKey != "" {
			delete(astCache, oldestKey)
		}
	}

	astCache[filePath] = cachedAST{
		text:       text,
		modTime:    modTime,
		sourceFile: astFact,
	}
}

type ASTFactProvider struct{}

func (p *ASTFactProvider) ID() string {
	return "fact.file.ast"
}

func (p *ASTFactProvider) Scope() Scope {
	return ScopeFile
}

func (p *ASTFactProvider) Requires() []string {
	return []string{"file.record", "file.text"}
}

func (p *ASTFactProvider) Provides() []string {
	return []string{"file.ast"}
}

func (p *ASTFactProvider) Supports(ctx ProviderContext) bool {
	return ctx.Scope == ScopeFile && ctx.File != nil
}

func (p *ASTFactProvider) Run(ctx ProviderContext) (map[string]any, error) {
	if ctx.File == nil {
		return map[string]any{}, nil
	}

	text := ctx.Runtime.Store.GetFileFact(ctx.File.Path, "file.text")
	if text == nil {
		return map[string]any{}, nil
	}

	textStr, ok := text.(string)
	if !ok {
		return map[string]any{}, nil
	}

	astCacheMu.RLock()
	cached, exists := astCache[ctx.File.AbsolutePath]
	astCacheMu.RUnlock()

	if exists {
		fileInfo, err := os.Stat(ctx.File.AbsolutePath)
		if err == nil && cached.text == textStr && fileInfo.ModTime().Equal(cached.modTime) {
			return map[string]any{
				"file.ast": cached.sourceFile,
			}, nil
		}
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, ctx.File.Path, textStr, parser.ParseComments)
	if err != nil {
		return map[string]any{}, nil
	}

	astFact := &ASTFact{
		Fset: fset,
		File: file,
	}

	fileInfo, err := os.Stat(ctx.File.AbsolutePath)
	if err != nil {
		return map[string]any{}, nil
	}

	cacheAST(ctx.File.AbsolutePath, textStr, fileInfo.ModTime(), astFact)

	return map[string]any{
		"file.ast": astFact,
	}, nil
}

func NewASTFactProvider() *ASTFactProvider {
	return &ASTFactProvider{}
}
