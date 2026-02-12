package reporter

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/stackgen-cli/compose-diff/internal/models"
)

// CategorySummary holds changes grouped by category
type CategorySummary struct {
	Category  string
	Count     int
	Breaking  int
	Warning   int
	Info      int
	Changes   []models.Change
}

// ToCategorySummary generates a category-based summary report
func ToCategorySummary(report *models.DiffReport, oldFile, newFile string) string {
	var sb strings.Builder

	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	sb.WriteString(cyan("compose-diff: Category Summary\n\n"))
	sb.WriteString(fmt.Sprintf("Comparing: %s → %s\n\n", oldFile, newFile))

	// Get category summaries
	summaries := groupByCategory(report.Changes)

	if len(summaries) == 0 {
		sb.WriteString(green("No differences found.\n"))
		return sb.String()
	}

	// Print table header
	sb.WriteString("┌" + strings.Repeat("─", 20) + "┬" + strings.Repeat("─", 8) + "┬" + strings.Repeat("─", 10) + "┬" + strings.Repeat("─", 10) + "┬" + strings.Repeat("─", 8) + "┐\n")
	sb.WriteString(fmt.Sprintf("│ %-18s │ %6s │ %8s │ %8s │ %6s │\n",
		"Category", "Total", "Breaking", "Warning", "Info"))
	sb.WriteString("├" + strings.Repeat("─", 20) + "┼" + strings.Repeat("─", 8) + "┼" + strings.Repeat("─", 10) + "┼" + strings.Repeat("─", 10) + "┼" + strings.Repeat("─", 8) + "┤\n")

	// Print each category
	for _, s := range summaries {
		breakingStr := fmt.Sprintf("%d", s.Breaking)
		warningStr := fmt.Sprintf("%d", s.Warning)
		if s.Breaking > 0 {
			breakingStr = red(fmt.Sprintf("%d", s.Breaking))
		}
		if s.Warning > 0 {
			warningStr = yellow(fmt.Sprintf("%d", s.Warning))
		}

		sb.WriteString(fmt.Sprintf("│ %-18s │ %6d │ %8s │ %8s │ %6d │\n",
			s.Category, s.Count, breakingStr, warningStr, s.Info))
	}

	sb.WriteString("└" + strings.Repeat("─", 20) + "┴" + strings.Repeat("─", 8) + "┴" + strings.Repeat("─", 10) + "┴" + strings.Repeat("─", 10) + "┴" + strings.Repeat("─", 8) + "┘\n")

	// Totals
	var totalCount, totalBreaking, totalWarning, totalInfo int
	for _, s := range summaries {
		totalCount += s.Count
		totalBreaking += s.Breaking
		totalWarning += s.Warning
		totalInfo += s.Info
	}

	sb.WriteString(fmt.Sprintf("\nTotal: %d changes (%s, %s, %d info)\n",
		totalCount,
		red(fmt.Sprintf("%d breaking", totalBreaking)),
		yellow(fmt.Sprintf("%d warning", totalWarning)),
		totalInfo))

	return sb.String()
}

