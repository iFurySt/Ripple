package substack

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ifuryst/ripple/internal/service/publisher"
	"go.uber.org/zap"
)

// SubstackPublisher handles publishing to Substack
type SubstackPublisher struct {
	logger             *zap.Logger
	contentTransformer *SubstackTransformer
	client             *http.Client
	domain             string
	cookie             string
}

// Substack API request structures
type SubstackCreateDraftRequest struct {
	DraftTitle                      string                    `json:"draft_title"`
	DraftSubtitle                   string                    `json:"draft_subtitle"`
	DraftPodcastURL                 string                    `json:"draft_podcast_url"`
	DraftPodcastDuration            *int                      `json:"draft_podcast_duration"`
	DraftVideoUploadID              *int                      `json:"draft_video_upload_id"`
	DraftPodcastUploadID            *int                      `json:"draft_podcast_upload_id"`
	DraftPodcastPreviewUploadID     *int                      `json:"draft_podcast_preview_upload_id"`
	DraftVoiceoverUploadID          *int                      `json:"draft_voiceover_upload_id"`
	DraftBody                       string                    `json:"draft_body"`
	SectionChosen                   bool                      `json:"section_chosen"`
	DraftSectionID                  *int                      `json:"draft_section_id"`
	DraftBylines                    []SubstackByline          `json:"draft_bylines"`
	Audience                        string                    `json:"audience"`
}

type SubstackByline struct {
	ID      int  `json:"id"`
	IsGuest bool `json:"is_guest"`
}

type SubstackUpdateDraftRequest struct {
	DraftTitle                      string                    `json:"draft_title"`
	DraftSubtitle                   string                    `json:"draft_subtitle"`
	DraftPodcastURL                 string                    `json:"draft_podcast_url"`
	DraftPodcastDuration            *int                      `json:"draft_podcast_duration"`
	DraftVideoUploadID              *int                      `json:"draft_video_upload_id"`
	DraftPodcastUploadID            *int                      `json:"draft_podcast_upload_id"`
	DraftPodcastPreviewUploadID     *int                      `json:"draft_podcast_preview_upload_id"`
	DraftVoiceoverUploadID          *int                      `json:"draft_voiceover_upload_id"`
	DraftBody                       string                    `json:"draft_body"`
	SectionChosen                   bool                      `json:"section_chosen"`
	DraftSectionID                  *int                      `json:"draft_section_id"`
	DraftBylines                    []SubstackByline          `json:"draft_bylines"`
	LastUpdatedAt                   string                    `json:"last_updated_at"`
}

type SubstackImageUploadRequest struct {
	Image  string `json:"image"`
	PostID int    `json:"postId"`
}

type SubstackImageUploadResponse struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
	Bytes       int    `json:"bytes"`
	ImageWidth  int    `json:"imageWidth"`
	ImageHeight int    `json:"imageHeight"`
}

type SubstackDraftResponse struct {
	ID                 int                 `json:"id"`
	UUID               string              `json:"uuid"`
	DraftTitle         string              `json:"draft_title"`
	DraftSubtitle      string              `json:"draft_subtitle"`
	DraftBody          string              `json:"draft_body"`
	DraftCreatedAt     string              `json:"draft_created_at"`
	DraftUpdatedAt     string              `json:"draft_updated_at"`
	IsPublished        bool                `json:"is_published"`
	PublicationID      int                 `json:"publication_id"`
	Type               string              `json:"type"`
	ShouldSendEmail    bool                `json:"should_send_email"`
	Audience           string              `json:"audience"`
	DraftBylines       []SubstackByline    `json:"draft_bylines"`
}

