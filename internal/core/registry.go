package core

type Registry struct {
	languages    []LanguagePlugin
	factProviders []FactProvider
	rules        []RulePlugin
	reporters    map[string]ReporterPlugin
}

func NewRegistry() *Registry {
	return &Registry{
		languages:     make([]LanguagePlugin, 0),
		factProviders: make([]FactProvider, 0),
		rules:         make([]RulePlugin, 0),
		reporters:     make(map[string]ReporterPlugin),
	}
}

func (r *Registry) RegisterLanguage(plugin LanguagePlugin) {
	for _, existing := range r.languages {
		if existing.ID() == plugin.ID() {
			panic("Duplicate language plugin id: " + plugin.ID())
		}
	}
	r.languages = append(r.languages, plugin)
}

func (r *Registry) RegisterFactProvider(plugin FactProvider) {
	for _, existing := range r.factProviders {
		if existing.ID() == plugin.ID() {
			panic("Duplicate fact provider id: " + plugin.ID())
		}
	}
	r.factProviders = append(r.factProviders, plugin)
}

func (r *Registry) RegisterRule(plugin RulePlugin) {
	for _, existing := range r.rules {
		if existing.ID() == plugin.ID() {
			panic("Duplicate rule id: " + plugin.ID())
		}
	}
	r.rules = append(r.rules, plugin)
}

func (r *Registry) RegisterReporter(plugin ReporterPlugin) {
	if _, exists := r.reporters[plugin.ID()]; exists {
		panic("Duplicate reporter id: " + plugin.ID())
	}
	r.reporters[plugin.ID()] = plugin
}

func (r *Registry) GetLanguages() []LanguagePlugin {
	result := make([]LanguagePlugin, len(r.languages))
	copy(result, r.languages)
	return result
}

func (r *Registry) GetFactProviders() []FactProvider {
	result := make([]FactProvider, len(r.factProviders))
	copy(result, r.factProviders)
	return result
}

func (r *Registry) GetRules() []RulePlugin {
	result := make([]RulePlugin, len(r.rules))
	copy(result, r.rules)
	return result
}

func (r *Registry) GetReporter(id string) (ReporterPlugin, error) {
	reporter, ok := r.reporters[id]
	if !ok {
		return nil, &Error{Message: "Unknown reporter: " + id}
	}
	return reporter, nil
}

func (r *Registry) DetectLanguage(filePath string) LanguagePlugin {
	for _, lang := range r.languages {
		if lang.Supports(filePath) {
			return lang
		}
	}
	return nil
}

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
