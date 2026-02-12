package diff

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/stackgen-cli/compose-diff/internal/models"
)

// imageSeverity determines the severity of an image change
func imageSeverity(old, new *string) models.Severity {
	if old == nil || new == nil {
		return models.SeverityInfo
	}

	oldTag := extractTag(*old)
	newTag := extractTag(*new)

	// Same tag or both latest
	if oldTag == newTag {
		return models.SeverityInfo
	}

	// Check for major version change
	oldMajor := extractMajorVersion(oldTag)
	newMajor := extractMajorVersion(newTag)

	if oldMajor != "" && newMajor != "" && oldMajor != newMajor {
		return models.SeverityWarning // Major version change
	}

	return models.SeverityInfo
}

// extractTag extracts the tag from an image reference
// e.g., "postgres:16-alpine" -> "16-alpine"
// e.g., "myregistry/app:v1.2.3" -> "v1.2.3"
func extractTag(image string) string {
	// Handle digest-based references
	if strings.Contains(image, "@") {
		return ""
	}

	parts := strings.Split(image, ":")
	if len(parts) < 2 {
		return "latest"
	}

	// The tag is after the last colon, but need to handle ports in registry
	// registry:port/image:tag
	lastColon := strings.LastIndex(image, ":")
	if lastColon == -1 {
		return "latest"
	}

	tag := image[lastColon+1:]
	// If tag contains /, it's probably a port not a tag
	if strings.Contains(tag, "/") {
		return "latest"
	}

	return tag
}

// extractMajorVersion extracts the major version from a tag
// e.g., "16-alpine" -> "16"
// e.g., "v1.2.3" -> "1"
// e.g., "3.9" -> "3"
func extractMajorVersion(tag string) string {
	if tag == "" || tag == "latest" {
		return ""
	}

	// Remove common prefixes
	tag = strings.TrimPrefix(tag, "v")

	// Extract leading numbers
	re := regexp.MustCompile(`^(\d+)`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// IsBreakingChange determines if a change is potentially breaking based on heuristics
func IsBreakingChange(c models.Change) bool {
	// Removals are generally breaking
	if c.Kind == models.ChangeRemoved {
		switch {
		case strings.Contains(c.Path, ".environment."):
			return true // Env var removed
		case strings.Contains(c.Path, ".ports."):
			return true // Port removed
		case strings.Contains(c.Path, ".volumes."):
			return true // Volume mount removed
		case strings.Contains(c.Path, ".depends_on."):
			return true // Dependency removed
		case strings.Contains(c.Path, ".healthcheck"):
			return true // Healthcheck removed
		case c.Scope == models.ScopeService:
			return true // Service removed
		case c.Scope == models.ScopeVolume:
			return true // Named volume removed
		}
	}

	// Image major version changes
	if strings.HasSuffix(c.Path, ".image") && c.Kind == models.ChangeModified {
		if before, ok := c.Before.(string); ok {
			if after, ok := c.After.(string); ok {
				oldMajor := extractMajorVersion(extractTag(before))
				newMajor := extractMajorVersion(extractTag(after))
				if oldMajor != "" && newMajor != "" {
					oldNum, _ := strconv.Atoi(oldMajor)
					newNum, _ := strconv.Atoi(newMajor)
					if newNum < oldNum {
						return true // Downgrade
					}
				}
			}
		}
	}

	return false
}
