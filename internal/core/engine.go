package core

import (
	"os"
	"sort"
	"time"

	"github.com/modem-dev/slop-scan-go/internal/config"
	"github.com/modem-dev/slop-scan-go/internal/discovery"
	"github.com/modem-dev/slop-scan-go/internal/facts"
	"github.com/modem-dev/slop-scan-go/internal/types"
)

type AnalyzeRepositoryOptions struct {
	Hooks *AnalyzeRepositoryHooks
}

type AnalyzeRepositoryHooks struct {
	OnFileAnalyzed func(file FileRecord, store *FactStore)
	OnFileReleased func(file FileRecord, store *FactStore)
}

func AnalyzeRepository(rootDir string, cfg *config.AnalyzerConfig, registry *Registry, options AnalyzeRepositoryOptions) (*AnalysisResult, error) {
	discoveryFiles, discoveryDirs, err := discovery.DiscoverSourceFiles(rootDir, cfg, registry.GetLanguages())
	if err != nil {
		return nil, err
	}

	files := make([]FileRecord, len(discoveryFiles))
	for i, f := range discoveryFiles {
		files[i] = FileRecord{
			Path:              f.Path,
			AbsolutePath:      f.AbsolutePath,
			Extension:         f.Extension,
			LineCount:          0,
			LogicalLineCount:   0,
			LanguageID:        f.LanguageID,
		}
	}

	directories := make([]DirectoryRecord, len(discoveryDirs))
	for i, d := range discoveryDirs {
		directories[i] = DirectoryRecord{
			Path:      d.Path,
			FilePaths: append([]string(nil), d.FilePaths...),
		}
	}

	store := NewFactStore()
	runtime := AnalyzerRuntime{
		RootDir:      rootDir,
		Config:       cfg,
		Files:        files,
		Directories:  directories,
		Store:        store,
	}

	fileProviders := make([]FactProvider, 0)
	directoryProviders := make([]FactProvider, 0)
	repoProviders := make([]FactProvider, 0)

	for _, provider := range registry.GetFactProviders() {
		switch provider.Scope() {
		case ScopeFile:
			fileProviders = append(fileProviders, provider)
		case ScopeDirectory:
			directoryProviders = append(directoryProviders, provider)
		case ScopeRepo:
			repoProviders = append(repoProviders, provider)
		}
	}

	fileRules := make([]RulePlugin, 0)
	directoryRules := make([]RulePlugin, 0)
	repoRules := make([]RulePlugin, 0)

	for _, rule := range registry.GetRules() {
		switch rule.Scope() {
		case ScopeFile:
			fileRules = append(fileRules, rule)
		case ScopeDirectory:
			directoryRules = append(directoryRules, rule)
		case ScopeRepo:
			repoRules = append(repoRules, rule)
		}
	}

	fileBaseFacts := []string{"file.record", "file.text", "file.lineCount", "file.logicalLineCount"}
	orderedFileProviders := OrderFactProviders(fileProviders, fileBaseFacts)

	fileDerivedFacts := make([]string, 0)
	for _, provider := range orderedFileProviders {
		fileDerivedFacts = append(fileDerivedFacts, provider.Provides()...)
	}

	directoryBaseFacts := append([]string{"directory.record"}, fileBaseFacts...)
	directoryBaseFacts = append(directoryBaseFacts, fileDerivedFacts...)
	orderedDirectoryProviders := OrderFactProviders(directoryProviders, directoryBaseFacts)

	directoryDerivedFacts := make([]string, 0)
	for _, provider := range orderedDirectoryProviders {
		directoryDerivedFacts = append(directoryDerivedFacts, provider.Provides()...)
	}

	repoBaseFacts := append([]string{"repo.files", "repo.directories", "directory.record"}, fileBaseFacts...)
	repoBaseFacts = append(repoBaseFacts, fileDerivedFacts...)
	repoBaseFacts = append(repoBaseFacts, directoryDerivedFacts...)
	orderedRepoProviders := OrderFactProviders(repoProviders, repoBaseFacts)

	availableFacts := append([]string{}, fileBaseFacts...)
	availableFacts = append(availableFacts, fileDerivedFacts...)
	availableFacts = append(availableFacts, "directory.record")
	availableFacts = append(availableFacts, directoryDerivedFacts...)
	availableFacts = append(availableFacts, "repo.files", "repo.directories")
	for _, provider := range orderedRepoProviders {
		availableFacts = append(availableFacts, provider.Provides()...)
	}

	ruleRequirements := make([]struct {
		ID       string
		Requires []string
	}, len(registry.GetRules()))

	for i, rule := range registry.GetRules() {
		ruleRequirements[i] = struct {
			ID       string
			Requires []string
		}{ID: rule.ID(), Requires: rule.Requires()}
	}

	ValidateRuleRequirements(ruleRequirements, availableFacts)

	store.SetRepoFact("repo.files", discoveryFiles)
	store.SetRepoFact("repo.directories", discoveryDirs)

	for _, directory := range directories {
		store.SetDirectoryFact(directory.Path, "directory.record", directory)
	}

	durableFileFacts := map[string]bool{
		"file.lineCount":        true,
		"file.logicalLineCount":  true,
	}

	findings := make([]Finding, 0)

	for i := range files {
		file := &files[i]
		text, err := os.ReadFile(file.AbsolutePath)
		if err != nil {
			continue
		}

		textStr := string(text)
		lines := countLines(textStr)
		logicalLines := countLogicalLines(textStr, file.Path)

		file.LineCount = lines
		file.LogicalLineCount = logicalLines

		store.SetFileFacts(file.Path, map[string]any{
			"file.record":           file,
			"file.text":             textStr,
			"file.lineCount":        lines,
			"file.logicalLineCount": logicalLines,
		})

		ctx := ProviderContext{
			Scope:   ScopeFile,
			File:    file,
			Runtime: runtime,
		}

		for _, provider := range orderedFileProviders {
			if !provider.Supports(ctx) {
				continue
			}

			providerFacts, err := provider.Run(ctx)
			if err != nil {
				continue
			}

			for factID, value := range providerFacts {
				store.SetFileFact(file.Path, factID, value)
			}
		}

		immediateFileRules := make([]RulePlugin, 0)
		for _, rule := range fileRules {
			allFileFacts := true
			for _, req := range rule.Requires() {
				if !hasPrefix(req, "file.") {
					allFileFacts = false
					break
				}
			}
			if allFileFacts {
				immediateFileRules = append(immediateFileRules, rule)
			}
		}

		for _, rule := range immediateFileRules {
			ruleCfg := resolveRuleConfig(cfg, rule.ID(), nil)
			if !ruleCfg.Enabled {
				continue
			}

			ruleCtx := ProviderContext{
				Scope:      ScopeFile,
				File:       file,
				Runtime:    runtime,
				RuleConfig: &ruleCfg,
			}

			if !rule.Supports(ruleCtx) {
				continue
			}

			ruleFindings, err := rule.Evaluate(ruleCtx)
			if err != nil {
				continue
			}

			for _, rf := range ruleFindings {
				rf.Score *= ruleCfg.Weight
				findings = append(findings, rf.Finding)
			}
		}

		if options.Hooks != nil && options.Hooks.OnFileAnalyzed != nil {
			options.Hooks.OnFileAnalyzed(*file, store)
		}

		store.RetainFileFacts(file.Path, durableFileFacts)

		if options.Hooks != nil && options.Hooks.OnFileReleased != nil {
			options.Hooks.OnFileReleased(*file, store)
		}
	}

	for i := range directories {
		directory := &directories[i]
		ctx := ProviderContext{
			Scope:      ScopeDirectory,
			Directory:  directory,
			Runtime:    runtime,
		}

		for _, provider := range orderedDirectoryProviders {
			if !provider.Supports(ctx) {
				continue
			}

			providerFacts, err := provider.Run(ctx)
			if err != nil {
				continue
			}

			for factID, value := range providerFacts {
				store.SetDirectoryFact(directory.Path, factID, value)
			}
		}
	}

	repoCtx := ProviderContext{
		Scope:   ScopeRepo,
		Runtime: runtime,
	}

	for _, provider := range orderedRepoProviders {
		if !provider.Supports(repoCtx) {
			continue
		}

		providerFacts, err := provider.Run(repoCtx)
		if err != nil {
			continue
		}

		for factID, value := range providerFacts {
			store.SetRepoFact(factID, value)
		}
	}

	delayedFileRules := make([]RulePlugin, 0)
	for _, rule := range fileRules {
		hasNonFileFact := false
		for _, req := range rule.Requires() {
			if !hasPrefix(req, "file.") {
				hasNonFileFact = true
				break
			}
		}
		if hasNonFileFact {
			delayedFileRules = append(delayedFileRules, rule)
		}
	}

	for i := range files {
		file := &files[i]

		for _, rule := range delayedFileRules {
			ruleCfg := resolveRuleConfig(cfg, rule.ID(), nil)
			if !ruleCfg.Enabled {
				continue
			}

			ruleCtx := ProviderContext{
				Scope:      ScopeFile,
				File:       file,
				Runtime:    runtime,
				RuleConfig: &ruleCfg,
			}

			if !rule.Supports(ruleCtx) {
				continue
			}

			ruleFindings, err := rule.Evaluate(ruleCtx)
			if err != nil {
				continue
			}

			for _, rf := range ruleFindings {
				rf.Score *= ruleCfg.Weight
				findings = append(findings, rf.Finding)
			}
		}
	}

	for i := range directories {
		directory := &directories[i]

		for _, rule := range directoryRules {
			ruleCfg := resolveRuleConfig(cfg, rule.ID(), nil)
			if !ruleCfg.Enabled {
				continue
			}

			ruleCtx := ProviderContext{
				Scope:      ScopeDirectory,
				Directory:  directory,
				Runtime:    runtime,
				RuleConfig: &ruleCfg,
			}

			if !rule.Supports(ruleCtx) {
				continue
			}

			ruleFindings, err := rule.Evaluate(ruleCtx)
			if err != nil {
				continue
			}

			for _, rf := range ruleFindings {
				rf.Score *= ruleCfg.Weight
				findings = append(findings, rf.Finding)
			}
		}
	}

	for _, rule := range repoRules {
		ruleCfg := resolveRuleConfig(cfg, rule.ID(), nil)
		if !ruleCfg.Enabled {
			continue
		}

		ruleCtx := ProviderContext{
			Scope:      ScopeRepo,
			Runtime:    runtime,
			RuleConfig: &ruleCfg,
		}

		if !rule.Supports(ruleCtx) {
			continue
		}

		ruleFindings, err := rule.Evaluate(ruleCtx)
		if err != nil {
			continue
		}

		for _, rf := range ruleFindings {
			rf.Score *= ruleCfg.Weight
			findings = append(findings, rf.Finding)
		}
	}

	fileScores := buildFileScores(files, findings)
	directoryScores := buildDirectoryScores(directories, findings)
	summary := buildSummary(files, directories, findings, store)

	return &AnalysisResult{
		RootDir:         rootDir,
		Summary:         summary,
		Files:          files,
		Directories:    directories,
		Findings:       findings,
		FileScores:     fileScores,
		DirectoryScores: directoryScores,
		RepoScore:      summary.RepoScore,
		AnalyzedAt:     time.Now(),
	}, nil
}

