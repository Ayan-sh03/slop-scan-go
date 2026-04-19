package core

import (
	"fmt"
)

type itemWithRequires interface {
	Requires() []string
}

func OrderFactProviders(providers []FactProvider, baseFacts []string) []FactProvider {
	baseFactSet := make(map[string]bool)
	for _, fact := range baseFacts {
		baseFactSet[fact] = true
	}

	availableFacts := make(map[string]bool)
	for fact := range baseFactSet {
		availableFacts[fact] = true
	}

	ordered := make([]FactProvider, 0, len(providers))
	remaining := make([]FactProvider, len(providers))
	copy(remaining, providers)

	maxIterations := len(providers) * len(providers)
	iterations := 0

	for len(remaining) > 0 && iterations < maxIterations {
		iterations++
		ready := make([]FactProvider, 0)
		notReady := make([]FactProvider, 0)

		for _, provider := range remaining {
			canRun := true
			for _, req := range provider.Requires() {
				if !availableFacts[req] {
					canRun = false
					break
				}
			}
			if canRun {
				ready = append(ready, provider)
				for _, provided := range provider.Provides() {
					availableFacts[provided] = true
				}
			} else {
				notReady = append(notReady, provider)
			}
		}

		ordered = append(ordered, ready...)
		remaining = notReady
	}

	if len(remaining) > 0 {
		unmet := make([]string, 0)
		for _, provider := range remaining {
			for _, req := range provider.Requires() {
				if !availableFacts[req] {
					unmet = append(unmet, fmt.Sprintf("%s requires %s", provider.ID(), req))
				}
			}
		}
		panic(fmt.Sprintf("Cannot satisfy fact dependencies: %v", unmet))
	}

	return ordered
}

func ValidateRuleRequirements(rules []struct {
	ID       string
	Requires []string
}, availableFacts []string) {
	factSet := make(map[string]bool)
	for _, fact := range availableFacts {
		factSet[fact] = true
	}

	var missing []string
	for _, rule := range rules {
		for _, req := range rule.Requires {
			if !factSet[req] {
				missing = append(missing, fmt.Sprintf("%s requires %s", rule.ID, req))
			}
		}
	}

	if len(missing) > 0 {
		panic(fmt.Sprintf("Missing facts for rules: %v", missing))
	}
}
