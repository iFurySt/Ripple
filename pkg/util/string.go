package util

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// EscapeYAML escapes special YAML characters in strings
func EscapeYAML(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// GenerateSlug creates a URL-friendly slug from title
func GenerateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9\p{Han}]+`) // Allow Chinese characters
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
		slug = strings.Trim(slug, "-")
	}

	return slug
}

// GenerateFilename creates a Jekyll post filename
func GenerateFilename(title string, date time.Time) string {
	slug := GenerateSlug(title)
	dateStr := date.Format("2006-01-02")
	return fmt.Sprintf("%s-%s.md", dateStr, slug)
}

// GenerateFilenameWithMetadata creates a Jekyll post filename using metadata
func GenerateFilenameWithMetadata(title string, date time.Time, metadata map[string]string) string {
	// Use EN title if available, fallback to regular title
	titleForSlug := title
	if enTitle, exists := metadata["en_title"]; exists && enTitle != "" {
		titleForSlug = enTitle
	}

	slug := GenerateSlug(titleForSlug)
	dateStr := date.Format("2006-01-02")
	return fmt.Sprintf("%s-%s.md", dateStr, slug)
}

// GenerateImageDir creates the image directory name for a post
func GenerateImageDir(title string, date time.Time) string {
	slug := GenerateSlug(title)
	dateStr := date.Format("2006-01-02")
	return fmt.Sprintf("%s-%s", dateStr, slug)
}

// GenerateImageDirWithMetadata creates the image directory name for a post using metadata
func GenerateImageDirWithMetadata(title string, date time.Time, metadata map[string]string) string {
	// Use EN title if available, fallback to regular title
	titleForSlug := title
	if enTitle, exists := metadata["en_title"]; exists && enTitle != "" {
		titleForSlug = enTitle
	}

	slug := GenerateSlug(titleForSlug)
	dateStr := date.Format("2006-01-02")
	return fmt.Sprintf("%s-%s", dateStr, slug)
}

// ParseTags parses tag strings into arrays
func ParseTags(tagStr string) []string {
	if tagStr == "" {
		return []string{}
	}

	// Remove brackets if present
	tagStr = strings.Trim(tagStr, "[]")

	// Split by comma and clean up
	tags := strings.Split(tagStr, ",")
	var cleanTags []string

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		tag = strings.Trim(tag, "\"'") // Remove quotes
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}

	return cleanTags
}