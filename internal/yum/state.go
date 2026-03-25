package yum

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const (
	// StateVersion is the current version of the state file format.
	StateVersion = 2
)

// State represents the yum-bundle managed state.
type State struct {
	Version  int      `json:"version"`
	Packages []string `json:"packages"`
	Repos    []string `json:"repos"`
	Keys     []string `json:"keys"`
	Groups   []string `json:"groups"`
}

// NewState creates a new empty state.
func NewState() *State {
	return &State{
		Version:  StateVersion,
		Packages: []string{},
		Repos:    []string{},
		Keys:     []string{},
		Groups:   []string{},
	}
}

// LoadState loads the state from disk, or returns a new state if none exists.
func (m *YumManager) LoadState() (*State, error) {
	path := m.StatePath()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewState(), nil
		}
		return nil, fmt.Errorf("load state from %s: %w", path, err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("load state from %s: %w", path, err)
	}

	return &state, nil
}

// SaveState persists the state to disk.
func (m *YumManager) SaveState(s *State) error {
	path := m.StatePath()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create state directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write state file %s: %w", path, err)
	}
	return nil
}

// AddPackage adds a package to the state if not already present.
func (s *State) AddPackage(pkg string) bool {
	if slices.Contains(s.Packages, pkg) {
		return false
	}
	s.Packages = append(s.Packages, pkg)
	return true
}

// RemovePackage removes a package from the state.
func (s *State) RemovePackage(pkg string) bool {
	idx := slices.Index(s.Packages, pkg)
	if idx == -1 {
		return false
	}
	s.Packages = slices.Delete(s.Packages, idx, idx+1)
	return true
}

// HasPackage checks if a package is tracked in the state.
func (s *State) HasPackage(pkg string) bool {
	return slices.Contains(s.Packages, pkg)
}

// AddRepo adds a repo file path to the state if not already present.
func (s *State) AddRepo(repo string) bool {
	if slices.Contains(s.Repos, repo) {
		return false
	}
	s.Repos = append(s.Repos, repo)
	return true
}

// RemoveRepo removes a repo path from the state.
func (s *State) RemoveRepo(repo string) bool {
	idx := slices.Index(s.Repos, repo)
	if idx == -1 {
		return false
	}
	s.Repos = slices.Delete(s.Repos, idx, idx+1)
	return true
}

// HasRepo checks if a repo path is tracked in the state.
func (s *State) HasRepo(repo string) bool {
	return slices.Contains(s.Repos, repo)
}

// AddKey adds a key path to the state if not already present.
func (s *State) AddKey(key string) bool {
	if slices.Contains(s.Keys, key) {
		return false
	}
	s.Keys = append(s.Keys, key)
	return true
}

// RemoveKey removes a key path from the state.
func (s *State) RemoveKey(key string) bool {
	idx := slices.Index(s.Keys, key)
	if idx == -1 {
		return false
	}
	s.Keys = slices.Delete(s.Keys, idx, idx+1)
	return true
}

// HasKey checks if a key path is tracked in the state.
func (s *State) HasKey(key string) bool {
	return slices.Contains(s.Keys, key)
}

// GetPackagesNotIn returns packages in state that are not in the given list.
func (s *State) GetPackagesNotIn(packages []string) []string {
	var result []string
	for _, pkg := range s.Packages {
		if !slices.Contains(packages, pkg) {
			result = append(result, pkg)
		}
	}
	return result
}

// AddGroup adds a group to the state if not already present.
func (s *State) AddGroup(group string) bool {
	if slices.Contains(s.Groups, group) {
		return false
	}
	s.Groups = append(s.Groups, group)
	return true
}

// RemoveGroup removes a group from the state.
func (s *State) RemoveGroup(group string) bool {
	idx := slices.Index(s.Groups, group)
	if idx == -1 {
		return false
	}
	s.Groups = slices.Delete(s.Groups, idx, idx+1)
	return true
}

// HasGroup checks if a group is tracked in the state.
func (s *State) HasGroup(group string) bool {
	return slices.Contains(s.Groups, group)
}

// GetGroupsNotIn returns groups in state that are not in the given list.
func (s *State) GetGroupsNotIn(groups []string) []string {
	var result []string
	for _, g := range s.Groups {
		if !slices.Contains(groups, g) {
			result = append(result, g)
		}
	}
	return result
}
