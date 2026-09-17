package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store persists the accumulated (deduplicated by IP) server list to a
// JSON file so the list survives restarts and works offline.
type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(configDir string) *Store {
	return &Store{path: filepath.Join(configDir, "servers.json")}
}

// Load returns every saved server, accumulated across refreshes.
func (st *Store) Load() []VpnServer {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.loadLocked()
}

// loadLocked reads and deduplicates the stored list. The caller must hold st.mu.
func (st *Store) loadLocked() []VpnServer {
	data, err := os.ReadFile(st.path)
	if err != nil {
		return nil
	}
	var servers []VpnServer
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil
	}
	// Deduplicate by IP defensively (older files may contain duplicates).
	seen := make(map[string]struct{}, len(servers))
	out := servers[:0]
	for _, s := range servers {
		if _, ok := seen[s.IP]; ok {
			continue
		}
		seen[s.IP] = struct{}{}
		out = append(out, s)
	}
	return out
}

// Merge combines a fresh fetch into the stored list. Existing entries are
// kept untouched; only new (non-duplicate) IPs are added. Returns the merged
// list and whether at least one new server was added (true when fresh data).
func (st *Store) Merge(fresh []VpnServer) ([]VpnServer, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()

	merged := st.loadLocked()
	byIP := make(map[string]int, len(merged))
	for i, s := range merged {
		byIP[s.IP] = i
	}
	added := false
	for _, s := range fresh {
		if _, ok := byIP[s.IP]; ok {
			continue
		}
		byIP[s.IP] = len(merged)
		merged = append(merged, s)
		added = true
	}
	if err := st.writeLocked(merged); err != nil {
		return merged, added
	}
	return merged, added
}

// Overwrite replaces the stored list entirely (used on cache clear / deletes).
func (st *Store) Overwrite(servers []VpnServer) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.writeLocked(servers)
}

// Delete removes servers by IP and returns the remaining list.
func (st *Store) Delete(ips map[string]struct{}) []VpnServer {
	st.mu.Lock()
	defer st.mu.Unlock()

	var remaining []VpnServer
	for _, s := range st.loadLocked() {
		if _, ok := ips[s.IP]; !ok {
			remaining = append(remaining, s)
		}
	}
	_ = st.writeLocked(remaining)
	return remaining
}

func (st *Store) Clear() {
	st.mu.Lock()
	defer st.mu.Unlock()
	_ = os.Remove(st.path)
}

func (st *Store) writeLocked(servers []VpnServer) error {
	if dir := filepath.Dir(st.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(st.path, data, 0o644)
}