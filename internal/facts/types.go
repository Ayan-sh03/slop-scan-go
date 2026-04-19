package facts

import "github.com/modem-dev/slop-scan-go/internal/types"

type Scope = types.Scope

const (
	ScopeFile      = types.ScopeFile
	ScopeDirectory = types.ScopeDirectory
	ScopeRepo      = types.ScopeRepo
)

type FileRecord = types.FileRecord
type DirectoryRecord = types.DirectoryRecord
type ProviderContext = types.ProviderContext
type RuleConfig = types.RuleConfig
