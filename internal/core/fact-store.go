package core

import (
	"sync"
)

type FactStore struct {
	mu            sync.RWMutex
	repoFacts     map[string]any
	directoryFacts map[string]map[string]any
	fileFacts     map[string]map[string]any
}

func NewFactStore() *FactStore {
	return &FactStore{
		repoFacts:     make(map[string]any),
		directoryFacts: make(map[string]map[string]any),
		fileFacts:     make(map[string]map[string]any),
	}
}

func (s *FactStore) SetRepoFact(factID string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repoFacts[factID] = value
}

func (s *FactStore) SetDirectoryFact(directoryPath, factID string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.directoryFacts[directoryPath] == nil {
		s.directoryFacts[directoryPath] = make(map[string]any)
	}
	s.directoryFacts[directoryPath][factID] = value
}

func (s *FactStore) SetFileFact(filePath, factID string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fileFacts[filePath] == nil {
		s.fileFacts[filePath] = make(map[string]any)
	}
	s.fileFacts[filePath][factID] = value
}

func (s *FactStore) SetFileFacts(filePath string, facts map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fileFacts[filePath] == nil {
		s.fileFacts[filePath] = make(map[string]any)
	}
	for k, v := range facts {
		s.fileFacts[filePath][k] = v
	}
}

func (s *FactStore) GetRepoFact(factID string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.repoFacts[factID]
}

func (s *FactStore) GetDirectoryFact(directoryPath, factID string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.directoryFacts[directoryPath] == nil {
		return nil
	}
	return s.directoryFacts[directoryPath][factID]
}

func (s *FactStore) GetFileFact(filePath, factID string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.fileFacts[filePath] == nil {
		return nil
	}
	return s.fileFacts[filePath][factID]
}

func (s *FactStore) HasRepoFact(factID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.repoFacts[factID]
	return ok
}

func (s *FactStore) HasDirectoryFact(directoryPath, factID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.directoryFacts[directoryPath] == nil {
		return false
	}
	_, ok := s.directoryFacts[directoryPath][factID]
	return ok
}

func (s *FactStore) HasFileFact(filePath, factID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.fileFacts[filePath] == nil {
		return false
	}
	_, ok := s.fileFacts[filePath][factID]
	return ok
}

func (s *FactStore) RetainFileFacts(filePath string, keepFacts map[string]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fileFacts[filePath] == nil {
		return
	}
	for factID := range s.fileFacts[filePath] {
		if !keepFacts[factID] {
			delete(s.fileFacts[filePath], factID)
		}
	}
}

func (s *FactStore) ClearFileFacts(filePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.fileFacts, filePath)
}
