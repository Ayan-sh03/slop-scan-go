package facts

import (
	"crypto/sha1"
	"fmt"
	"go/ast"
	"os"
	"strings"
	"sync"
	"time"
)

type FunctionSummary struct {
	Name                 string
	Receiver             string
	Line                 int
	ParameterCount       int
	IsAsync              bool
	HasDefer             bool
	HasRecover           bool
	HasPanic             bool
	StatementCount       int
	IsPassThroughWrapper bool
	PassThroughTarget    string
	HasErrorReturn       bool
	DuplicationFingerprint string
}

var (
	functionCacheMu sync.RWMutex
	functionCache   = make(map[string]cachedFunctions)
)

type cachedFunctions struct {
	text      string
	modTime   time.Time
	functions []FunctionSummary
}

const maxFunctionCacheEntries = 500

func cacheFunctions(filePath string, text string, modTime time.Time, functions []FunctionSummary) {
	functionCacheMu.Lock()
	defer functionCacheMu.Unlock()

	if _, exists := functionCache[filePath]; !exists && len(functionCache) >= maxFunctionCacheEntries {
		var oldestKey string
		var oldestTime time.Time
		for key, cached := range functionCache {
			if oldestKey == "" || cached.modTime.Before(oldestTime) {
				oldestKey = key
				oldestTime = cached.modTime
			}
		}
		if oldestKey != "" {
			delete(functionCache, oldestKey)
		}
	}

	functionCache[filePath] = cachedFunctions{
		text:      text,
		modTime:   modTime,
		functions: functions,
	}
}

type FunctionsFactProvider struct{}

func (p *FunctionsFactProvider) ID() string {
	return "fact.file.functions"
}

func (p *FunctionsFactProvider) Scope() Scope {
	return ScopeFile
}

func (p *FunctionsFactProvider) Requires() []string {
	return []string{"file.ast"}
}

func (p *FunctionsFactProvider) Provides() []string {
	return []string{"file.functionSummaries"}
}

func (p *FunctionsFactProvider) Supports(ctx ProviderContext) bool {
	return ctx.Scope == ScopeFile && ctx.File != nil
}

func (p *FunctionsFactProvider) Run(ctx ProviderContext) (map[string]any, error) {
	if ctx.File == nil {
		return map[string]any{"file.functionSummaries": []FunctionSummary{}}, nil
	}

	astFact := ctx.Runtime.Store.GetFileFact(ctx.File.Path, "file.ast")
	if astFact == nil {
		return map[string]any{"file.functionSummaries": []FunctionSummary{}}, nil
	}

	astData, ok := astFact.(*ASTFact)
	if !ok {
		return map[string]any{"file.functionSummaries": []FunctionSummary{}}, nil
	}

	functionCacheMu.RLock()
	cached, exists := functionCache[ctx.File.AbsolutePath]
	functionCacheMu.RUnlock()

	if exists {
		fileInfo, err := os.Stat(ctx.File.AbsolutePath)
		if err == nil && cached.text == astData.File.Comments[0].Text() && fileInfo.ModTime().Equal(cached.modTime) {
			return map[string]any{
				"file.functionSummaries": cached.functions,
			}, nil
		}
	}

	functions := collectFunctionSummaries(astData.File, ctx.File.Path)

	fileInfo, err := os.Stat(ctx.File.AbsolutePath)
	if err != nil {
		return map[string]any{"file.functionSummaries": functions}, nil
	}

	cacheFunctions(ctx.File.AbsolutePath, "", fileInfo.ModTime(), functions)

	return map[string]any{
		"file.functionSummaries": functions,
	}, nil
}

func collectFunctionSummaries(file *ast.File, filePath string) []FunctionSummary {
	functions := make([]FunctionSummary, 0)

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			summary := collectFuncDecl(node, file, filePath)
			if summary != nil {
				functions = append(functions, *summary)
			}
		case *ast.FuncLit:
			summary := collectFuncLit(node, file)
			if summary != nil {
				functions = append(functions, *summary)
			}
		}
		return true
	})

	return functions
}

