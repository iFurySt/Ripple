package notion

import (
	"fmt"
	"github.com/ifuryst/ripple/internal/models"
	"time"
)

func (s *Service) extractTitle(properties map[string]any) string {
	// Look for title property
	for _, prop := range properties {
		if propMap, ok := prop.(map[string]any); ok {
			if propMap["type"] == "title" {
				if title, ok := propMap["title"].([]any); ok && len(title) > 0 {
					if titleObj, ok := title[0].(map[string]any); ok {
						if plainText, ok := titleObj["plain_text"].(string); ok {
							return plainText
						}
					}
				}
			}
		}
	}
	return "Untitled"
}

func (s *Service) extractTags(properties map[string]any) models.StringArray {
	// Look for tags/multi_select property
	for _, prop := range properties {
		if propMap, ok := prop.(map[string]any); ok {
			if propMap["type"] == "multi_select" {
				if tags, ok := propMap["multi_select"].([]any); ok {
					var tagNames []string
					for _, tag := range tags {
						if tagMap, ok := tag.(map[string]any); ok {
							if name, ok := tagMap["name"].(string); ok {
								tagNames = append(tagNames, name)
							}
						}
					}
					return tagNames
				}
			}
		}
	}
	return models.StringArray{}
}

func (s *Service) extractStatus(properties map[string]any) string {
	// Look for status property
	for _, prop := range properties {
		if propMap, ok := prop.(map[string]any); ok {
			if propMap["type"] == "status" {
				if statusObj, ok := propMap["status"].(map[string]any); ok {
					if name, ok := statusObj["name"].(string); ok {
						return name
					}
				}
			}
		}
	}
	return "draft"
}

func (s *Service) extractENTitle(properties map[string]any) string {
	// Look for EN Title rich_text property
	for propName, prop := range properties {
		if propName == "EN Title" {
			if propMap, ok := prop.(map[string]any); ok {
				if propMap["type"] == "rich_text" {
					if richText, ok := propMap["rich_text"].([]any); ok && len(richText) > 0 {
						if textObj, ok := richText[0].(map[string]any); ok {
							if plainText, ok := textObj["plain_text"].(string); ok {
								return plainText
							}
						}
					}
				}
			}
		}
	}
	return ""
}

func (s *Service) extractPostDate(properties map[string]any) *time.Time {
	// Look for Post date property
	for propName, prop := range properties {
		if propName == "Post date" {
			if propMap, ok := prop.(map[string]any); ok {
				if propMap["type"] == "date" {
					if dateObj, ok := propMap["date"].(map[string]any); ok {
						if startStr, ok := dateObj["start"].(string); ok {
							if date, err := time.Parse("2006-01-02", startStr); err == nil {
								return &date
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *Service) extractOwner(properties map[string]any) string {
	// Look for Owner people property
	for propName, prop := range properties {
		if propName == "Owner" {
			if propMap, ok := prop.(map[string]any); ok {
				if propMap["type"] == "people" {
					if people, ok := propMap["people"].([]any); ok && len(people) > 0 {
						var owners []string
						for _, person := range people {
							if personMap, ok := person.(map[string]any); ok {
								if name, ok := personMap["name"].(string); ok {
									owners = append(owners, name)
								}
							}
						}
						if len(owners) > 0 {
							return fmt.Sprintf("%v", owners)
						}
					}
				}
			}
		}
	}
	return ""
}

func (s *Service) extractPlatforms(properties map[string]any) models.StringArray {
	// Look for Platform multi_select property
	for propName, prop := range properties {
		if propName == "Platform" {
			if propMap, ok := prop.(map[string]any); ok {
				if propMap["type"] == "multi_select" {
					if platforms, ok := propMap["multi_select"].([]any); ok {
						var platformNames []string
						for _, platform := range platforms {
							if platformMap, ok := platform.(map[string]any); ok {
								if name, ok := platformMap["name"].(string); ok {
									platformNames = append(platformNames, name)
								}
							}
						}
						return platformNames
					}
				}
			}
		}
	}
	return models.StringArray{}
}

func (s *Service) extractContentType(properties map[string]any) models.StringArray {
	// Look for Content type multi_select property
	for propName, prop := range properties {
		if propName == "Content type" {
			if propMap, ok := prop.(map[string]any); ok {
				if propMap["type"] == "multi_select" {
					if contentTypes, ok := propMap["multi_select"].([]any); ok {
						var typeNames []string
						for _, contentType := range contentTypes {
							if typeMap, ok := contentType.(map[string]any); ok {
								if name, ok := typeMap["name"].(string); ok {
									typeNames = append(typeNames, name)
								}
							}
						}
						return typeNames
					}
				}
			}
		}
	}
	return models.StringArray{}
}
