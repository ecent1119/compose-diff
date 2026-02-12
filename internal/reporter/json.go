package reporter

import "github.com/stackgen-cli/compose-diff/internal/models"

// JSONReport is the stable JSON output format
type JSONReport struct {
	SchemaVersion string          `json:"schema_version"`
	OldFile       string          `json:"old_file"`
	NewFile       string          `json:"new_file"`
	Summary       JSONSummary     `json:"summary"`
	Changes       []models.Change `json:"changes"`
}

// JSONSummary is the summary section of JSON output
type JSONSummary struct {
	ServicesAdded   int `json:"services_added"`
	ServicesRemoved int `json:"services_removed"`
	ServicesChanged int `json:"services_changed"`
	VolumesAdded    int `json:"volumes_added"`
	VolumesRemoved  int `json:"volumes_removed"`
	NetworksAdded   int `json:"networks_added"`
	NetworksRemoved int `json:"networks_removed"`
	TotalChanges    int `json:"total_changes"`
	BreakingCount   int `json:"breaking_count"`
	WarningCount    int `json:"warning_count"`
	InfoCount       int `json:"info_count"`
}

// ToJSON converts a DiffReport to the stable JSON format
func ToJSON(report *models.DiffReport, oldFile, newFile string) *JSONReport {
	return &JSONReport{
		SchemaVersion: "1.0",
		OldFile:       oldFile,
		NewFile:       newFile,
		Summary: JSONSummary{
			ServicesAdded:   report.Summary.ServicesAdded,
			ServicesRemoved: report.Summary.ServicesRemoved,
			ServicesChanged: report.Summary.ServicesChanged,
			VolumesAdded:    report.Summary.VolumesAdded,
			VolumesRemoved:  report.Summary.VolumesRemoved,
			NetworksAdded:   report.Summary.NetworksAdded,
			NetworksRemoved: report.Summary.NetworksRemoved,
			TotalChanges:    report.Summary.TotalChanges,
			BreakingCount:   report.Summary.BreakingCount,
			WarningCount:    report.Summary.WarningCount,
			InfoCount:       report.Summary.InfoCount,
		},
		Changes: report.Changes,
	}
}
