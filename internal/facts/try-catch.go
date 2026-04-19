package facts

import (
	"go/ast"
	"strings"
	"sync"
	"time"
)

type TryCatchSummary struct {
	Line                      int
	EnclosingSymbol          string
	HasCatchClause           bool
	TryStatementCount        int
	CatchStatementCount      int
	CatchLogsOnly            bool
	CatchReturnsDefault      bool
	CatchHasLogging          bool
	CatchHasDefaultReturn    bool
	CatchIsEmpty             bool
	CatchHasComment          bool
	CatchThrowsGeneric       bool
	BoundaryCategories       []string
	BoundaryOperationPaths   []string
	IsFilesystemExistenceProbe bool
	TryResolvesLocalValues   bool
	IsDocumentedLocalFallback bool
	HasErrorCheck            bool
	ErrorVariableName        string
}

var (
	tryCatchCacheMu sync.RWMutex
	tryCatchCache   = make(map[string]cachedTryCatch)
)

type cachedTryCatch struct {
	text      string
	modTime   time.Time
	summaries []TryCatchSummary
}

const maxTryCatchCacheEntries = 500

func cacheTryCatch(filePath string, text string, modTime time.Time, summaries []TryCatchSummary) {
	tryCatchCacheMu.Lock()
	defer tryCatchCacheMu.Unlock()

	if _, exists := tryCatchCache[filePath]; !exists && len(tryCatchCache) >= maxTryCatchCacheEntries {
		var oldestKey string
		var oldestTime time.Time
		for key, cached := range tryCatchCache {
			if oldestKey == "" || cached.modTime.Before(oldestTime) {
				oldestKey = key
				oldestTime = cached.modTime
			}
		}
		if oldestKey != "" {
			delete(tryCatchCache, oldestKey)
		}
	}

	tryCatchCache[filePath] = cachedTryCatch{
		text:      text,
		modTime:   modTime,
		summaries: summaries,
	}
}

type TryCatchFactProvider struct{}

func (p *TryCatchFactProvider) ID() string {
	return "fact.file.tryCatch"
}

func (p *TryCatchFactProvider) Scope() Scope {
	return ScopeFile
}

func (p *TryCatchFactProvider) Requires() []string {
	return []string{"file.ast"}
}

func (p *TryCatchFactProvider) Provides() []string {
	return []string{"file.tryCatchSummaries"}
}

func (p *TryCatchFactProvider) Supports(ctx ProviderContext) bool {
	return ctx.Scope == ScopeFile && ctx.File != nil
}

func (p *TryCatchFactProvider) Run(ctx ProviderContext) (map[string]any, error) {
	if ctx.File == nil {
		return map[string]any{"file.tryCatchSummaries": []TryCatchSummary{}}, nil
	}

	astFact := ctx.Runtime.Store.GetFileFact(ctx.File.Path, "file.ast")
	if astFact == nil {
		return map[string]any{"file.tryCatchSummaries": []TryCatchSummary{}}, nil
	}

	astData, ok := astFact.(*ASTFact)
	if !ok {
		return map[string]any{"file.tryCatchSummaries": []TryCatchSummary{}}, nil
	}

	summaries := collectTryCatchSummaries(astData.File, ctx.File.Path)
	return map[string]any{
		"file.tryCatchSummaries": summaries,
	}, nil
}

func collectTryCatchSummaries(file *ast.File, filePath string) []TryCatchSummary {
	summaries := make([]TryCatchSummary, 0)

	ast.Inspect(file, func(n ast.Node) bool {
		if block, ok := n.(*ast.BlockStmt); ok {
			for _, stmt := range block.List {
				if ifStmt, ok := stmt.(*ast.IfStmt); ok {
					summary := analyzeErrorCheck(ifStmt, file)
					if summary != nil {
						summaries = append(summaries, *summary)
					}
				}
			}
		}
		return true
	})

	return summaries
}

func analyzeErrorCheck(ifStmt *ast.IfStmt, file *ast.File) *TryCatchSummary {
	if ifStmt.Cond == nil || ifStmt.Body == nil {
		return nil
	}

	binaryExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	if !isErrorCheck(binaryExpr) {
		return nil
	}

	errorVarName := extractErrorVariableName(binaryExpr)
	boundaryCategories, boundaryPaths := analyzeErrorBoundary(ifStmt.Body)

	hasLogging := hasLoggingInBlock(ifStmt.Body)
	hasReturn := hasReturnInBlock(ifStmt.Body)
	hasDefaultReturn := hasDefaultReturnInBlock(ifStmt.Body)

	return &TryCatchSummary{
		Line:                    0,
		EnclosingSymbol:         "",
		HasCatchClause:          true,
		TryStatementCount:        1,
		CatchStatementCount:     len(ifStmt.Body.List),
		CatchLogsOnly:          hasLogging && !hasReturn,
		CatchReturnsDefault:    hasDefaultReturn,
		CatchHasLogging:        hasLogging,
		CatchHasDefaultReturn:  hasDefaultReturn,
		CatchIsEmpty:           len(ifStmt.Body.List) == 0,
		CatchHasComment:        false,
		CatchThrowsGeneric:     false,
		BoundaryCategories:      boundaryCategories,
		BoundaryOperationPaths:  boundaryPaths,
		HasErrorCheck:          true,
		ErrorVariableName:      errorVarName,
	}
}

