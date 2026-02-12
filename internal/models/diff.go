package models

// ChangeKind represents the type of change
type ChangeKind string

const (
	ChangeAdded    ChangeKind = "added"
	ChangeRemoved  ChangeKind = "removed"
	ChangeModified ChangeKind = "modified"
)

// Scope represents what entity was changed
type Scope string

const (
	ScopeService Scope = "service"
	ScopeVolume  Scope = "volume"
	ScopeNetwork Scope = "network"
)

// Severity represents the impact level of a change
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityBreaking Severity = "breaking"
)

// Change represents a single difference between two compose configurations
type Change struct {
	Kind     ChangeKind  `json:"kind"`
	Scope    Scope       `json:"scope"`
	Name     string      `json:"name"`      // e.g., service name
	Path     string      `json:"path"`      // e.g., services.api.environment.DATABASE_URL
	Before   interface{} `json:"before"`
	After    interface{} `json:"after"`
	Severity Severity    `json:"severity"`
}

// DiffSummary provides aggregate counts of changes
type DiffSummary struct {
	ServicesAdded    int `json:"services_added"`
	ServicesRemoved  int `json:"services_removed"`
	ServicesChanged  int `json:"services_changed"`
	VolumesAdded     int `json:"volumes_added"`
	VolumesRemoved   int `json:"volumes_removed"`
	NetworksAdded    int `json:"networks_added"`
	NetworksRemoved  int `json:"networks_removed"`
	TotalChanges     int `json:"total_changes"`
	BreakingCount    int `json:"breaking_count"`
	WarningCount     int `json:"warning_count"`
	InfoCount        int `json:"info_count"`
}

// DiffReport contains the full comparison result
type DiffReport struct {
	Summary DiffSummary `json:"summary"`
	Changes []Change    `json:"changes"`
}

// NewDiffReport creates an empty diff report
func NewDiffReport() *DiffReport {
	return &DiffReport{
		Changes: make([]Change, 0),
	}
}

// AddChange adds a change to the report and updates summary
func (r *DiffReport) AddChange(c Change) {
	r.Changes = append(r.Changes, c)
	r.Summary.TotalChanges++

	switch c.Severity {
	case SeverityBreaking:
		r.Summary.BreakingCount++
	case SeverityWarning:
		r.Summary.WarningCount++
	case SeverityInfo:
		r.Summary.InfoCount++
	}
}

// SeverityLevel returns a numeric level for severity comparison
func SeverityLevel(s Severity) int {
	switch s {
	case SeverityBreaking:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

// ParseSeverity converts a string to Severity
func ParseSeverity(s string) Severity {
	switch s {
	case "breaking":
		return SeverityBreaking
	case "warning":
		return SeverityWarning
	default:
		return SeverityInfo
	}
}
