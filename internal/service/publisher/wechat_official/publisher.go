package wechat_official

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ifuryst/ripple/internal/service/publisher"

	"go.uber.org/zap"
)

// WeChatOfficialPublisher handles publishing to WeChat Official Account
type WeChatOfficialPublisher struct {
	logger             *zap.Logger
	contentTransformer *WeChatTransformer
	mediaProcessor     *WeChatMediaProcessor
	client             *http.Client
	accessToken        string
}

// WeChat API response structures
type WeChatAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type WeChatDraftAddRequest struct {
	Articles []WeChatArticle `json:"articles"`
}

type WeChatArticle struct {
	Title              string `json:"title"`
	Author             string `json:"author"`
	Digest             string `json:"digest"`
	Content            string `json:"content"`
	ContentSourceURL   string `json:"content_source_url"`
	ThumbMediaID       string `json:"thumb_media_id"`
	ShowCoverPic       int    `json:"show_cover_pic"`
	NeedOpenComment    int    `json:"need_open_comment"`
	OnlyFansCanComment int    `json:"only_fans_can_comment"`
}

type WeChatDraftResponse struct {
	MediaID string `json:"media_id"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type WeChatPublishRequest struct {
	MediaID string `json:"media_id"`
}

type WeChatPublishResponse struct {
	PublishID string `json:"publish_id"`
	MsgID     string `json:"msg_id"`
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
}

func NewWeChatOfficialPublisher(logger *zap.Logger) publisher.Publisher {
	wechatTransformer := NewWeChatTransformer()
	mediaProcessor := NewWeChatMediaProcessor(logger)

	return &WeChatOfficialPublisher{
		logger:             logger,
		contentTransformer: wechatTransformer,
		mediaProcessor:     mediaProcessor,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (p *WeChatOfficialPublisher) GetPlatformName() string {
	return "wechat-official"
}

func (p *WeChatOfficialPublisher) Initialize(ctx context.Context, config publisher.PublishConfig) error {
	if err := p.ValidateConfig(config); err != nil {
		return err
	}

	// Get access token
	accessToken, err := p.getAccessToken(config)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	p.accessToken = accessToken
	p.mediaProcessor.SetAccessToken(accessToken)

	p.logger.Info("WeChat Official Account publisher initialized successfully")
	return nil
}

func (p *WeChatOfficialPublisher) ValidateConfig(config publisher.PublishConfig) error {
	required := []string{"app_id", "app_secret"}

	for _, key := range required {
		if config.Config[key] == "" {
			return fmt.Errorf("missing required config: %s", key)
		}
	}

	return nil
}

func (p *WeChatOfficialPublisher) TransformContent(ctx context.Context, content publisher.PublishContent) (*publisher.PublishContent, error) {
	// Prepare metadata for transformation
	metadata := make(map[string]string)
	for k, v := range content.Metadata {
		metadata[k] = v
	}

	// Add content fields to metadata
	metadata["title"] = content.Title
	metadata["author"] = content.Author
	metadata["summary"] = content.Summary

	if content.PublishDate != nil {
		metadata["publish_date"] = content.PublishDate.Format(time.RFC3339)
	}

	if len(content.Tags) > 0 {
		metadata["tags"] = strings.Join(content.Tags, ", ")
	}

	// Transform content to WeChat HTML format
	transformedContent, err := p.contentTransformer.TransformContent(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to transform content: %w", err)
	}
	transformedHTMLContent := transformedContent.Content

	// Extract images from content for processing
	imageURLs := p.contentTransformer.ExtractImages(transformedHTMLContent)

	// Create resources for images
	var resources []publisher.Resource
	for i, url := range imageURLs {
		resources = append(resources, publisher.Resource{
			ID:   fmt.Sprintf("wechat_img_%d", i+1),
			Type: publisher.ResourceTypeImage,
			URL:  url,
		})
	}

	// Create new content with transformed data
	result := content
	result.Content = transformedHTMLContent
	result.Resources = resources

	return &result, nil
}

func (p *WeChatOfficialPublisher) ProcessResources(ctx context.Context, content *publisher.PublishContent, config publisher.PublishConfig) error {
	if len(content.Resources) == 0 {
		return nil
	}

	// Process images and upload to WeChat
	processedResources, err := p.mediaProcessor.ProcessResources(ctx, content.Resources, config)
	if err != nil {
		return fmt.Errorf("failed to process resources: %w", err)
	}

	// Update content with processed resources
	content.Resources = processedResources

	// Update content to use WeChat media references
	content.Content = p.contentTransformer.UpdateImageReferences(content.Content, processedResources)

	p.logger.Info("Processed WeChat resources",
		zap.Int("image_count", len(processedResources)))

	return nil
}

func (p *WeChatOfficialPublisher) SaveToDraft(ctx context.Context, content publisher.PublishContent, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	// Validate content before creating draft
	if content.Title == "" {
		titleErr := fmt.Errorf("article title is required")
		return &publisher.PublishResult{
			Success:  false,
			Error:    titleErr,
			ErrorMsg: titleErr.Error(),
		}, nil
	}

	if content.Content == "" {
		contentErr := fmt.Errorf("article content is required")
		return &publisher.PublishResult{
			Success:  false,
			Error:    contentErr,
			ErrorMsg: contentErr.Error(),
		}, nil
	}

	// Create article for WeChat draft
	article := WeChatArticle{
		Title:              content.Title,
		Author:             content.Author,
		Digest:             "", // 暂时留空，避免长度超限问题
		Content:            content.Content,
		ContentSourceURL:   config.Config["source_url"],
		ShowCoverPic:       1,
		NeedOpenComment:    p.getIntConfig(config.Config["need_open_comment"], 0),
		OnlyFansCanComment: p.getIntConfig(config.Config["only_fans_can_comment"], 0),
	}

	// Use default thumb media ID from config
	defaultThumbMediaID := config.Config["default_thumb_media_id"]
	p.logger.Info("Checking default thumb media_id from config",
		zap.String("default_thumb_media_id", defaultThumbMediaID),
		zap.Any("all_config", config.Config))

	if defaultThumbMediaID != "" {
		article.ThumbMediaID = defaultThumbMediaID
		p.logger.Info("Using default thumb media_id for article thumbnail",
			zap.String("media_id", defaultThumbMediaID))
	} else {
		p.logger.Warn("No default thumb media_id configured, creating draft without thumbnail")
	}

	// Create draft request
	draftRequest := WeChatDraftAddRequest{
		Articles: []WeChatArticle{article},
	}

	// Call WeChat API to add draft
	mediaID, err := p.addDraft(draftRequest, config)
	if err != nil {
		draftErr := fmt.Errorf("failed to create WeChat draft: %w", err)
		return &publisher.PublishResult{
			Success:  false,
			Error:    draftErr,
			ErrorMsg: draftErr.Error(),
		}, nil
	}

	p.logger.Info("Draft saved successfully",
		zap.String("media_id", mediaID),
		zap.String("title", content.Title))

	return &publisher.PublishResult{
		Success:   true,
		PublishID: mediaID,
		Metadata: map[string]string{
			"media_id":     mediaID,
			"platform":     "wechat-official",
			"draft_status": "saved",
		},
	}, nil
}

func (p *WeChatOfficialPublisher) Publish(ctx context.Context, draftID string, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	// Publish the draft using media_id
	publishRequest := WeChatPublishRequest{
		MediaID: draftID,
	}

	publishResponse, err := p.publishDraft(publishRequest, config)
	if err != nil {
		return &publisher.PublishResult{
			Success:  false,
			Error:    err,
			ErrorMsg: err.Error(),
		}, nil
	}

	p.logger.Info("Content published successfully",
		zap.String("publish_id", publishResponse.PublishID),
		zap.String("msg_id", publishResponse.MsgID))

	return &publisher.PublishResult{
		Success:     true,
		PublishID:   publishResponse.PublishID,
		PublishedAt: time.Now(),
		Metadata: map[string]string{
			"publish_id": publishResponse.PublishID,
			"msg_id":     publishResponse.MsgID,
			"media_id":   draftID,
		},
	}, nil
}

func (p *WeChatOfficialPublisher) PublishDirect(ctx context.Context, content publisher.PublishContent, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	// Stage 1: Initialize - validate access token
	if p.accessToken == "" {
		tokenErr := fmt.Errorf("WeChat publisher not initialized - access token missing")
		return &publisher.PublishResult{
			Success:  false,
			Error:    tokenErr,
			ErrorMsg: tokenErr.Error(),
		}, nil
	}

	// Stage 2: Transform content first (before processing media)
	transformedContent, err := p.TransformContent(ctx, content)
	if err != nil {
		transformErr := fmt.Errorf("content transformation failed: %w", err)
		return &publisher.PublishResult{
			Success:  false,
			Error:    transformErr,
			ErrorMsg: transformErr.Error(),
		}, nil
	}

	// Stage 3: Process media resources (upload images to WeChat)
	if err := p.ProcessResources(ctx, transformedContent, config); err != nil {
		mediaErr := fmt.Errorf("media processing failed: %w", err)
		return &publisher.PublishResult{
			Success:  false,
			Error:    mediaErr,
			ErrorMsg: mediaErr.Error(),
		}, nil
	}

	// Stage 4: Save to draft
	draftResult, err := p.SaveToDraft(ctx, *transformedContent, config)
	if err != nil {
		draftCreationErr := fmt.Errorf("draft creation failed: %w", err)
		return &publisher.PublishResult{
			Success:  false,
			Error:    draftCreationErr,
			ErrorMsg: draftCreationErr.Error(),
		}, nil
	}

	// Stage 5: Auto-publish if enabled
	if autoPublish := config.Config["auto_publish"]; autoPublish == "true" {
		publishResult, err := p.Publish(ctx, draftResult.PublishID, config)
		if err != nil {
			// Even if publish fails, draft was successful
			draftResult.Metadata["publish_error"] = err.Error()
			p.logger.Warn("Auto-publish failed but draft created successfully",
				zap.String("draft_id", draftResult.PublishID),
				zap.Error(err))
			return draftResult, nil
		}
		return publishResult, nil
	}

	// Stage 6: Cleanup will be called separately if needed
	return draftResult, nil
}

func (p *WeChatOfficialPublisher) GetPublishStatus(ctx context.Context, publishID string, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	// Check draft status by trying to get material info
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/draft/get?access_token=%s", p.accessToken)

	reqBody := map[string]interface{}{
		"media_id": publishID,
		"index":    0,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := p.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to check status: %w", err)
	}
	defer resp.Body.Close()

	var statusResp struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	success := statusResp.ErrCode == 0
	statusErr := fmt.Errorf("WeChat API error: %s", statusResp.ErrMsg)
	return &publisher.PublishResult{
		Success:   success,
		PublishID: publishID,
		Error:     statusErr,
		ErrorMsg:  statusErr.Error(),
	}, nil
}

func (p *WeChatOfficialPublisher) Cleanup(ctx context.Context, publishID string, config publisher.PublishConfig) error {
	// Clean up temporary files if any
	p.logger.Info("WeChat cleanup completed", zap.String("publish_id", publishID))
	return nil
}

// Helper methods

func (p *WeChatOfficialPublisher) getAccessToken(config publisher.PublishConfig) (string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		config.Config["app_id"], config.Config["app_secret"])

	resp, err := p.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResponse WeChatAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", err
	}

	if tokenResponse.ErrCode != 0 {
		return "", fmt.Errorf("WeChat API error: %s", tokenResponse.ErrMsg)
	}

	return tokenResponse.AccessToken, nil
}

func (p *WeChatOfficialPublisher) addDraft(draftRequest WeChatDraftAddRequest, config publisher.PublishConfig) (string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/draft/add?access_token=%s", p.accessToken)

	jsonData, err := json.Marshal(draftRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal draft request: %w", err)
	}

	// Log the request details for debugging
	p.logger.Info("Sending draft request to WeChat API",
		zap.String("url", url),
		zap.String("request_json", string(jsonData)))

	resp, err := p.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send draft request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read draft response: %w", err)
	}

	p.logger.Info("Received draft response from WeChat API",
		zap.String("response_body", string(body)))

	var draftResponse WeChatDraftResponse
	if err := json.Unmarshal(body, &draftResponse); err != nil {
		return "", fmt.Errorf("failed to parse draft response: %w", err)
	}

	if draftResponse.ErrCode != 0 {
		p.logger.Error("WeChat draft API returned error",
			zap.Int("error_code", draftResponse.ErrCode),
			zap.String("error_message", draftResponse.ErrMsg))
		return "", fmt.Errorf("WeChat draft API error: %s", draftResponse.ErrMsg)
	}

	return draftResponse.MediaID, nil
}

func (p *WeChatOfficialPublisher) publishDraft(publishRequest WeChatPublishRequest, config publisher.PublishConfig) (*WeChatPublishResponse, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/freepublish/submit?access_token=%s", p.accessToken)

	jsonData, err := json.Marshal(publishRequest)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var publishResponse WeChatPublishResponse
	if err := json.Unmarshal(body, &publishResponse); err != nil {
		return nil, err
	}

	if publishResponse.ErrCode != 0 {
		return nil, fmt.Errorf("WeChat publish API error: %s", publishResponse.ErrMsg)
	}

	return &publishResponse, nil
}

func (p *WeChatOfficialPublisher) getIntConfig(value string, defaultValue int) int {
	if value == "true" || value == "1" {
		return 1
	}
	if value == "false" || value == "0" {
		return 0
	}
	return defaultValue
}