func isErrorCheck(expr *ast.BinaryExpr) bool {
	if expr.Op.String() != "!=" && expr.Op.String() != "==" {
		return false
	}

	ident, ok := expr.X.(*ast.Ident)
	if !ok {
		return false
	}

	if ident.Name != "err" && ident.Name != "error" && ident.Name != "e" {
		return false
	}

	if selExpr, ok := expr.Y.(*ast.SelectorExpr); ok {
		if xIdent, ok := selExpr.X.(*ast.Ident); ok {
			return xIdent.Name == "nil"
		}
	}

	if identY, ok := expr.Y.(*ast.Ident); ok {
		return identY.Name == "nil"
	}

	return false
}

func extractErrorVariableName(expr *ast.BinaryExpr) string {
	if ident, ok := expr.X.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func analyzeErrorBoundary(block *ast.BlockStmt) ([]string, []string) {
	categories := make(map[string]bool)
	paths := make(map[string]bool)

	ast.Inspect(block, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			path := getCallPath(callExpr)
			if path != "" {
				paths[path] = true

				if isFileSystemOperation(path) {
					categories["filesystem"] = true
				}
				if isNetworkOperation(path) {
					categories["network"] = true
				}
				if isDatabaseOperation(path) {
					categories["database"] = true
				}
				if isProcessOperation(path) {
					categories["process"] = true
				}
			}
		}
		return true
	})

	categoryList := make([]string, 0, len(categories))
	for cat := range categories {
		categoryList = append(categoryList, cat)
	}

	pathList := make([]string, 0, len(paths))
	for p := range paths {
		pathList = append(pathList, p)
	}

	return categoryList, pathList
}

func getCallPath(callExpr *ast.CallExpr) string {
	return getExpressionPath(callExpr.Fun)
}

func isFileSystemOperation(path string) bool {
	fsPackages := []string{"os", "io", "io/ioutil"}
	for _, pkg := range fsPackages {
		if strings.HasPrefix(path, pkg+".") {
			return true
		}
	}
	return false
}

func isNetworkOperation(path string) bool {
	netPackages := []string{"net", "net/http", "http"}
	for _, pkg := range netPackages {
		if strings.HasPrefix(path, pkg+".") {
			return true
		}
	}
	return false
}

func isDatabaseOperation(path string) bool {
	dbPackages := []string{"database/sql", "sql"}
	for _, pkg := range dbPackages {
		if strings.HasPrefix(path, pkg+".") {
			return true
		}
	}
	return false
}

func isProcessOperation(path string) bool {
	procPackages := []string{"os/exec", "exec"}
	for _, pkg := range procPackages {
		if strings.HasPrefix(path, pkg+".") {
			return true
		}
	}
	return false
}

func hasLoggingInBlock(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if path := getCallPath(callExpr); path != "" {
				if strings.Contains(path, "log.") || strings.Contains(path, "fmt.Println") ||
					strings.Contains(path, "fmt.Printf") || strings.Contains(path, "fmt.Fprintf") {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

func hasReturnInBlock(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if _, ok := n.(*ast.ReturnStmt); ok {
			found = true
			return false
		}
		return true
	})
	return found
}

func hasDefaultReturnInBlock(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if retStmt, ok := n.(*ast.ReturnStmt); ok {
			if len(retStmt.Results) == 1 {
				if lit, ok := retStmt.Results[0].(*ast.BasicLit); ok {
					if lit.Kind.String() == "NIL" {
						found = true
						return false
					}
				}
				if ident, ok := retStmt.Results[0].(*ast.Ident); ok {
					if ident.Name == "nil" {
						found = true
						return false
					}
				}
			}
			if len(retStmt.Results) >= 1 {
				if last := retStmt.Results[len(retStmt.Results)-1]; last != nil {
					if ident, ok := last.(*ast.Ident); ok && ident.Name == "nil" {
						found = true
						return false
					}
				}
			}
		}
		return true
	})
	return found
}

func NewTryCatchFactProvider() *TryCatchFactProvider {
	return &TryCatchFactProvider{}
}
