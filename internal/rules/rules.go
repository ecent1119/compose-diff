package rules

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/stackgen-cli/compose-diff/internal/models"
	"gopkg.in/yaml.v3"
)

// RulesConfig defines custom diff rules
type RulesConfig struct {
	Version string `yaml:"version"`

	// SeverityOverrides maps path patterns to severity levels
	SeverityOverrides []SeverityRule `yaml:"severity_overrides"`

	// IgnorePatterns defines paths to completely ignore
	IgnorePatterns []IgnoreRule `yaml:"ignore_patterns"`

	// ServiceIgnores defines per-service ignore lists
	ServiceIgnores map[string]ServiceIgnoreRules `yaml:"service_ignores"`

	// Categories defines custom category mappings
	Categories map[string][]string `yaml:"categories"`
}

// SeverityRule maps a path pattern to a severity
type SeverityRule struct {
	Pattern  string `yaml:"pattern"`  // glob or regex pattern
	Severity string `yaml:"severity"` // info, warning, breaking
	IsRegex  bool   `yaml:"regex"`    // if true, use regex matching
}

// IgnoreRule defines what to ignore
type IgnoreRule struct {
	Pattern string `yaml:"pattern"` // path pattern to ignore
	IsRegex bool   `yaml:"regex"`   // if true, use regex matching
	Reason  string `yaml:"reason"`  // why it's ignored (for reports)
}

// ServiceIgnoreRules defines ignores for a specific service
type ServiceIgnoreRules struct {
	Paths  []string `yaml:"paths"`  // paths to ignore for this service
	Fields []string `yaml:"fields"` // field names to ignore (e.g., "image", "environment")
}

// Rules holds the loaded rules configuration
type Rules struct {
	config           *RulesConfig
	severityPatterns []compiledSeverity
	ignorePatterns   []compiledIgnore
}

type compiledSeverity struct {
	pattern  *regexp.Regexp
	literal  string
	severity string
	isRegex  bool
}

type compiledIgnore struct {
	pattern *regexp.Regexp
	literal string
	isRegex bool
	reason  string
}

// LoadRules loads rules from a file path
func LoadRules(path string) (*Rules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config RulesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return compileRules(&config)
}

// LoadRulesFromDir finds and loads .compose-diff.yaml from directory
func LoadRulesFromDir(dir string) (*Rules, error) {
	candidates := []string{
		".compose-diff.yaml",
		".compose-diff.yml",
		"compose-diff.yaml",
		"compose-diff.yml",
	}

	for _, name := range candidates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return LoadRules(path)
		}
	}

	// Return empty rules if no file found
	return &Rules{config: &RulesConfig{}}, nil
}

// compileRules compiles the patterns for efficient matching
func compileRules(config *RulesConfig) (*Rules, error) {
	rules := &Rules{
		config:           config,
		severityPatterns: make([]compiledSeverity, 0, len(config.SeverityOverrides)),
		ignorePatterns:   make([]compiledIgnore, 0, len(config.IgnorePatterns)),
	}

	// Compile severity patterns
	for _, sr := range config.SeverityOverrides {
		cs := compiledSeverity{
			severity: sr.Severity,
			isRegex:  sr.IsRegex,
		}
		if sr.IsRegex {
			re, err := regexp.Compile(sr.Pattern)
			if err != nil {
				return nil, err
			}
			cs.pattern = re
		} else {
			cs.literal = sr.Pattern
		}
		rules.severityPatterns = append(rules.severityPatterns, cs)
	}

	// Compile ignore patterns
	for _, ir := range config.IgnorePatterns {
		ci := compiledIgnore{
			isRegex: ir.IsRegex,
			reason:  ir.Reason,
		}
		if ir.IsRegex {
			re, err := regexp.Compile(ir.Pattern)
			if err != nil {
				return nil, err
			}
			ci.pattern = re
		} else {
			ci.literal = ir.Pattern
		}
		rules.ignorePatterns = append(rules.ignorePatterns, ci)
	}

	return rules, nil
}

// GetSeverityOverride returns custom severity if matched, empty otherwise
func (r *Rules) GetSeverityOverride(path string) (models.Severity, bool) {
	for _, sp := range r.severityPatterns {
		if sp.isRegex {
			if sp.pattern.MatchString(path) {
				return models.Severity(sp.severity), true
			}
		} else {
			if matchGlob(sp.literal, path) {
				return models.Severity(sp.severity), true
			}
		}
	}
	return "", false
}

// ShouldIgnore returns true if the path should be ignored
func (r *Rules) ShouldIgnore(path string) (bool, string) {
	for _, ip := range r.ignorePatterns {
		if ip.isRegex {
			if ip.pattern.MatchString(path) {
				return true, ip.reason
			}
		} else {
			if matchGlob(ip.literal, path) {
				return true, ip.reason
			}
		}
	}
	return false, ""
}

// ShouldIgnoreServiceField returns true if the field should be ignored for service
func (r *Rules) ShouldIgnoreServiceField(service, field string) bool {
	if r.config.ServiceIgnores == nil {
		return false
	}

	si, ok := r.config.ServiceIgnores[service]
	if !ok {
		return false
	}

	for _, f := range si.Fields {
		if f == field {
			return true
		}
	}

	for _, p := range si.Paths {
		if matchGlob(p, field) {
			return true
		}
	}

	return false
}

// GetCategory returns the category for a path
func (r *Rules) GetCategory(path string) string {
	// Default categories based on path
	if strings.Contains(path, ".environment") {
		return "environment"
	}
	if strings.Contains(path, ".ports") {
		return "ports"
	}
	if strings.Contains(path, ".image") {
		return "images"
	}
	if strings.Contains(path, ".volumes") {
		return "volumes"
	}
	if strings.Contains(path, "networks.") {
		return "networks"
	}
	if strings.Contains(path, "volumes.") && !strings.Contains(path, ".volumes") {
		return "volumes"
	}
	return "other"
}

// GetCustomCategories returns custom category definitions
func (r *Rules) GetCustomCategories() map[string][]string {
	if r.config.Categories == nil {
		return DefaultCategories()
	}
	return r.config.Categories
}

// DefaultCategories returns the default category groupings
func DefaultCategories() map[string][]string {
	return map[string][]string{
		"environment": {"*.environment.*"},
		"ports":       {"*.ports.*", "*.expose.*"},
		"images":      {"*.image", "*.build.*"},
		"volumes":     {"*.volumes.*", "volumes.*"},
		"networks":    {"*.networks.*", "networks.*"},
		"deploy":      {"*.deploy.*", "*.replicas", "*.resources.*"},
		"other":       {"*"},
	}
}

// matchGlob does simple glob matching (* wildcards)
func matchGlob(pattern, str string) bool {
	// Simple glob matching
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	pattern = "^" + pattern + "$"

	re, err := regexp.Compile(pattern)
	if err != nil {
		return pattern == str
	}
	return re.MatchString(str)
}
