package reporter

import (
	"fmt"
	"strings"

	"github.com/stackgen-cli/compose-diff/internal/models"
)

// ToMarkdown generates a Markdown report suitable for PR comments
func ToMarkdown(report *models.DiffReport, oldFile, newFile string) string {
	var sb strings.Builder

	// Header
	sb.WriteString("## Docker Compose Diff\n\n")
	sb.WriteString(fmt.Sprintf("**Comparing:** `%s` → `%s`\n\n", oldFile, newFile))

	// Summary
	s := report.Summary
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("| Metric | Count |\n"))
	sb.WriteString(fmt.Sprintf("|--------|-------|\n"))
	sb.WriteString(fmt.Sprintf("| Services Changed | %d |\n", s.ServicesChanged))
	sb.WriteString(fmt.Sprintf("| Services Added | %d |\n", s.ServicesAdded))
	sb.WriteString(fmt.Sprintf("| Services Removed | %d |\n", s.ServicesRemoved))
	sb.WriteString(fmt.Sprintf("| Total Changes | %d |\n", s.TotalChanges))

	if s.BreakingCount > 0 {
		sb.WriteString(fmt.Sprintf("| ⚠️ **Breaking Changes** | **%d** |\n", s.BreakingCount))
	}
	if s.WarningCount > 0 {
		sb.WriteString(fmt.Sprintf("| ⚡ Warnings | %d |\n", s.WarningCount))
	}

	sb.WriteString("\n")

	if s.TotalChanges == 0 {
		sb.WriteString("✅ No differences found.\n")
		return sb.String()
	}

	// Breaking changes first
	breakingChanges := filterBySeverity(report.Changes, models.SeverityBreaking)
	if len(breakingChanges) > 0 {
		sb.WriteString("### ⚠️ Breaking Changes\n\n")
		sb.WriteString("| Service | Field | Change |\n")
		sb.WriteString("|---------|-------|--------|\n")
		for _, c := range breakingChanges {
			field := extractField(c.Path)
			change := formatChangeDescription(c)
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", c.Name, field, change))
		}
		sb.WriteString("\n")
	}

	// Warnings
	warningChanges := filterBySeverity(report.Changes, models.SeverityWarning)
	if len(warningChanges) > 0 {
		sb.WriteString("### ⚡ Warnings\n\n")
		sb.WriteString("| Service | Field | Change |\n")
		sb.WriteString("|---------|-------|--------|\n")
		for _, c := range warningChanges {
			field := extractField(c.Path)
			change := formatChangeDescription(c)
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", c.Name, field, change))
		}
		sb.WriteString("\n")
	}

	// Info changes (collapsed by default in long reports)
	infoChanges := filterBySeverity(report.Changes, models.SeverityInfo)
	if len(infoChanges) > 0 {
		if len(infoChanges) > 5 {
			sb.WriteString("<details>\n<summary>ℹ️ Info Changes (" + fmt.Sprintf("%d", len(infoChanges)) + ")</summary>\n\n")
		} else {
			sb.WriteString("### ℹ️ Info Changes\n\n")
		}

		sb.WriteString("| Service | Field | Change |\n")
		sb.WriteString("|---------|-------|--------|\n")
		for _, c := range infoChanges {
			field := extractField(c.Path)
			change := formatChangeDescription(c)
			name := c.Name
			if c.Scope != models.ScopeService {
				name = fmt.Sprintf("(%s)", c.Scope)
			}
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", name, field, change))
		}

		if len(infoChanges) > 5 {
			sb.WriteString("\n</details>\n")
		}
	}

	return sb.String()
}

func filterBySeverity(changes []models.Change, severity models.Severity) []models.Change {
	var result []models.Change
	for _, c := range changes {
		if c.Severity == severity {
			result = append(result, c)
		}
	}
	return result
}

func formatChangeDescription(c models.Change) string {
	switch c.Kind {
	case models.ChangeAdded:
		return fmt.Sprintf("Added: `%v`", truncateValue(c.After))
	case models.ChangeRemoved:
		return fmt.Sprintf("Removed (was: `%v`)", truncateValue(c.Before))
	case models.ChangeModified:
		return fmt.Sprintf("`%v` → `%v`", truncateValue(c.Before), truncateValue(c.After))
	}
	return ""
}

func truncateValue(v interface{}) string {
	if v == nil {
		return "null"
	}
	s := fmt.Sprintf("%v", v)
	if len(s) > 30 {
		return s[:27] + "..."
	}
	return s
}
