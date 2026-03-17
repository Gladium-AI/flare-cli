package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store persists and retrieves sessions.
type Store interface {
	Save(ctx context.Context, s *Session) error
	Load(ctx context.Context, id string) (*Session, error)
	List(ctx context.Context, stateFilter ...State) ([]*Session, error)
	Delete(ctx context.Context, id string) error
	Resolve(ctx context.Context, idOrPrefix string) (*Session, error)
}

// FileStore implements Store using JSON files on disk.
type FileStore struct {
	dir string
}

// NewFileStore creates a FileStore at the given directory.
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating sessions dir: %w", err)
	}
	return &FileStore{dir: dir}, nil
}

// Save writes a session to disk atomically (write tmp, then rename).
func (fs *FileStore) Save(_ context.Context, s *Session) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	target := fs.path(s.ID)
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("writing session tmp file: %w", err)
	}

	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming session file: %w", err)
	}

	return nil
}

// Load reads a session from disk by ID.
func (fs *FileStore) Load(_ context.Context, id string) (*Session, error) {
	data, err := os.ReadFile(fs.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session %q not found", id)
		}
		return nil, fmt.Errorf("reading session: %w", err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing session: %w", err)
	}

	return &s, nil
}

// List returns all sessions, optionally filtered by state.
func (fs *FileStore) List(_ context.Context, stateFilter ...State) ([]*Session, error) {
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading sessions dir: %w", err)
	}

	filterSet := make(map[State]bool, len(stateFilter))
	for _, s := range stateFilter {
		filterSet[s] = true
	}

	var sessions []*Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(fs.dir, entry.Name()))
		if err != nil {
			continue // Skip unreadable files.
		}

		var s Session
		if err := json.Unmarshal(data, &s); err != nil {
			continue // Skip corrupt files.
		}

		if len(filterSet) > 0 && !filterSet[s.State] {
			continue
		}

		sessions = append(sessions, &s)
	}

	return sessions, nil
}

// Delete removes a session file.
func (fs *FileStore) Delete(_ context.Context, id string) error {
	if err := os.Remove(fs.path(id)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

// Resolve finds a session by full or partial ID (prefix match).
func (fs *FileStore) Resolve(ctx context.Context, idOrPrefix string) (*Session, error) {
	// Try exact match first.
	s, err := fs.Load(ctx, idOrPrefix)
	if err == nil {
		return s, nil
	}

	// Try prefix match.
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("reading sessions dir: %w", err)
	}

	var matches []*Session
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".json")
		if strings.HasPrefix(name, idOrPrefix) {
			data, err := os.ReadFile(filepath.Join(fs.dir, entry.Name()))
			if err != nil {
				continue
			}
			var sess Session
			if err := json.Unmarshal(data, &sess); err != nil {
				continue
			}
			matches = append(matches, &sess)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no session matching %q", idOrPrefix)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous session prefix %q matches %d sessions", idOrPrefix, len(matches))
	}
}

func (fs *FileStore) path(id string) string {
	return filepath.Join(fs.dir, id+".json")
}
