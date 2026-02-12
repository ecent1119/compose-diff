package reporter

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/stackgen-cli/compose-diff/internal/models"
)

// ToText generates a human-readable text report
func ToText(report *models.DiffReport, oldFile, newFile string) string {
	var sb strings.Builder

	// Header
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	sb.WriteString(cyan("compose-diff\n\n"))
	sb.WriteString(fmt.Sprintf("Comparing: %s → %s\n\n", oldFile, newFile))

	// Summary
	s := report.Summary
	sb.WriteString(fmt.Sprintf("Summary: %d services changed, %d added, %d removed\n",
		s.ServicesChanged, s.ServicesAdded, s.ServicesRemoved))
	sb.WriteString(fmt.Sprintf("         %d changes (%s, %s, %d info)\n\n",
		s.TotalChanges,
		red(fmt.Sprintf("%d breaking", s.BreakingCount)),
		yellow(fmt.Sprintf("%d warnings", s.WarningCount)),
		s.InfoCount))

	if s.TotalChanges == 0 {
		sb.WriteString(green("No differences found.\n"))
		return sb.String()
	}

	sb.WriteString(strings.Repeat("━", 50) + "\n\n")

	// Group changes by service/scope
	byService := groupByService(report.Changes)

	for _, svc := range sortedKeys(byService) {
		changes := byService[svc]
		sb.WriteString(fmt.Sprintf("Service: %s\n", cyan(svc)))

		for _, c := range changes {
			icon := changeIcon(c.Kind, c.Severity)
			sevLabel := severityLabel(c.Severity)
			field := extractField(c.Path)

			switch c.Kind {
			case models.ChangeAdded:
				sb.WriteString(fmt.Sprintf("  %s %s %s = %v\n", icon, sevLabel, field, formatValue(c.After)))
			case models.ChangeRemoved:
				sb.WriteString(fmt.Sprintf("  %s %s %s removed\n", icon, sevLabel, field))
			case models.ChangeModified:
				sb.WriteString(fmt.Sprintf("  %s %s %s changed: %v → %v\n", icon, sevLabel, field, formatValue(c.Before), formatValue(c.After)))
			}
		}
		sb.WriteString("\n")
	}

	// Also show volume/network changes
	volNetChanges := filterVolNetChanges(report.Changes)
	if len(volNetChanges) > 0 {
		sb.WriteString("Top-level changes:\n")
		for _, c := range volNetChanges {
			icon := changeIcon(c.Kind, c.Severity)
			sevLabel := severityLabel(c.Severity)
			sb.WriteString(fmt.Sprintf("  %s %s %s %s\n", icon, sevLabel, c.Scope, c.Name))
		}
	}

	return sb.String()
}

func changeIcon(kind models.ChangeKind, severity models.Severity) string {
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	switch kind {
	case models.ChangeAdded:
		return green("➕")
	case models.ChangeRemoved:
		if severity == models.SeverityBreaking {
			return red("⚠️")
		}
		return yellow("➖")
	case models.ChangeModified:
		if severity == models.SeverityBreaking {
			return red("⚠️")
		}
		if severity == models.SeverityWarning {
			return yellow("⚡")
		}
		return "🔄"
	}
	return "•"
}

func severityLabel(s models.Severity) string {
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	switch s {
	case models.SeverityBreaking:
		return red("BREAKING")
	case models.SeverityWarning:
		return yellow("WARNING ")
	default:
		return "INFO    "
	}
}

func extractField(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) >= 3 {
		return strings.Join(parts[2:], ".")
	}
	return path
}

func formatValue(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		if len(val) > 50 {
			return fmt.Sprintf("%q...", val[:47])
		}
		return fmt.Sprintf("%q", val)
	default:
		s := fmt.Sprintf("%v", v)
		if len(s) > 50 {
			return s[:47] + "..."
		}
		return s
	}
}

func groupByService(changes []models.Change) map[string][]models.Change {
	result := make(map[string][]models.Change)
	for _, c := range changes {
		if c.Scope == models.ScopeService {
			result[c.Name] = append(result[c.Name], c)
		}
	}
	return result
}

func filterVolNetChanges(changes []models.Change) []models.Change {
	var result []models.Change
	for _, c := range changes {
		if c.Scope == models.ScopeVolume || c.Scope == models.ScopeNetwork {
			result = append(result, c)
		}
	}
	return result
}

func sortedKeys(m map[string][]models.Change) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple bubble sort for small maps
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
