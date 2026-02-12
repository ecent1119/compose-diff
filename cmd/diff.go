package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/stackgen-cli/compose-diff/internal/baseline"
	"github.com/stackgen-cli/compose-diff/internal/diff"
	"github.com/stackgen-cli/compose-diff/internal/models"
	"github.com/stackgen-cli/compose-diff/internal/parser"
	"github.com/stackgen-cli/compose-diff/internal/reporter"
	"github.com/stackgen-cli/compose-diff/internal/rules"
	"gopkg.in/yaml.v3"
)

var (
	formatFlag       string
	serviceFilter    string
	severityMin      string
	strictMode       bool
	normalizeOn      bool
	rulesFile        string
	baselineFlag     string
	saveBaseline     string
	resolveConfig    bool
	categoryMode     bool
	categoryDetail   bool
)

var diffCmd = &cobra.Command{
	Use:   "diff <old-compose.yml> <new-compose.yml>",
	Short: "Compare two Docker Compose files",
	Long: `Compare two Docker Compose configurations and report semantic differences.

Examples:
  compose-diff diff docker-compose.old.yml docker-compose.new.yml
  compose-diff diff --format json old.yml new.yml
  compose-diff diff --service api old.yml new.yml
  compose-diff diff --strict old.yml new.yml
  
  # Use resolved config (interpolated)
  compose-diff diff --resolve old.yml new.yml
  
  # Compare against saved baseline
  compose-diff diff --baseline production new.yml
  compose-diff diff --save-baseline production docker-compose.yml
  
  # Category summary
  compose-diff diff --category old.yml new.yml
  compose-diff diff --category-detail old.yml new.yml
  
  # Use custom rules file
  compose-diff diff --rules .compose-diff.yaml old.yml new.yml`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runDiff,
}

func init() {
	diffCmd.Flags().StringVarP(&formatFlag, "format", "f", "text", "Output format: text, json, markdown")
	diffCmd.Flags().StringVarP(&serviceFilter, "service", "s", "", "Filter to specific service")
	diffCmd.Flags().StringVar(&severityMin, "severity", "info", "Minimum severity: info, warning, breaking")
	diffCmd.Flags().BoolVar(&strictMode, "strict", false, "Exit 1 if breaking changes detected")
	diffCmd.Flags().BoolVar(&normalizeOn, "normalize", true, "Normalize configs before diff")

	// New flags
	diffCmd.Flags().StringVar(&rulesFile, "rules", "", "Path to rules file (default: .compose-diff.yaml)")
	diffCmd.Flags().StringVar(&baselineFlag, "baseline", "", "Compare against saved baseline")
	diffCmd.Flags().StringVar(&saveBaseline, "save-baseline", "", "Save current config as baseline")
	diffCmd.Flags().BoolVar(&resolveConfig, "resolve", false, "Use docker compose config resolved output")
	diffCmd.Flags().BoolVar(&categoryMode, "category", false, "Show category summary report")
	diffCmd.Flags().BoolVar(&categoryDetail, "category-detail", false, "Show detailed category report")

	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) {
	var oldFile, newFile string
	var oldIR, newIR *models.ComposeIR
	var err error

	// Load rules
	var r *rules.Rules
	if rulesFile != "" {
		r, err = rules.LoadRules(rulesFile)
		if err != nil {
			color.Red("Error loading rules: %v", err)
			os.Exit(2)
		}
	} else {
		r, _ = rules.LoadRulesFromDir(".")
	}

	baselineMgr := baseline.NewManager(".compose-diff")

	// Handle save-baseline mode
	if saveBaseline != "" {
		if len(args) < 1 {
			color.Red("Usage: compose-diff diff --save-baseline <name> <compose-file>")
			os.Exit(2)
		}
		composeFile := args[0]
		var data map[string]any

		if resolveConfig {
			data, err = parseResolved(composeFile)
		} else {
			data, err = parseRaw(composeFile)
		}
		if err != nil {
			color.Red("Error: %v", err)
			os.Exit(2)
		}

		if err := baselineMgr.Save(saveBaseline, data, composeFile, resolveConfig); err != nil {
			color.Red("Error saving baseline: %v", err)
			os.Exit(2)
		}
		color.Green("Saved baseline '%s'", saveBaseline)
		return
	}

	// Handle baseline comparison
	if baselineFlag != "" {
		if len(args) < 1 {
			color.Red("Usage: compose-diff diff --baseline <name> <compose-file>")
			os.Exit(2)
		}
		newFile = args[0]
		oldFile = "(baseline: " + baselineFlag + ")"

		bl, err := baselineMgr.Load(baselineFlag)
		if err != nil {
			color.Red("Error loading baseline '%s': %v", baselineFlag, err)
			os.Exit(2)
		}

		oldIR, err = parser.ParseFromMap(bl.Data)
		if err != nil {
			color.Red("Error parsing baseline: %v", err)
			os.Exit(2)
		}

		if resolveConfig {
			newIR, err = parseResolvedToIR(newFile)
		} else {
			newIR, err = parser.ParseComposeFile(newFile)
		}
		if err != nil {
			color.Red("Error parsing %s: %v", newFile, err)
			os.Exit(2)
		}
	} else {
		// Standard two-file comparison
		if len(args) < 2 {
			color.Red("Usage: compose-diff diff <old-compose.yml> <new-compose.yml>")
			os.Exit(2)
		}
		oldFile = args[0]
		newFile = args[1]

		// Parse files (possibly with resolve)
		if resolveConfig {
			oldIR, err = parseResolvedToIR(oldFile)
			if err != nil {
				color.Red("Error resolving %s: %v", oldFile, err)
				os.Exit(2)
			}
			newIR, err = parseResolvedToIR(newFile)
			if err != nil {
				color.Red("Error resolving %s: %v", newFile, err)
				os.Exit(2)
			}
		} else {
			oldIR, err = parser.ParseComposeFile(oldFile)
			if err != nil {
				color.Red("Error parsing %s: %v", oldFile, err)
				os.Exit(2)
			}
			newIR, err = parser.ParseComposeFile(newFile)
			if err != nil {
				color.Red("Error parsing %s: %v", newFile, err)
				os.Exit(2)
			}
		}
	}

	// Normalize if enabled
	if normalizeOn {
		oldIR = parser.Normalize(oldIR)
		newIR = parser.Normalize(newIR)
	}

	// Compute diff
	report := diff.Compare(oldIR, newIR)

	// Apply rules-based severity overrides and filtering
	if r != nil {
		report = applyRules(report, r)
	}

	// Filter by service if specified
	if serviceFilter != "" {
		report = diff.FilterByService(report, serviceFilter)
	}

	// Filter by severity
	report = diff.FilterBySeverity(report, severityMin)

	// Output
	var output string
	switch {
	case categoryDetail:
		output = reporter.ToCategoryDetail(report, oldFile, newFile)
	case categoryMode:
		output = reporter.ToCategorySummary(report, oldFile, newFile)
	case formatFlag == "json":
		jsonBytes, err := json.MarshalIndent(reporter.ToJSON(report, oldFile, newFile), "", "  ")
		if err != nil {
			color.Red("Error generating JSON: %v", err)
			os.Exit(2)
		}
		output = string(jsonBytes)
	case formatFlag == "markdown":
		output = reporter.ToMarkdown(report, oldFile, newFile)
	default:
		output = reporter.ToText(report, oldFile, newFile)
	}

	fmt.Println(output)

	// Exit code handling
	if strictMode && report.Summary.BreakingCount > 0 {
		os.Exit(1)
	}
}

