package wechat_official

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ifuryst/ripple/internal/service/publisher"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// WeChatMediaProcessor handles WeChat media upload and management
type WeChatMediaProcessor struct {
	logger      *zap.Logger
	client      *http.Client
	accessToken string
}

// WeChatMediaResponse represents WeChat media upload response
type WeChatMediaResponse struct {
	Type      string `json:"type"`
	MediaID   string `json:"media_id"`
	CreatedAt int64  `json:"created_at"`
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
}

// WeChatMaterialAddResponse represents permanent material upload response
type WeChatMaterialAddResponse struct {
	MediaID string `json:"media_id"`
	URL     string `json:"url"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// WeChatUploadImageResponse represents uploadimg API response
type WeChatUploadImageResponse struct {
	URL     string `json:"url"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func NewWeChatMediaProcessor(logger *zap.Logger) *WeChatMediaProcessor {
	return &WeChatMediaProcessor{
		logger: logger,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (p *WeChatMediaProcessor) SetAccessToken(token string) {
	p.accessToken = token
}

func (p *WeChatMediaProcessor) GetSupportedTypes() []publisher.ResourceType {
	return []publisher.ResourceType{
		publisher.ResourceTypeImage,
		publisher.ResourceTypeVideo,
		publisher.ResourceTypeFile,
	}
}

func (p *WeChatMediaProcessor) ProcessResource(ctx context.Context, resource publisher.Resource, config publisher.PublishConfig) (*publisher.Resource, error) {
	if resource.Type != publisher.ResourceTypeImage {
		return &resource, nil // Only process images for now
	}

	// Download image if it's a URL
	localPath := resource.LocalPath
	if localPath == "" && resource.URL != "" {
		var err error
		localPath, err = p.downloadImage(ctx, resource.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to download image: %w", err)
		}
	}

	if localPath == "" {
		return nil, fmt.Errorf("no local path or URL provided for resource")
	}

	// Upload image using uploadimg API to get permanent URL
	wechatImageURL, err := p.uploadImage(ctx, localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image to WeChat: %w", err)
	}

	p.logger.Info("Successfully uploaded image to WeChat",
		zap.String("resource_id", resource.ID),
		zap.String("wechat_image_url", wechatImageURL))

	// Validate the wechat image URL
	if wechatImageURL == "" {
		return nil, fmt.Errorf("empty URL returned from WeChat upload")
	}

	// Create processed resource
	processedResource := resource
	processedResource.LocalPath = localPath
	if processedResource.Metadata == nil {
		processedResource.Metadata = make(map[string]string)
	}
	
	// Store the WeChat image URL for use in article content
	processedResource.Metadata["wechat_image_url"] = wechatImageURL
	processedResource.Metadata["wechat_uploaded"] = "true"

	p.logger.Info("Image processed successfully for WeChat",
		zap.String("resource_id", resource.ID),
		zap.String("wechat_image_url", wechatImageURL))

	return &processedResource, nil
}

func (p *WeChatMediaProcessor) ProcessResources(ctx context.Context, resources []publisher.Resource, config publisher.PublishConfig) ([]publisher.Resource, error) {
	var processedResources []publisher.Resource

	for _, resource := range resources {
		processed, err := p.ProcessResource(ctx, resource, config)
		if err != nil {
			p.logger.Error("Failed to process WeChat resource",
				zap.String("resource_id", resource.ID),
				zap.Error(err))
			// Continue with original resource
			processedResources = append(processedResources, resource)
			continue
		}
		processedResources = append(processedResources, *processed)
	}

	return processedResources, nil
}

// uploadPermanentMaterial uploads image as permanent material (recommended for articles)
func (p *WeChatMediaProcessor) uploadPermanentMaterial(ctx context.Context, filePath, mediaType string) (string, string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/material/add_material?access_token=%s&type=%s", p.accessToken, mediaType)

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return "", "", fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close writer
	err = writer.Close()
	if err != nil {
		return "", "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	var materialResp WeChatMaterialAddResponse
	if err := json.Unmarshal(respBody, &materialResp); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	if materialResp.ErrCode != 0 {
		return "", "", fmt.Errorf("WeChat API error: %d - %s", materialResp.ErrCode, materialResp.ErrMsg)
	}

	return materialResp.MediaID, materialResp.URL, nil
}

// uploadTemporaryMedia uploads image as temporary media (3 days expiry)
func (p *WeChatMediaProcessor) uploadTemporaryMedia(ctx context.Context, filePath, mediaType string) (string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/media/upload?access_token=%s&type=%s", p.accessToken, mediaType)

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close writer
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var mediaResp WeChatMediaResponse
	if err := json.Unmarshal(respBody, &mediaResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if mediaResp.ErrCode != 0 {
		return "", fmt.Errorf("WeChat API error: %d - %s", mediaResp.ErrCode, mediaResp.ErrMsg)
	}

	return mediaResp.MediaID, nil
}

func (p *WeChatMediaProcessor) downloadImage(ctx context.Context, url string) (string, error) {
	// Create temp directory
	tempDir := "temp/wechat_images"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("wechat_%d%s", time.Now().Unix(), p.getFileExtension(url))
	localPath := filepath.Join(tempDir, filename)

	// Download image
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	// Create file
	file, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy content
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	return localPath, nil
}

// uploadThumbMaterial uploads image as thumb material for WeChat articles
func (p *WeChatMediaProcessor) uploadThumbMaterial(ctx context.Context, filePath string) (string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/material/add_material?access_token=%s&type=thumb", p.accessToken)

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close writer
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var thumbResp WeChatMaterialAddResponse
	if err := json.Unmarshal(respBody, &thumbResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if thumbResp.ErrCode != 0 {
		return "", fmt.Errorf("WeChat thumb API error: %d - %s", thumbResp.ErrCode, thumbResp.ErrMsg)
	}

	return thumbResp.MediaID, nil
}

// uploadImage uploads image using the uploadimg API to get permanent URL
func (p *WeChatMediaProcessor) uploadImage(ctx context.Context, filePath string) (string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/media/uploadimg?access_token=%s", p.accessToken)

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close writer
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var uploadResp WeChatUploadImageResponse
	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if uploadResp.ErrCode != 0 {
		return "", fmt.Errorf("WeChat uploadimg API error: %d - %s", uploadResp.ErrCode, uploadResp.ErrMsg)
	}

	return uploadResp.URL, nil
}

func (p *WeChatMediaProcessor) getFileExtension(url string) string {
	parts := strings.Split(url, ".")
	if len(parts) > 1 {
		ext := strings.ToLower(parts[len(parts)-1])
		// Remove query parameters
		if idx := strings.Index(ext, "?"); idx != -1 {
			ext = ext[:idx]
		}
		// Common image extensions
		validExts := []string{"jpg", "jpeg", "png", "gif"}
		for _, validExt := range validExts {
			if ext == validExt {
				return "." + ext
			}
		}
	}
	return ".jpg" // Default
}

// GetMediaInfo retrieves information about uploaded media
func (p *WeChatMediaProcessor) GetMediaInfo(ctx context.Context, mediaID string) (map[string]string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/material/get_material?access_token=%s", p.accessToken)

	reqBody := map[string]string{
		"media_id": mediaID,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check if response is JSON (error) or binary (success)
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		// Error response
		var errorResp struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("failed to decode error response: %w", err)
		}
		return nil, fmt.Errorf("WeChat API error: %d - %s", errorResp.ErrCode, errorResp.ErrMsg)
	}

	// Success - media exists
	return map[string]string{
		"media_id":     mediaID,
		"status":       "exists",
		"content_type": contentType,
	}, nil
}
