package diff

import (
	"testing"
)

func TestExtractTag(t *testing.T) {
	tests := []struct {
		image    string
		expected string
	}{
		{"postgres:16-alpine", "16-alpine"},
		{"node:18", "18"},
		{"redis:latest", "latest"},
		{"myregistry.com/app:v1.2.3", "v1.2.3"},
		{"nginx", "latest"},
		{"postgres@sha256:abc123", ""},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			result := extractTag(tt.image)
			if result != tt.expected {
				t.Errorf("extractTag(%q) = %q, want %q", tt.image, result, tt.expected)
			}
		})
	}
}

func TestExtractMajorVersion(t *testing.T) {
	tests := []struct {
		tag      string
		expected string
	}{
		{"16-alpine", "16"},
		{"18", "18"},
		{"v1.2.3", "1"},
		{"3.9", "3"},
		{"latest", ""},
		{"alpine", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result := extractMajorVersion(tt.tag)
			if result != tt.expected {
				t.Errorf("extractMajorVersion(%q) = %q, want %q", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestImageSeverity(t *testing.T) {
	tests := []struct {
		old      string
		new      string
		expected string
	}{
		{"node:18", "node:20", "warning"},  // Major version change
		{"node:18", "node:18.1", "info"},   // Minor change
		{"postgres:16-alpine", "postgres:17-alpine", "warning"},
		{"redis:7", "redis:7", "info"},     // Same
	}

	for _, tt := range tests {
		t.Run(tt.old+"->"+tt.new, func(t *testing.T) {
			oldPtr := &tt.old
			newPtr := &tt.new
			result := imageSeverity(oldPtr, newPtr)
			if string(result) != tt.expected {
				t.Errorf("imageSeverity(%q, %q) = %q, want %q", tt.old, tt.new, result, tt.expected)
			}
		})
	}
}
