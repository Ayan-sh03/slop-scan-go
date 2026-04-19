package core

import "github.com/modem-dev/slop-scan-go/internal/types"

type Scope = types.Scope

const (
	ScopeFile      = types.ScopeFile
	ScopeDirectory = types.ScopeDirectory
	ScopeRepo      = types.ScopeRepo
)

type FileRecord = types.FileRecord
type DirectoryRecord = types.DirectoryRecord
type FindingLocation = types.FindingLocation
type Finding = types.Finding
type RuleFinding = types.RuleFinding
type FileScore = types.FileScore
type DirectoryScore = types.DirectoryScore
type NormalizedMetrics = types.NormalizedMetrics
type AnalysisSummary = types.AnalysisSummary
type AnalysisResult = types.AnalysisResult
type LanguagePlugin = types.LanguagePlugin
type FactProvider = types.FactProvider
type RulePlugin = types.RulePlugin
type ReporterPlugin = types.ReporterPlugin
type ProviderContext = types.ProviderContext
type AnalyzerRuntime = types.AnalyzerRuntime
type FactStoreReader = types.FactStoreReader
type RuleConfig = types.RuleConfig
