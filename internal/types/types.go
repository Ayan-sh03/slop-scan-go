package types

import "time"

type Scope string

const (
	ScopeFile      Scope = "file"
	ScopeDirectory Scope = "directory"
	ScopeRepo     Scope = "repo"
)

type FileRecord struct {
	Path              string
	AbsolutePath      string
	Extension         string
	LineCount         int
	LogicalLineCount  int
	LanguageID        string
}

type DirectoryRecord struct {
	Path      string
	FilePaths []string
}

type FindingLocation struct {
	Path   string
	Line   int
	Column int
}

type FindingDeltaOccurrenceIdentity struct {
	Fingerprint       string
	GroupFingerprint string
	Path              string
	Line              int
	Column            int
}

type FindingDeltaIdentity struct {
	FingerprintVersion int
	Occurrences       []FindingDeltaOccurrenceIdentity
}

type Finding struct {
	RuleID        string
	Family        string
	Severity      string
	Scope         Scope
	Message       string
	Evidence      []string
	Score         float64
	Locations     []FindingLocation
	Path         string
	DeltaIdentity *FindingDeltaIdentity
}

type DeltaKey struct {
	Key    string
	Group  string
	Path   string
	Line   int
	Column int
}

type RuleFinding struct {
	Finding
	DeltaKeys []DeltaKey
}

type FileScore struct {
	Path         string
	Score        float64
	FindingCount int
}

type DirectoryScore struct {
	Path         string
	Score        float64
	FindingCount int
}

type NormalizedMetrics struct {
	ScorePerFile        float64
	ScorePerKloc        float64
	ScorePerFunction    float64
	FindingsPerFile     float64
	FindingsPerKloc     float64
	FindingsPerFunction float64
	HasData            bool
}

type AnalysisSummary struct {
	FileCount         int
	DirectoryCount    int
	FindingCount     int
	RepoScore        float64
	PhysicalLineCount int
	LogicalLineCount int
	FunctionCount    int
	Normalized      NormalizedMetrics
}

type ReportPluginMetadata struct {
	Namespace string
	Name     string
	Version  string
	Source   string
}

type ReportMetadata struct {
	SchemaVersion            int
	ToolName                string
	ToolVersion             string
	ConfigHash              string
	FindingFingerprintVersion int
	Plugins                 []ReportPluginMetadata
}

type AnalysisResult struct {
	RootDir         string
	Summary         AnalysisSummary
	Files           []FileRecord
	Directories     []DirectoryRecord
	Findings        []Finding
	FileScores      []FileScore
	DirectoryScores []DirectoryScore
	RepoScore       float64
	Metadata        *ReportMetadata
	AnalyzedAt      time.Time
}

type LanguagePlugin interface {
	ID() string
	Supports(filePath string) bool
}

type ProviderBase interface {
	ID() string
	Scope() Scope
	Requires() []string
	Supports(context ProviderContext) bool
}

type FactProvider interface {
	ProviderBase
	Provides() []string
	Run(context ProviderContext) (map[string]any, error)
}

type RulePlugin interface {
	ProviderBase
	Family() string
	Severity() string
	Evaluate(context ProviderContext) ([]RuleFinding, error)
}

type ReporterPlugin interface {
	ID() string
	Render(result AnalysisResult) (string, error)
}

type AnalyzerRuntime struct {
	RootDir     string
	Config      any
	Files       []FileRecord
	Directories []DirectoryRecord
	Store       FactStoreReader
}

type ProviderContext struct {
	Scope      Scope
	Runtime    AnalyzerRuntime
	File       *FileRecord
	Directory  *DirectoryRecord
	RuleConfig *RuleConfig
}

type FactStoreReader interface {
	GetRepoFact(factID string) any
	GetDirectoryFact(directoryPath, factID string) any
	GetFileFact(filePath, factID string) any
	HasRepoFact(factID string) bool
	HasDirectoryFact(directoryPath, factID string) bool
	HasFileFact(filePath, factID string) bool
}

type RuleConfig struct {
	Enabled bool
	Weight  float64
	Options any
}
