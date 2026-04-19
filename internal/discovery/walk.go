package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/modem-dev/slop-scan-go/internal/config"
	"github.com/modem-dev/slop-scan-go/internal/types"
)

type cachedFile struct {
	Path         string
	AbsolutePath string
	Extension    string
	LanguageID   string
}

type cachedDirectory struct {
	Path      string
	FilePaths []string
}

type cachedDiscoveryResult struct {
	Files           []cachedFile
	Directories     []cachedDirectory
	DirectoryMtimes map[string]int64
	GitignoreMtime  int64
}

var discoveryCache = make(map[string]*cachedDiscoveryResult)

type FileRecord struct {
	Path         string
	AbsolutePath string
	Extension    string
	LanguageID   string
}

type DirectoryRecord struct {
	Path      string
	FilePaths []string
}

func normalizePath(path string) string {
	return filepath.ToSlash(path)
}

func discoveryCacheKey(rootDir string, cfg *config.AnalyzerConfig, languages []types.LanguagePlugin) string {
	var keyBuilder strings.Builder
	keyBuilder.WriteString(rootDir)
	keyBuilder.WriteString("\x01")
	for _, ignore := range cfg.Ignores {
		keyBuilder.WriteString(ignore)
		keyBuilder.WriteString("\x00")
	}
	keyBuilder.WriteString("\x01")
	for _, lang := range languages {
		keyBuilder.WriteString(lang.ID())
		keyBuilder.WriteString("\x00")
	}
	return keyBuilder.String()
}

func shouldIgnore(path string, ignores []string) bool {
	for _, pattern := range ignores {
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		if strings.Contains(pattern, "**") {
			globPattern := strings.ReplaceAll(pattern, "**/", "*")
			matched, err := filepath.Match(globPattern, path)
			if err == nil && matched {
				return true
			}
		}
	}
	return false
}

func DiscoverSourceFiles(rootDir string, cfg *config.AnalyzerConfig, languages []types.LanguagePlugin) ([]FileRecord, []DirectoryRecord, error) {
	cacheKey := discoveryCacheKey(rootDir, cfg, languages)

	if cached, ok := discoveryCache[cacheKey]; ok {
		files := make([]FileRecord, len(cached.Files))
		for i, f := range cached.Files {
			files[i] = FileRecord{
				Path:         f.Path,
				AbsolutePath: f.AbsolutePath,
				Extension:    f.Extension,
				LanguageID:   f.LanguageID,
			}
		}

		directories := make([]DirectoryRecord, len(cached.Directories))
		for i, d := range cached.Directories {
			directories[i] = DirectoryRecord{
				Path:      d.Path,
				FilePaths: append([]string(nil), d.FilePaths...),
			}
		}

		return files, directories, nil
	}

	files := make([]FileRecord, 0)

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		relPath = normalizePath(relPath)

		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			if shouldIgnore(relPath, cfg.Ignores) {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldIgnore(relPath, cfg.Ignores) {
			return nil
		}

		language := detectLanguage(path, languages)
		if language == nil {
			return nil
		}

		files = append(files, FileRecord{
			Path:         relPath,
			AbsolutePath: path,
			Extension:    filepath.Ext(relPath),
			LanguageID:   language.ID(),
		})

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	directoryMap := make(map[string][]string)
	for _, file := range files {
		dirPath := normalizePath(filepath.Dir(file.Path))
		directoryMap[dirPath] = append(directoryMap[dirPath], file.Path)
	}

	directories := make([]DirectoryRecord, 0, len(directoryMap))
	for dirPath, filePaths := range directoryMap {
		sortedPaths := append([]string(nil), filePaths...)
		sort.Strings(sortedPaths)
		directories = append(directories, DirectoryRecord{
			Path:      dirPath,
			FilePaths: sortedPaths,
		})
	}

	sort.Slice(directories, func(i, j int) bool {
		return directories[i].Path < directories[j].Path
	})

	cachedFiles := make([]cachedFile, len(files))
	for i, f := range files {
		cachedFiles[i] = cachedFile{
			Path:         f.Path,
			AbsolutePath: f.AbsolutePath,
			Extension:    f.Extension,
			LanguageID:   f.LanguageID,
		}
	}

	cachedDirs := make([]cachedDirectory, len(directories))
	for i, d := range directories {
		cachedDirs[i] = cachedDirectory{
			Path:      d.Path,
			FilePaths: append([]string(nil), d.FilePaths...),
		}
	}

	discoveryCache[cacheKey] = &cachedDiscoveryResult{
		Files:       cachedFiles,
		Directories: cachedDirs,
	}

	return files, directories, nil
}

func detectLanguage(path string, languages []types.LanguagePlugin) types.LanguagePlugin {
	for _, lang := range languages {
		if lang.Supports(path) {
			return lang
		}
	}
	return nil
}
