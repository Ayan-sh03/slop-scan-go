package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type RuleConfig struct {
	Enabled *bool    `json:"enabled,omitempty"`
	Weight  *float64 `json:"weight,omitempty"`
	Options any      `json:"options,omitempty"`
}

type Override struct {
	Files []string              `json:"files"`
	Rules map[string]RuleConfig `json:"rules"`
}

type AnalyzerConfig struct {
	Ignores   []string              `json:"ignores,omitempty"`
	Plugins   map[string]string     `json:"plugins,omitempty"`
	Extends   []string              `json:"extends,omitempty"`
	Rules     map[string]RuleConfig `json:"rules,omitempty"`
	Overrides []Override            `json:"overrides,omitempty"`
}

type ResolvedRuleConfig struct {
	Enabled bool
	Weight  float64
	Options any
}

func DefaultConfig() *AnalyzerConfig {
	return &AnalyzerConfig{
		Ignores:   make([]string, 0),
		Plugins:   make(map[string]string),
		Extends:   make([]string, 0),
		Rules:     make(map[string]RuleConfig),
		Overrides: make([]Override, 0),
	}
}

func LoadConfig(rootDir string) (*AnalyzerConfig, error) {
	configFiles := []string{
		"slop-scan.config.json",
		"slop-scan.config.ts",
		"slop-scan.config.js",
		"slop-scan.config.mjs",
		"slop-scan.config.cjs",
	}

	for _, configFile := range configFiles {
		configPath := filepath.Join(rootDir, configFile)
		if _, err := os.Stat(configPath); err == nil {
			if filepath.Ext(configFile) == ".json" {
				return loadJSONConfig(configPath)
			}
		}
	}

	return DefaultConfig(), nil
}

func loadJSONConfig(path string) (*AnalyzerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

func ResolveRuleConfigDefaults(config RuleConfig) ResolvedRuleConfig {
	resolved := ResolvedRuleConfig{
		Enabled: true,
		Weight:  1.0,
		Options: nil,
	}

	if config.Enabled != nil {
		resolved.Enabled = *config.Enabled
	}
	if config.Weight != nil {
		resolved.Weight = *config.Weight
	}
	if config.Options != nil {
		resolved.Options = config.Options
	}

	return resolved
}
