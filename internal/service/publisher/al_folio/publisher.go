package al_folio

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ifuryst/ripple/pkg/util"

	"github.com/ifuryst/ripple/internal/service/publisher"
	"github.com/ifuryst/ripple/pkg/git"

	"go.uber.org/zap"
)

// AlFolioPublisher handles publishing to Al-Folio blogs
type AlFolioPublisher struct {
	logger             *zap.Logger
	contentTransformer *AlFolioTransformer
	imageProcessor     *AlFolioImageProcessor
	repository         *git.Repository
}

func NewAlFolioPublisher(logger *zap.Logger) publisher.Publisher {
	alFolioTransformer := NewAlFolioTransformer()

	return &AlFolioPublisher{
		logger:             logger,
		contentTransformer: alFolioTransformer,
		imageProcessor:     NewAlFolioImageProcessor(logger, "temp/images"),
	}
}

func (p *AlFolioPublisher) GetPlatformName() string {
	return "al-folio"
}

func (p *AlFolioPublisher) Initialize(ctx context.Context, config publisher.PublishConfig) error {
	// Validate required configuration
	if err := p.ValidateConfig(config); err != nil {
		return err
	}

	// Initialize git repository
	repoConfig := git.RepositoryConfig{
		URL:          config.Config["repo_url"],
		Branch:       config.Config["branch"],
		WorkspaceDir: config.Config["workspace_dir"],
		GitUsername:  config.Config["git_username"],
		GitEmail:     config.Config["git_email"],
	}

	p.repository = git.NewRepository(repoConfig, p.logger)

	// Initialize (clone or pull) the repository
	if err := p.repository.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	p.logger.Info("Al-Folio blog publisher initialized",
		zap.String("repo_url", config.Config["repo_url"]),
		zap.String("branch", config.Config["branch"]))

	return nil
}

func (p *AlFolioPublisher) ValidateConfig(config publisher.PublishConfig) error {
	required := []string{"repo_url", "branch", "workspace_dir"}

	for _, key := range required {
		if config.Config[key] == "" {
			return fmt.Errorf("missing required config: %s", key)
		}
	}

	return nil
}

func (p *AlFolioPublisher) TransformContent(ctx context.Context, content publisher.PublishContent) (*publisher.PublishContent, error) {
	// Generate filename and image directory
	publishDate := time.Now()
	if content.PublishDate != nil {
		publishDate = *content.PublishDate
	}

	// Use metadata-aware filename generation
	filename := util.GenerateFilenameWithMetadata(content.Title, publishDate, content.Metadata)
	imageDir := util.GenerateImageDirWithMetadata(content.Title, publishDate, content.Metadata)

	// Prepare metadata for Jekyll transformation
	metadata := make(map[string]string)
	for k, v := range content.Metadata {
		metadata[k] = v
	}

	// Add content fields to metadata
	metadata["title"] = content.Title
	metadata["author"] = content.Author
	metadata["summary"] = content.Summary
	metadata["filename"] = filename
	metadata["image_dir"] = imageDir
	metadata["content"] = content.Content // For TOC detection

	if content.PublishDate != nil {
		metadata["publish_date"] = content.PublishDate.Format(time.RFC3339)
	}

	if len(content.Tags) > 0 {
		metadata["tags"] = strings.Join(content.Tags, ", ")
	}

	// Parse categories from content metadata or use tags as fallback
	if categories := content.Metadata["categories"]; categories != "" {
		metadata["categories"] = categories
	} else if len(content.Tags) > 0 {
		// Use first tag as category
		metadata["categories"] = content.Tags[0]
	}

	// Transform content to Al-Folio format
	transformedContent, err := p.contentTransformer.Transform(ctx, content.Content, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to transform content: %w", err)
	}

	// Create new content with transformed data
	result := content
	result.Content = transformedContent
	result.Metadata["filename"] = filename
	result.Metadata["image_dir"] = imageDir

	return &result, nil
}

func (p *AlFolioPublisher) ProcessResources(ctx context.Context, content *publisher.PublishContent, config publisher.PublishConfig) error {
	// Get repository path
	repoPath := p.repository.GetLocalPath()

	// Process images and update content
	processedContent, resources, err := p.imageProcessor.ProcessContent(
		ctx,
		content.Content,
		content.Metadata,
		repoPath,
	)
	if err != nil {
		return fmt.Errorf("failed to process images: %w", err)
	}

	// Update content with processed images
	content.Content = processedContent
	content.Resources = resources

	p.logger.Info("Processed resources",
		zap.Int("image_count", len(resources)),
		zap.String("image_dir", content.Metadata["image_dir"]))

	return nil
}