// ToCategoryDetail generates detailed output grouped by category
func ToCategoryDetail(report *models.DiffReport, oldFile, newFile string) string {
	var sb strings.Builder

	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	sb.WriteString(cyan("compose-diff: Category Report\n\n"))
	sb.WriteString(fmt.Sprintf("Comparing: %s → %s\n\n", oldFile, newFile))

	summaries := groupByCategory(report.Changes)

	if len(summaries) == 0 {
		sb.WriteString(green("No differences found.\n"))
		return sb.String()
	}

	for _, s := range summaries {
		var header string
		switch s.Category {
		case "environment":
			header = "🔧 Environment Variables"
		case "ports":
			header = "🔌 Port Mappings"
		case "images":
			header = "📦 Images & Builds"
		case "volumes":
			header = "💾 Volumes"
		case "networks":
			header = "🌐 Networks"
		case "deploy":
			header = "🚀 Deployment"
		default:
			header = "📋 " + titleCase(s.Category)
		}

		sb.WriteString(cyan(fmt.Sprintf("\n%s (%d changes)\n", header, s.Count)))
		sb.WriteString(strings.Repeat("─", 40) + "\n")

		for _, c := range s.Changes {
			icon := changeIcon(c.Kind, c.Severity)
			sevLabel := severityLabel(c.Severity)

			// Format service.field
			svcField := c.Name
			if c.Scope == models.ScopeService {
				field := extractField(c.Path)
				svcField = fmt.Sprintf("%s.%s", c.Name, field)
			}

			switch c.Kind {
			case models.ChangeAdded:
				sb.WriteString(fmt.Sprintf("  %s %s %s = %v\n", icon, sevLabel, svcField, formatValue(c.After)))
			case models.ChangeRemoved:
				sb.WriteString(fmt.Sprintf("  %s %s %s (removed)\n", icon, sevLabel, svcField))
			case models.ChangeModified:
				sb.WriteString(fmt.Sprintf("  %s %s %s: %v → %v\n", icon, sevLabel, svcField, formatValue(c.Before), formatValue(c.After)))
			}
		}
	}

	// Summary footer
	var totalBreaking, totalWarning int
	for _, s := range summaries {
		totalBreaking += s.Breaking
		totalWarning += s.Warning
	}

	if totalBreaking > 0 {
		sb.WriteString(fmt.Sprintf("\n%s\n", red(fmt.Sprintf("⚠️  %d breaking changes detected!", totalBreaking))))
	}
	if totalWarning > 0 {
		sb.WriteString(fmt.Sprintf("%s\n", yellow(fmt.Sprintf("⚡ %d warnings", totalWarning))))
	}

	return sb.String()
}

// groupByCategory groups changes by their category
func groupByCategory(changes []models.Change) []CategorySummary {
	categories := make(map[string]*CategorySummary)

	for _, c := range changes {
		cat := categorizeChange(c)
		if _, ok := categories[cat]; !ok {
			categories[cat] = &CategorySummary{
				Category: cat,
			}
		}

		cs := categories[cat]
		cs.Count++
		cs.Changes = append(cs.Changes, c)

		switch c.Severity {
		case models.SeverityBreaking:
			cs.Breaking++
		case models.SeverityWarning:
			cs.Warning++
		default:
			cs.Info++
		}
	}

	// Convert to slice and sort by importance
	result := make([]CategorySummary, 0, len(categories))
	for _, cs := range categories {
		result = append(result, *cs)
	}

	sort.Slice(result, func(i, j int) bool {
		// Sort by breaking count, then warning, then total
		if result[i].Breaking != result[j].Breaking {
			return result[i].Breaking > result[j].Breaking
		}
		if result[i].Warning != result[j].Warning {
			return result[i].Warning > result[j].Warning
		}
		return result[i].Count > result[j].Count
	})

	return result
}

// categorizeChange determines the category of a change
func categorizeChange(c models.Change) string {
	path := c.Path

	if strings.Contains(path, ".environment.") || strings.Contains(path, ".env_file") {
		return "environment"
	}
	if strings.Contains(path, ".ports.") || strings.Contains(path, ".expose.") {
		return "ports"
	}
	if strings.Contains(path, ".image") || strings.Contains(path, ".build.") {
		return "images"
	}
	if strings.Contains(path, ".volumes.") {
		return "volumes"
	}
	if strings.Contains(path, ".networks.") {
		return "networks"
	}
	if strings.Contains(path, ".deploy.") || strings.Contains(path, ".replicas") ||
		strings.Contains(path, ".resources.") {
		return "deploy"
	}
	if strings.Contains(path, ".depends_on") || strings.Contains(path, ".healthcheck") {
		return "dependencies"
	}
	if strings.Contains(path, ".labels.") {
		return "labels"
	}
	if strings.Contains(path, ".logging") {
		return "logging"
	}
	if c.Scope == models.ScopeVolume {
		return "volumes"
	}
	if c.Scope == models.ScopeNetwork {
		return "networks"
	}

	return "other"
}

// titleCase converts a string to title case (first letter uppercased)
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