func collectFuncDecl(node *ast.FuncDecl, file *ast.File, filePath string) *FunctionSummary {
	if node.Body == nil {
		return nil
	}

	name := node.Name.Name
	receiver := ""
	if node.Recv != nil && len(node.Recv.List) > 0 {
		if field := node.Recv.List[0]; field != nil {
			switch t := field.Type.(type) {
			case *ast.Ident:
				receiver = t.Name
			case *ast.StarExpr:
				if ident, ok := t.X.(*ast.Ident); ok {
					receiver = ident.Name
				}
			}
		}
	}

	paramCount := 0
	if node.Type.Params != nil {
		paramCount = len(node.Type.Params.List)
	}

	statementCount := countStatements(node.Body)
	isPassThrough, passThroughTarget := isPassThroughFunc(node)
	hasDefer := hasDeferStatement(node.Body)
	hasRecover := hasRecoverCall(node.Body)
	hasPanic := hasPanicCall(node.Body)
	hasErrorReturn := hasErrorReturn(node)

	var dupFingerprint string
	if !isTestFile(filePath) && statementCount >= 2 {
		dupFingerprint = buildFunctionFingerprint(node.Body, paramCount, statementCount, isPassThrough)
	}

	return &FunctionSummary{
		Name:                 name,
		Receiver:             receiver,
		Line:                 0,
		ParameterCount:       paramCount,
		IsAsync:              false,
		HasDefer:             hasDefer,
		HasRecover:           hasRecover,
		HasPanic:             hasPanic,
		StatementCount:       statementCount,
		IsPassThroughWrapper: isPassThrough,
		PassThroughTarget:    passThroughTarget,
		HasErrorReturn:       hasErrorReturn,
		DuplicationFingerprint: dupFingerprint,
	}
}

func collectFuncLit(node *ast.FuncLit, file *ast.File) *FunctionSummary {
	if node.Body == nil {
		return nil
	}

	paramCount := 0
	if node.Type.Params != nil {
		paramCount = len(node.Type.Params.List)
	}

	statementCount := countStatements(node.Body)
	isPassThrough, passThroughTarget := isPassThroughFuncLit(node)
	hasDefer := hasDeferStatement(node.Body)
	hasRecover := hasRecoverCall(node.Body)
	hasPanic := hasPanicCall(node.Body)

	return &FunctionSummary{
		Name:                 "anonymous",
		Receiver:             "",
		Line:                 0,
		ParameterCount:       paramCount,
		IsAsync:              false,
		HasDefer:             hasDefer,
		HasRecover:           hasRecover,
		HasPanic:             hasPanic,
		StatementCount:       statementCount,
		IsPassThroughWrapper: isPassThrough,
		PassThroughTarget:    passThroughTarget,
		HasErrorReturn:       false,
		DuplicationFingerprint: "",
	}
}

func countStatements(block *ast.BlockStmt) int {
	if block == nil {
		return 0
	}
	count := 0
	for _, stmt := range block.List {
		countStmt(stmt, &count)
	}
	return count
}

func countStmt(stmt ast.Stmt, count *int) {
	*count++
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		for _, inner := range s.List {
			countStmt(inner, count)
		}
	case *ast.IfStmt:
		if s.Body != nil {
			for _, inner := range s.Body.List {
				countStmt(inner, count)
			}
		}
		if s.Else != nil {
			if elseBlock, ok := s.Else.(*ast.BlockStmt); ok {
				for _, inner := range elseBlock.List {
					countStmt(inner, count)
				}
			}
		}
	case *ast.ForStmt:
		if s.Body != nil {
			for _, inner := range s.Body.List {
				countStmt(inner, count)
			}
		}
	case *ast.RangeStmt:
		if s.Body != nil {
			for _, inner := range s.Body.List {
				countStmt(inner, count)
			}
		}
	case *ast.SwitchStmt:
		if s.Body != nil {
			for _, inner := range s.Body.List {
				countStmt(inner, count)
			}
		}
	case *ast.TypeSwitchStmt:
		if s.Body != nil {
			for _, inner := range s.Body.List {
				countStmt(inner, count)
			}
		}
	case *ast.CaseClause:
		for _, inner := range s.Body {
			countStmt(inner, count)
		}
	case *ast.CommClause:
		for _, inner := range s.Body {
			countStmt(inner, count)
		}
	}
}