func (p *AlFolioPublisher) SaveToDraft(ctx context.Context, content publisher.PublishContent, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	// Transform content first
	transformedContent, err := p.TransformContent(ctx, content)
	if err != nil {
		return &publisher.PublishResult{
			Success:  false,
			Error:    err,
			ErrorMsg: err.Error(),
		}, nil
	}

	// Process resources (images)
	if err := p.ProcessResources(ctx, transformedContent, config); err != nil {
		return &publisher.PublishResult{
			Success:  false,
			Error:    err,
			ErrorMsg: err.Error(),
		}, nil
	}

	// For Al-Folio, draft means creating the file with "draft_" prefix
	filename := transformedContent.Metadata["filename"]
	if filename == "" {
		return &publisher.PublishResult{
			Success: false,
			Error:   fmt.Errorf("filename not found in metadata"),
		}, nil
	}

	draftFilename := "draft_" + filename
	return p.writePostFile(ctx, *transformedContent, draftFilename, true)
}

func (p *AlFolioPublisher) Publish(ctx context.Context, draftID string, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	// For Al-Folio, publishing means committing and pushing to git
	repoPath := p.repository.GetLocalPath()

	// Check if there are changes to commit
	hasChanges, err := p.repository.HasChanges()
	if err != nil {
		return &publisher.PublishResult{
			Success: false,
			Error:   fmt.Errorf("failed to check git status: %w", err),
		}, nil
	}

	if !hasChanges {
		p.logger.Info("No changes to commit")
		return &publisher.PublishResult{
			Success:     true,
			PublishID:   draftID,
			PublishedAt: time.Now(),
		}, nil
	}

	// Stage all changes
	if err := p.repository.Add(); err != nil {
		return &publisher.PublishResult{
			Success: false,
			Error:   fmt.Errorf("failed to stage changes: %w", err),
		}, nil
	}

	// Commit changes
	commitMessage := fmt.Sprintf("Add new post: %s", draftID)
	if customMessage := config.Config["commit_message"]; customMessage != "" {
		commitMessage = customMessage
	}

	if err := p.repository.Commit(commitMessage); err != nil {
		return &publisher.PublishResult{
			Success: false,
			Error:   fmt.Errorf("failed to commit changes: %w", err),
		}, nil
	}

	// Push to remote only if auto_publish is enabled
	autoPublish := true // default to true for backward compatibility
	if autoPublishStr := config.Config["auto_publish"]; autoPublishStr != "" {
		autoPublish = autoPublishStr == "true"
	}

	if autoPublish {
		if err := p.repository.Push(); err != nil {
			return &publisher.PublishResult{
				Success: false,
				Error:   fmt.Errorf("failed to push changes: %w", err),
			}, nil
		}
	}

	// Generate URL if base_url is provided
	var url string
	if baseURL := config.Config["base_url"]; baseURL != "" {
		slug := p.generateSlugFromFilename(draftID)
		// Al-Folio URL format: /blog/YYYY/title/
		publishDate := time.Now()
		url = fmt.Sprintf("%s/blog/%d/%s/", baseURL, publishDate.Year(), slug)
	}

	// Get commit hash
	commitHash, _ := p.repository.GetLastCommitHash()

	logMsg := "Successfully committed to Al-Folio blog"
	if autoPublish {
		logMsg = "Successfully published to Al-Folio blog"
	}

	p.logger.Info(logMsg,
		zap.String("draft_id", draftID),
		zap.String("url", url),
		zap.String("commit_hash", commitHash),
		zap.Bool("auto_publish", autoPublish))

	return &publisher.PublishResult{
		Success:     true,
		PublishID:   draftID,
		URL:         url,
		PublishedAt: time.Now(),
		Metadata: map[string]string{
			"commit_hash": commitHash,
			"branch":      p.repository.GetBranch(),
			"repo_path":   repoPath,
		},
	}, nil
}