// parseRaw parses a compose file to raw map
func parseRaw(composeFile string) (map[string]any, error) {
	data, err := os.ReadFile(composeFile)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// parseResolved uses docker compose config to get resolved output
func parseResolved(composeFile string) (map[string]any, error) {
	dir := filepath.Dir(composeFile)
	if dir == "" {
		dir = "."
	}

	cmd := exec.Command("docker", "compose", "-f", filepath.Base(composeFile), "config")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker compose config failed: %w", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(output, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// parseResolvedToIR parses resolved config to IR
func parseResolvedToIR(composeFile string) (*models.ComposeIR, error) {
	data, err := parseResolved(composeFile)
	if err != nil {
		return nil, err
	}
	return parser.ParseFromMap(data)
}

// applyRules applies rules-based modifications to the report
func applyRules(report *models.DiffReport, r *rules.Rules) *models.DiffReport {
	var filtered []models.Change
	var breakingCount, warningCount, infoCount int

	for _, c := range report.Changes {
		// Check if should be ignored
		if ignore, _ := r.ShouldIgnore(c.Path); ignore {
			continue
		}

		// Check service-specific ignores
		if r.ShouldIgnoreServiceField(c.Name, extractFieldFromPath(c.Path)) {
			continue
		}

		// Apply severity overrides
		if severity, ok := r.GetSeverityOverride(c.Path); ok {
			c.Severity = severity
		}

		filtered = append(filtered, c)

		// Recount
		switch c.Severity {
		case models.SeverityBreaking:
			breakingCount++
		case models.SeverityWarning:
			warningCount++
		default:
			infoCount++
		}
	}

	report.Changes = filtered
	report.Summary.TotalChanges = len(filtered)
	report.Summary.BreakingCount = breakingCount
	report.Summary.WarningCount = warningCount
	report.Summary.InfoCount = infoCount

	return report
}

func extractFieldFromPath(path string) string {
	// Extract field from path like "services.api.environment.DEBUG"
	parts := splitPath(path)
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, r := range path {
		if r == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
