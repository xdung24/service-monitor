package database

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
)

// Registry manages per-user SQLite database connections, lazily opening and
// caching them on first access.
type Registry struct {
	mu    sync.RWMutex
	conns map[string]*sql.DB
	dir   string // data root, e.g. "./data"
}

// NewRegistry creates a Registry that stores user databases under
// <dir>/users/<username>/data.db.
func NewRegistry(dataDir string) *Registry {
	return &Registry{
		conns: make(map[string]*sql.DB),
		dir:   dataDir,
	}
}

// Get returns (and lazily opens+migrates) the database for the given user.
func (r *Registry) Get(username string) (*sql.DB, error) {
	// Fast path — already open.
	r.mu.RLock()
	if db, ok := r.conns[username]; ok {
		r.mu.RUnlock()
		return db, nil
	}
	r.mu.RUnlock()

	// Slow path — open and migrate.
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring the write lock.
	if db, ok := r.conns[username]; ok {
		return db, nil
	}

	path := filepath.Join(r.dir, "users", username, "data.db")
	db, err := Open(path)
	if err != nil {
		return nil, fmt.Errorf("registry: open db for %q: %w", username, err)
	}

	if err := MigrateUserDB(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("registry: migrate db for %q: %w", username, err)
	}

	r.conns[username] = db
	return db, nil
}

// Remove closes and evicts the cached database connection for a single user.
func (r *Registry) Remove(username string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if db, ok := r.conns[username]; ok {
		_ = db.Close()
		delete(r.conns, username)
	}
}

// Close closes all open per-user database connections.
func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for username, db := range r.conns {
		_ = db.Close()
		delete(r.conns, username)
	}
}
