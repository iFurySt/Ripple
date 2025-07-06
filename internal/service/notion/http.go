package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
)

func (s *Service) queryDatabase(cursor string) (*DatabaseResponse, error) {
	url := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", s.config.DatabaseID)

	body := map[string]any{
		"page_size": 100,
		"filter": map[string]any{
			"property": "Status",
			"status": map[string]any{
				"equals": "Done",
			},
		},
	}
	if cursor != "" {
		body["start_cursor"] = cursor
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", s.config.APIVersion)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("notion API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response DatabaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// getAllBlocksRecursively recursively fetches all blocks including children of blocks that have has_children: true
func (s *Service) getAllBlocksRecursively(blockID string) ([]map[string]any, error) {
	var allBlocks []map[string]any
	cursor := ""

	// Loop through all pages of content
	pageCount := 0
	for {
		pageCount++
		blocks, nextCursor, hasMore, err := s.getPageBlocks(blockID, cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to get page blocks: %w", err)
		}

		// Process each block and recursively fetch children if has_children is true
		for _, block := range blocks {
			// Add the current block
			allBlocks = append(allBlocks, block)

			// Check if this block has children
			if hasChildren, ok := block["has_children"].(bool); ok && hasChildren {
				if blockIDStr, ok := block["id"].(string); ok {
					s.logger.Debug("Fetching children for block",
						zap.String("block_id", blockIDStr),
						zap.String("block_type", getBlockType(block)))

					// Recursively fetch children
					children, err := s.getAllBlocksRecursively(blockIDStr)
					if err != nil {
						s.logger.Warn("Failed to fetch children blocks",
							zap.String("block_id", blockIDStr),
							zap.Error(err))
						continue
					}

					// Add children blocks
					allBlocks = append(allBlocks, children...)
				}
			}
		}

		s.logger.Debug("Retrieved page content",
			zap.String("block_id", blockID),
			zap.Int("page_number", pageCount),
			zap.Int("blocks_in_page", len(blocks)),
			zap.Int("total_blocks", len(allBlocks)),
			zap.Bool("has_more", hasMore))

		// Check if there are more pages
		if !hasMore {
			break
		}
		cursor = nextCursor
	}

	return allBlocks, nil
}

func (s *Service) getPageBlocks(pageID, cursor string) ([]map[string]any, string, bool, error) {
	url := fmt.Sprintf("https://api.notion.com/v1/blocks/%s/children", pageID)

	// Add pagination parameters if cursor is provided
	if cursor != "" {
		url += "?start_cursor=" + cursor
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.Token)
	req.Header.Set("Notion-Version", s.config.APIVersion)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", false, fmt.Errorf("notion API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Results    []map[string]any `json:"results"`
		NextCursor string           `json:"next_cursor"`
		HasMore    bool             `json:"has_more"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, "", false, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Results, response.NextCursor, response.HasMore, nil
}

// getBlockType extracts the block type from a block object
func getBlockType(block map[string]any) string {
	if blockType, ok := block["type"].(string); ok {
		return blockType
	}
	return "unknown"
}
