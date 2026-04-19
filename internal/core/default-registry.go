package core

import (
	"github.com/modem-dev/slop-scan-go/internal/facts"
	"github.com/modem-dev/slop-scan-go/internal/languages"
	"github.com/modem-dev/slop-scan-go/internal/reporters"
	"github.com/modem-dev/slop-scan-go/internal/rules"
)

func CreateDefaultRegistry() *Registry {
	registry := NewRegistry()

	registry.RegisterLanguage(languages.NewGoLanguagePlugin())

	registry.RegisterFactProvider(facts.NewASTFactProvider())
	registry.RegisterFactProvider(facts.NewFunctionsFactProvider())
	registry.RegisterFactProvider(facts.NewTryCatchFactProvider())
	registry.RegisterFactProvider(facts.NewCommentsFactProvider())

	registry.RegisterRule(rules.NewErrorSwallowingRule())
	registry.RegisterRule(rules.NewErrorObscuringRule())
	registry.RegisterRule(rules.NewIgnoredErrorRule())
	registry.RegisterRule(rules.NewPassThroughWrappersRule())
	registry.RegisterRule(rules.NewPlaceholderCommentsRule())

	registry.RegisterReporter(reporters.NewTextReporter())
	registry.RegisterReporter(reporters.NewJSONReporter())
	registry.RegisterReporter(reporters.NewLintReporter())

	return registry
}