func countLines(text string) int {
	if text == "" {
		return 0
	}
	count := 0
	for _, r := range text {
		if r == '\n' {
			count++
		}
	}
	if len(text) > 0 && text[len(text)-1] != '\n' {
		count++
	}
	return count
}

func countLogicalLines(text string, filePath string) int {
	if text == "" {
		return 0
	}

	count := 0
	inBlockComment := false
	lines := splitLines(text)

	for _, line := range lines {
		trimmed := trimSpace(line)

		if inBlockComment {
			if contains(trimmed, "*/") {
				inBlockComment = false
			}
			continue
		}

		if contains(trimmed, "/*") && !contains(trimmed, "*/") {
			inBlockComment = true
			continue
		}

		if trimmed == "" {
			continue
		}

		if hasPrefix(trimmed, "//") || hasPrefix(trimmed, "#") {
			continue
		}

		count++
	}

	return count
}

func splitLines(text string) []string {
	var lines []string
	start := 0
	for i := 0; i <= len(text); i++ {
		if i == len(text) || text[i] == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func hasPrefix(s, prefix string) bool {
	if len(prefix) == 0 {
		return true
	}
	if len(s) < len(prefix) {
		return false
	}
	return s[0:len(prefix)] == prefix
}

func hasSuffixAny(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}

func buildFileScores(files []FileRecord, findings []Finding) []FileScore {
	byFile := make(map[string]struct {
		score        float64
		findingCount int
	})

	for _, finding := range findings {
		if finding.Path == "" {
			continue
		}

		agg := byFile[finding.Path]
		agg.score += finding.Score
		agg.findingCount++
		byFile[finding.Path] = agg
	}

	fileScores := make([]FileScore, 0)
	for _, file := range files {
		agg, exists := byFile[file.Path]
		if exists && agg.findingCount > 0 {
			fileScores = append(fileScores, FileScore{
				Path:         file.Path,
				Score:        agg.score,
				FindingCount: agg.findingCount,
			})
		}
	}

	sort.Slice(fileScores, func(i, j int) bool {
		if fileScores[i].Score != fileScores[j].Score {
			return fileScores[i].Score > fileScores[j].Score
		}
		return fileScores[i].Path < fileScores[j].Path
	})

	return fileScores
}

func buildDirectoryScores(directories []DirectoryRecord, findings []Finding) []DirectoryScore {
	byDir := make(map[string]struct {
		score        float64
		findingCount int
	})

	for _, finding := range findings {
		if finding.Scope != ScopeDirectory || finding.Path == "" {
			continue
		}

		agg := byDir[finding.Path]
		agg.score += finding.Score
		agg.findingCount++
		byDir[finding.Path] = agg
	}

	dirScores := make([]DirectoryScore, 0)
	for _, dir := range directories {
		agg, exists := byDir[dir.Path]
		if exists && agg.findingCount > 0 {
			dirScores = append(dirScores, DirectoryScore{
				Path:         dir.Path,
				Score:        agg.score,
				FindingCount: agg.findingCount,
			})
		}
	}

	sort.Slice(dirScores, func(i, j int) bool {
		if dirScores[i].Score != dirScores[j].Score {
			return dirScores[i].Score > dirScores[j].Score
		}
		return dirScores[i].Path < dirScores[j].Path
	})

	return dirScores
}

func buildSummary(files []FileRecord, directories []DirectoryRecord, findings []Finding, store *FactStore) AnalysisSummary {
	repoScore := 0.0
	for _, f := range findings {
		repoScore += f.Score
	}

	physicalLineCount := 0
	for _, f := range files {
		physicalLineCount += f.LineCount
	}

	logicalLineCount := 0
	for _, f := range files {
		logicalLineCount += f.LogicalLineCount
	}

	functionCount := 0
	for _, file := range files {
		funcs := store.GetFileFact(file.Path, "file.functionSummaries")
		if funcs != nil {
			if funcList, ok := funcs.([]facts.FunctionSummary); ok {
				functionCount += len(funcList)
			}
		}
	}

	kloc := float64(logicalLineCount) / 1000.0

	normalized := NormalizedMetrics{
		HasData: true,
	}

	if len(files) > 0 {
		normalized.ScorePerFile = repoScore / float64(len(files))
		normalized.FindingsPerFile = float64(len(findings)) / float64(len(files))
	}

	if kloc > 0 {
		normalized.ScorePerKloc = repoScore / kloc
		normalized.FindingsPerKloc = float64(len(findings)) / kloc
	}

	if functionCount > 0 {
		normalized.ScorePerFunction = repoScore / float64(functionCount)
		normalized.FindingsPerFunction = float64(len(findings)) / float64(functionCount)
	}

	return AnalysisSummary{
		FileCount:         len(files),
		DirectoryCount:    len(directories),
		FindingCount:      len(findings),
		RepoScore:         repoScore,
		PhysicalLineCount: physicalLineCount,
		LogicalLineCount:  logicalLineCount,
		FunctionCount:     functionCount,
		Normalized:       normalized,
	}
}

func resolveRuleConfig(cfg *config.AnalyzerConfig, ruleID string, pathOverride *config.Override) types.RuleConfig {
	resolved := types.RuleConfig{
		Enabled: true,
		Weight:  1.0,
	}

	ruleConfig, exists := cfg.Rules[ruleID]
	if exists {
		if ruleConfig.Enabled != nil {
			resolved.Enabled = *ruleConfig.Enabled
		}
		if ruleConfig.Weight != nil {
			resolved.Weight = *ruleConfig.Weight
		}
		if ruleConfig.Options != nil {
			resolved.Options = ruleConfig.Options
		}
	}

	return resolved
}
