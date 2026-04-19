package languages

import (
	"path/filepath"
	"strings"
)

type GoLanguagePlugin struct{}

var supportedExtensions = map[string]bool{
	".go": true,
}

func (g *GoLanguagePlugin) ID() string {
	return "go"
}

func (g *GoLanguagePlugin) Supports(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return supportedExtensions[ext]
}

func NewGoLanguagePlugin() *GoLanguagePlugin {
	return &GoLanguagePlugin{}
}
