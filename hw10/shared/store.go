package shared

import "sync"

// Store is a thread-safe in-memory key-value store.
// Every node (Leader, Follower, or Leaderless) embeds one of these.
type Store struct {
	mu   sync.RWMutex
	data map[string]Entry
}

// NewStore creates an empty store, ready to use.
func NewStore() *Store {
	return &Store{
		data: make(map[string]Entry),
	}
}

// Set writes a key-value pair and auto-increments the version.
// This is used when the node itself originates the write
// (i.e. the Leader or Coordinator receiving a client request).
// Returns the new version number.
func (s *Store) Set(key, value string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.data[key]
	newVersion := 1
	if exists {
		newVersion = existing.Version + 1
	}

	s.data[key] = Entry{
		Value:   value,
		Version: newVersion,
	}
	return newVersion
}

// SetWithVersion writes a key-value pair with a specific version.
// This is used by Followers/replicas when they receive a replicated
// write from the Leader or Coordinator — the version is already
// determined, so they just store it as-is.
func (s *Store) SetWithVersion(key, value string, version int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = Entry{
		Value:   value,
		Version: version,
	}
}

// Get retrieves the entry for a key.
// Returns the entry and true if found, or a zero Entry and false if not.
func (s *Store) Get(key string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.data[key]
	return entry, exists
}