func NewSubstackPublisher(logger *zap.Logger) publisher.Publisher {
	return &SubstackPublisher{
		logger:             logger,
		contentTransformer: NewSubstackTransformer(),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (p *SubstackPublisher) GetPlatformName() string {
	return "substack"
}

func (p *SubstackPublisher) Initialize(ctx context.Context, config publisher.PublishConfig) error {
	if err := p.ValidateConfig(config); err != nil {
		return err
	}

	p.domain = config.Config["domain"]
	p.cookie = config.Config["cookie"]

	p.logger.Info("Substack publisher initialized successfully",
		zap.String("domain", p.domain))
	return nil
}

func (p *SubstackPublisher) ValidateConfig(config publisher.PublishConfig) error {
	required := []string{"domain", "cookie"}

	for _, key := range required {
		if config.Config[key] == "" {
			return fmt.Errorf("missing required config: %s", key)
		}
	}

	return nil
}

func (p *SubstackPublisher) TransformContent(ctx context.Context, content publisher.PublishContent) (*publisher.PublishContent, error) {
	// Transform content to Substack's JSON format
	transformedContent, err := p.contentTransformer.Transform(ctx, content.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to transform content: %w", err)
	}

	// Extract images from content for processing
	imageURLs := p.contentTransformer.ExtractImages(content.Content)

	// Create resources for images
	var resources []publisher.Resource
	for i, url := range imageURLs {
		resources = append(resources, publisher.Resource{
			ID:   fmt.Sprintf("substack_img_%d", i+1),
			Type: publisher.ResourceTypeImage,
			URL:  url,
		})
	}

	// Create new content with transformed data
	result := content
	result.Content = transformedContent
	result.Resources = resources
	
	// Initialize Metadata if it's nil
	if result.Metadata == nil {
		result.Metadata = make(map[string]string)
	}

	return &result, nil
}

func (p *SubstackPublisher) ProcessResources(ctx context.Context, content *publisher.PublishContent, config publisher.PublishConfig) error {
	if len(content.Resources) == 0 {
		return nil
	}

	// First, we need to create a draft to get the post ID for image uploads
	draftID := content.Metadata["draft_id"]
	if draftID == "" {
		return fmt.Errorf("draft_id not found in metadata - needed for image uploads")
	}

	postID, err := strconv.Atoi(draftID)
	if err != nil {
		return fmt.Errorf("invalid draft_id format: %w", err)
	}

	// Process each image resource
	successfulUploads := 0
	for i, resource := range content.Resources {
		if resource.Type == publisher.ResourceTypeImage {
			// Upload image to Substack
			uploadedImageURL, err := p.uploadImage(ctx, resource.URL, postID)
			if err != nil {
				p.logger.Warn("Failed to upload image, skipping", 
					zap.String("image_url", resource.URL),
					zap.Error(err))
				// Skip this image but continue with others
				continue
			}

			// Update resource with uploaded URL
			content.Resources[i].URL = uploadedImageURL
			content.Resources[i].Metadata = map[string]string{
				"uploaded_url": uploadedImageURL,
				"original_url": resource.URL,
			}
			successfulUploads++
		}
	}

	// Update content to use uploaded image URLs
	content.Content = p.contentTransformer.UpdateImageReferences(content.Content, content.Resources)

	// Store successful upload count in metadata for later use
	content.Metadata["successful_uploads"] = fmt.Sprintf("%d", successfulUploads)

	p.logger.Info("Processed Substack resources",
		zap.Int("total_images", len(content.Resources)),
		zap.Int("successful_uploads", successfulUploads))

	return nil
}

func (p *SubstackPublisher) SaveToDraft(ctx context.Context, content publisher.PublishContent, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	p.logger.Debug("Starting SaveToDraft for Substack", 
		zap.String("title", content.Title),
		zap.Int("resources_count", len(content.Resources)))
		
	// Transform content first
	transformedContent, err := p.TransformContent(ctx, content)
	if err != nil {
		p.logger.Error("Failed to transform content", zap.Error(err))
		return &publisher.PublishResult{
			Success:  false,
			Error:    err,
			ErrorMsg: err.Error(),
		}, nil
	}
	
	p.logger.Debug("Content transformed successfully", 
		zap.Int("transformed_resources_count", len(transformedContent.Resources)))

	// Create draft request
	draftRequest := SubstackCreateDraftRequest{
		DraftTitle:                      transformedContent.Title,
		DraftSubtitle:                   transformedContent.Summary,
		DraftPodcastURL:                 "",
		DraftPodcastDuration:            nil,
		DraftVideoUploadID:              nil,
		DraftPodcastUploadID:            nil,
		DraftPodcastPreviewUploadID:     nil,
		DraftVoiceoverUploadID:          nil,
		DraftBody:                       transformedContent.Content,
		SectionChosen:                   false,
		DraftSectionID:                  nil,
		DraftBylines:                    []SubstackByline{}, // Will be populated by Substack
		Audience:                        "everyone",
	}

	// Create draft
	draftResponse, err := p.createDraft(ctx, draftRequest)
	if err != nil {
		draftErr := fmt.Errorf("failed to create Substack draft: %w", err)
		return &publisher.PublishResult{
			Success:  false,
			Error:    draftErr,
			ErrorMsg: draftErr.Error(),
		}, nil
	}

	// Store draft ID for image processing
	transformedContent.Metadata["draft_id"] = fmt.Sprintf("%d", draftResponse.ID)

	// Process resources (images) now that we have a draft ID
	p.logger.Debug("Processing resources", 
		zap.Int("resource_count", len(transformedContent.Resources)),
		zap.String("draft_id", transformedContent.Metadata["draft_id"]))
		
	if err := p.ProcessResources(ctx, transformedContent, config); err != nil {
		p.logger.Error("Failed to process resources", zap.Error(err))
		resourceErr := fmt.Errorf("failed to process resources: %w", err)
		return &publisher.PublishResult{
			Success:  false,
			Error:    resourceErr,
			ErrorMsg: resourceErr.Error(),
		}, nil
	}
	
	// Get successful upload count from metadata
	successfulUploads := 0
	if successfulUploadsStr, ok := transformedContent.Metadata["successful_uploads"]; ok {
		if count, err := strconv.Atoi(successfulUploadsStr); err == nil {
			successfulUploads = count
		}
	}
	
	p.logger.Debug("Resources processed successfully", 
		zap.Int("successful_uploads", successfulUploads))

	// Note: Skip final update step as image uploads may have already updated the draft
	// and caused version conflicts (409 "Post out of date" error)
	if successfulUploads > 0 {
		p.logger.Info("Images uploaded successfully, draft auto-updated by Substack", 
			zap.Int("successful_uploads", successfulUploads),
			zap.Int("draft_id", draftResponse.ID))
	}

	p.logger.Info("Draft saved successfully",
		zap.Int("draft_id", draftResponse.ID),
		zap.String("title", transformedContent.Title))

	return &publisher.PublishResult{
		Success:   true,
		PublishID: fmt.Sprintf("%d", draftResponse.ID),
		Metadata: map[string]string{
			"draft_id":     fmt.Sprintf("%d", draftResponse.ID),
			"uuid":         draftResponse.UUID,
			"platform":     "substack",
			"draft_status": "saved",
		},
	}, nil
}

func (p *SubstackPublisher) Publish(ctx context.Context, draftID string, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	// For Substack, publishing is done through the web interface
	// The API doesn't provide a direct publish endpoint based on the documentation
	// We'll return success but indicate that manual publishing is required
	p.logger.Info("Substack draft created, manual publishing required",
		zap.String("draft_id", draftID))

	return &publisher.PublishResult{
		Success:     true,
		PublishID:   draftID,
		PublishedAt: time.Now(),
		Metadata: map[string]string{
			"draft_id":       draftID,
			"publish_status": "manual_required",
			"message":        "Draft created successfully. Please publish manually through Substack interface.",
		},
	}, nil
}

func (p *SubstackPublisher) PublishDirect(ctx context.Context, content publisher.PublishContent, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	// Save to draft first
	draftResult, err := p.SaveToDraft(ctx, content, config)
	if err != nil {
		return &publisher.PublishResult{
			Success:  false,
			Error:    err,
			ErrorMsg: err.Error(),
		}, nil
	}

	if !draftResult.Success {
		return draftResult, nil
	}

	// Auto-publish if enabled (though for Substack this means just creating the draft)
	if autoPublish := config.Config["auto_publish"]; autoPublish == "true" {
		publishResult, err := p.Publish(ctx, draftResult.PublishID, config)
		if err != nil {
			draftResult.Metadata["publish_error"] = err.Error()
			p.logger.Warn("Auto-publish not available for Substack, draft created successfully",
				zap.String("draft_id", draftResult.PublishID))
			return draftResult, nil
		}
		return publishResult, nil
	}

	return draftResult, nil
}

func (p *SubstackPublisher) GetPublishStatus(ctx context.Context, publishID string, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	// Check draft status by trying to get draft info
	draftID, err := strconv.Atoi(publishID)
	if err != nil {
		return nil, fmt.Errorf("invalid publish ID: %w", err)
	}

	exists, err := p.checkDraftExists(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to check draft status: %w", err)
	}

	return &publisher.PublishResult{
		Success:   exists,
		PublishID: publishID,
	}, nil
}

func (p *SubstackPublisher) Cleanup(ctx context.Context, publishID string, config publisher.PublishConfig) error {
	// Clean up temporary files if any
	p.logger.Info("Substack cleanup completed", zap.String("publish_id", publishID))
	return nil
}

// Helper methods

func (p *SubstackPublisher) createDraft(ctx context.Context, request SubstackCreateDraftRequest) (*SubstackDraftResponse, error) {
	url := fmt.Sprintf("https://%s/api/v1/drafts", p.domain)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal draft request: %w", err)
	}
	
	p.logger.Debug("Creating Substack draft", 
		zap.String("url", url),
		zap.String("request_body", string(jsonData)))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", p.cookie)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en,zh-CN;q=0.9,zh;q=0.8")
	req.Header.Set("Origin", fmt.Sprintf("https://%s", p.domain))
	req.Header.Set("Referer", fmt.Sprintf("https://%s/publish/post", p.domain))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Sec-Ch-Ua", `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.Error("Failed to send Substack request", zap.Error(err), zap.String("url", url))
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Error("Failed to read Substack response", zap.Error(err))
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	p.logger.Debug("Substack API response", 
		zap.Int("status_code", resp.StatusCode),
		zap.String("response_body", string(body)))

	if resp.StatusCode != http.StatusOK {
		p.logger.Error("Substack API error", 
			zap.Int("status_code", resp.StatusCode), 
			zap.String("response_body", string(body)),
			zap.String("request_url", url))
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var draftResponse SubstackDraftResponse
	if err := json.Unmarshal(body, &draftResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &draftResponse, nil
}

func (p *SubstackPublisher) updateDraft(ctx context.Context, draftID int, request SubstackUpdateDraftRequest) error {
	url := fmt.Sprintf("https://%s/api/v1/drafts/%d", p.domain, draftID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", p.cookie)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en,zh-CN;q=0.9,zh;q=0.8")
	req.Header.Set("Origin", fmt.Sprintf("https://%s", p.domain))
	req.Header.Set("Referer", fmt.Sprintf("https://%s/publish/post", p.domain))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Sec-Ch-Ua", `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (p *SubstackPublisher) uploadImage(ctx context.Context, imageURL string, postID int) (string, error) {
	// Download the image from the URL
	base64Image, err := p.downloadAndEncodeImage(ctx, imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to download and encode image: %w", err)
	}
	
	url := fmt.Sprintf("https://%s/api/v1/image", p.domain)

	request := SubstackImageUploadRequest{
		Image:  base64Image, // Base64 encoded image data
		PostID: postID,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal image upload request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", p.cookie)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en,zh-CN;q=0.9,zh;q=0.8")
	req.Header.Set("Origin", fmt.Sprintf("https://%s", p.domain))
	req.Header.Set("Referer", fmt.Sprintf("https://%s/publish/post", p.domain))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Sec-Ch-Ua", `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var uploadResponse SubstackImageUploadResponse
	if err := json.Unmarshal(body, &uploadResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return uploadResponse.URL, nil
}

func (p *SubstackPublisher) checkDraftExists(ctx context.Context, draftID int) (bool, error) {
	// This is a simplified check - in reality you'd call a specific endpoint
	// to check if the draft exists
	return true, nil
}

func (p *SubstackPublisher) downloadAndEncodeImage(ctx context.Context, imageURL string) (string, error) {
	// Download the image
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image, status: %d", resp.StatusCode)
	}

	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Encode to base64 with data URL prefix
	// Get content type from response or default to image/png
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	base64Data := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)

	p.logger.Debug("Image downloaded and encoded", 
		zap.String("url", imageURL),
		zap.String("content_type", contentType),
		zap.Int("data_size", len(imageData)))

	return dataURL, nil
}