func (p *AlFolioPublisher) PublishDirect(ctx context.Context, content publisher.PublishContent, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	// Transform content
	transformedContent, err := p.TransformContent(ctx, content)
	if err != nil {
		return &publisher.PublishResult{
			Success:  false,
			Error:    err,
			ErrorMsg: err.Error(),
		}, nil
	}

	// Process resources (images)
	if err := p.ProcessResources(ctx, transformedContent, config); err != nil {
		return &publisher.PublishResult{
			Success:  false,
			Error:    err,
			ErrorMsg: err.Error(),
		}, nil
	}

	// Write post file
	filename := transformedContent.Metadata["filename"]
	writeResult, err := p.writePostFile(ctx, *transformedContent, filename, false)
	if err != nil {
		return &publisher.PublishResult{
			Success:  false,
			Error:    err,
			ErrorMsg: err.Error(),
		}, nil
	}

	// Publish (commit and push)
	publishResult, err := p.Publish(ctx, writeResult.PublishID, config)
	if err != nil {
		return &publisher.PublishResult{
			Success:  false,
			Error:    err,
			ErrorMsg: err.Error(),
		}, nil
	}

	return publishResult, nil
}

func (p *AlFolioPublisher) GetPublishStatus(ctx context.Context, publishID string, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	// Check if the file exists in the repository
	if !p.repository.FileExists(filepath.Join("_posts", publishID)) {
		err := fmt.Errorf("post file not found: %s", publishID)
		return &publisher.PublishResult{
			Success:   false,
			PublishID: publishID,
			Error:     err,
			ErrorMsg:  err.Error(),
		}, nil
	}

	return &publisher.PublishResult{
		Success:   true,
		PublishID: publishID,
	}, nil
}

func (p *AlFolioPublisher) Cleanup(ctx context.Context, publishID string, config publisher.PublishConfig) error {
	// For Al-Folio, cleanup might involve removing temporary files
	p.logger.Info("Al-Folio blog cleanup completed", zap.String("publish_id", publishID))
	return nil
}

// Helper methods

func (p *AlFolioPublisher) writePostFile(ctx context.Context, content publisher.PublishContent, filename string, isDraft bool) (*publisher.PublishResult, error) {
	// Write to _posts directory
	postsDir := "_posts"
	relativePath := filepath.Join(postsDir, filename)

	// Create the file in the repository
	if err := p.repository.CreateFile(relativePath, []byte(content.Content)); err != nil {
		return &publisher.PublishResult{
			Success: false,
			Error:   fmt.Errorf("failed to create post file: %w", err),
		}, nil
	}

	// Run prettier to format the markdown file
	if err := p.runPrettier(ctx); err != nil {
		p.logger.Warn("Failed to run prettier, continuing without formatting",
			zap.Error(err))
	}

	p.logger.Info("Post file created",
		zap.String("filename", filename),
		zap.String("path", relativePath),
		zap.Bool("is_draft", isDraft))

	return &publisher.PublishResult{
		Success:   true,
		PublishID: filename,
		Metadata: map[string]string{
			"file_path": relativePath,
			"filename":  filename,
		},
	}, nil
}

func (p *AlFolioPublisher) generateSlugFromFilename(filename string) string {
	// Extract slug from Al-Folio filename: YYYY-MM-DD-slug.md -> slug
	parts := strings.Split(filename, "-")
	if len(parts) >= 4 {
		// Join parts after date (first 3 parts)
		slug := strings.Join(parts[3:], "-")
		// Remove .md extension
		if strings.HasSuffix(slug, ".md") {
			slug = strings.TrimSuffix(slug, ".md")
		}
		return slug
	}
	return filename
}

func (p *AlFolioPublisher) runPrettier(ctx context.Context) error {
	// Get the repository path
	repoPath := p.repository.GetLocalPath()

	// First, run npm ci to ensure dependencies are installed
	p.logger.Info("Installing dependencies with npm ci...")
	npmCmd := exec.CommandContext(ctx, "npm", "ci")
	npmCmd.Dir = repoPath

	npmOutput, err := npmCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm ci command failed: %w, output: %s", err, string(npmOutput))
	}

	p.logger.Info("Dependencies installed successfully",
		zap.String("output", string(npmOutput)))

	// Then run prettier to format the markdown file
	p.logger.Info("Running prettier to format files...")
	cmd := exec.CommandContext(ctx, "npx", "prettier", "--write", ".")
	cmd.Dir = repoPath

	// Run the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("prettier command failed: %w, output: %s", err, string(output))
	}

	p.logger.Info("Prettier formatting completed",
		zap.String("output", string(output)))

	return nil
}