func isPassThroughFunc(node *ast.FuncDecl) (bool, string) {
	if len(node.Body.List) != 1 {
		return false, ""
	}

	retStmt, ok := node.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(retStmt.Results) == 0 {
		return false, ""
	}

	result := retStmt.Results[0]
	if callExpr, ok := result.(*ast.CallExpr); ok {
		return isPassThroughCall(callExpr, node), getCallTarget(callExpr)
	}

	if unaryExpr, ok := result.(*ast.UnaryExpr); ok && unaryExpr.Op.String() == "<-" {
		if callExpr, ok := unaryExpr.X.(*ast.CallExpr); ok {
			return isPassThroughCall(callExpr, node), getCallTarget(callExpr)
		}
	}

	return false, ""
}

func isPassThroughFuncLit(node *ast.FuncLit) (bool, string) {
	if len(node.Body.List) != 1 {
		return false, ""
	}

	retStmt, ok := node.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(retStmt.Results) == 0 {
		return false, ""
	}

	result := retStmt.Results[0]
	if callExpr, ok := result.(*ast.CallExpr); ok {
		return isPassThroughCallLit(callExpr, node), getCallTarget(callExpr)
	}

	return false, ""
}

func isPassThroughCall(callExpr *ast.CallExpr, funcDecl *ast.FuncDecl) bool {
	paramNames := make(map[string]bool)
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			for _, name := range param.Names {
				paramNames[name.Name] = true
			}
		}
	}

	for _, arg := range callExpr.Args {
		if ident, ok := arg.(*ast.Ident); ok {
			if !paramNames[ident.Name] {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

func isPassThroughCallLit(callExpr *ast.CallExpr, funcLit *ast.FuncLit) bool {
	paramNames := make(map[string]bool)
	if funcLit.Type.Params != nil {
		for _, param := range funcLit.Type.Params.List {
			for _, name := range param.Names {
				paramNames[name.Name] = true
			}
		}
	}

	for _, arg := range callExpr.Args {
		if ident, ok := arg.(*ast.Ident); ok {
			if !paramNames[ident.Name] {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

func getCallTarget(callExpr *ast.CallExpr) string {
	return getExpressionPath(callExpr.Fun)
}

func getExpressionPath(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		xPath := getExpressionPath(e.X)
		if xPath != "" {
			return xPath + "." + e.Sel.Name
		}
		return e.Sel.Name
	case *ast.CallExpr:
		return getExpressionPath(e.Fun)
	}
	return ""
}

func hasDeferStatement(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if _, ok := n.(*ast.DeferStmt); ok {
			found = true
			return false
		}
		return true
	})
	return found
}

func hasRecoverCall(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if ident, ok := callExpr.Fun.(*ast.Ident); ok && ident.Name == "recover" {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func hasPanicCall(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if ident, ok := callExpr.Fun.(*ast.Ident); ok && ident.Name == "panic" {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func hasErrorReturn(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Results == nil {
		return false
	}

	for _, field := range funcDecl.Type.Results.List {
		if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "error" {
			return true
		}
		if starExpr, ok := field.Type.(*ast.StarExpr); ok {
			if ident, ok := starExpr.X.(*ast.Ident); ok && ident.Name == "error" {
				return true
			}
		}
	}

	return false
}

func buildFunctionFingerprint(block *ast.BlockStmt, paramCount, statementCount int, isPassThrough bool) string {
	if isPassThrough || statementCount < 2 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("sync:%d:%d:", paramCount, statementCount))

	visitor := func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Ident:
			builder.WriteString("id:")
			builder.WriteString(node.Name)
		case *ast.BasicLit:
			builder.WriteString("literal:")
			builder.WriteString(node.Kind.String())
		default:
			builder.WriteString(fmt.Sprintf("%T", node))
		}
		return true
	}

	ast.Inspect(block, visitor)

	hash := sha1.Sum([]byte(builder.String()))
	return fmt.Sprintf("%x", hash[:8])
}

func isTestFile(filePath string) bool {
	return strings.HasSuffix(filePath, "_test.go")
}

func NewFunctionsFactProvider() *FunctionsFactProvider {
	return &FunctionsFactProvider{}
}
