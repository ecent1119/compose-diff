package baseline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Baseline represents a saved compose state for comparison
type Baseline struct {
	Version   string            `json:"version"`
	Name      string            `json:"name"`
	CreatedAt time.Time         `json:"created_at"`
	Source    string            `json:"source"`           // original file path
	Resolved  bool              `json:"resolved"`         // if from docker compose config
	Data      map[string]any    `json:"data"`             // the parsed compose content
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Manager handles baseline operations
type Manager struct {
	baseDir string
}

// NewManager creates a baseline manager
func NewManager(baseDir string) *Manager {
	if baseDir == "" {
		baseDir = ".compose-diff"
	}
	return &Manager{baseDir: baseDir}
}

// Save saves a baseline snapshot
func (m *Manager) Save(name string, data map[string]any, source string, resolved bool) error {
	if err := os.MkdirAll(m.baseDir, 0755); err != nil {
		return err
	}

	baseline := &Baseline{
		Version:   "1.0",
		Name:      name,
		CreatedAt: time.Now(),
		Source:    source,
		Resolved:  resolved,
		Data:      data,
	}

	content, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return err
	}

	filename := sanitizeFilename(name) + ".json"
	path := filepath.Join(m.baseDir, filename)

	return os.WriteFile(path, content, 0644)
}

// Load loads a baseline by name
func (m *Manager) Load(name string) (*Baseline, error) {
	filename := sanitizeFilename(name) + ".json"
	path := filepath.Join(m.baseDir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var baseline Baseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, err
	}

	return &baseline, nil
}

// List returns all available baselines
func (m *Manager) List() ([]*Baseline, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var baselines []*Baseline
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(m.baseDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var baseline Baseline
		if err := json.Unmarshal(data, &baseline); err != nil {
			continue
		}

		baselines = append(baselines, &baseline)
	}

	return baselines, nil
}

// Delete removes a baseline
func (m *Manager) Delete(name string) error {
	filename := sanitizeFilename(name) + ".json"
	path := filepath.Join(m.baseDir, filename)
	return os.Remove(path)
}

// Exists checks if a baseline exists
func (m *Manager) Exists(name string) bool {
	filename := sanitizeFilename(name) + ".json"
	path := filepath.Join(m.baseDir, filename)
	_, err := os.Stat(path)
	return err == nil
}

// sanitizeFilename makes a name safe for filesystem
func sanitizeFilename(name string) string {
	// Replace problematic characters
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			result += string(r)
		} else if r == ' ' || r == '/' || r == '\\' {
			result += "_"
		}
	}
	if result == "" {
		result = "default"
	}
	return result
}
