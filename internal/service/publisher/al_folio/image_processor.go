package al_folio

import (
	"context"
	"fmt"
	"github.com/ifuryst/ripple/internal/service/publisher"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// AlFolioImageProcessor handles image processing for Al-Folio blogs
type AlFolioImageProcessor struct {
	logger       *zap.Logger
	tempDir      string
	imageCounter int
}

// ImageLayout represents different image layout options
type ImageLayout int

const (
	SingleImage ImageLayout = iota
	TwoColumnRow
	ThreeColumnRow
	FourColumnRow
)

func NewAlFolioImageProcessor(logger *zap.Logger, tempDir string) *AlFolioImageProcessor {
	return &AlFolioImageProcessor{
		logger:       logger,
		tempDir:      tempDir,
		imageCounter: 0,
	}
}

func (p *AlFolioImageProcessor) ProcessContent(ctx context.Context, content string, metadata map[string]string, repoPath string) (string, []publisher.Resource, error) {
	var processedResources []publisher.Resource

	// Extract image directory name from metadata
	imageDir := metadata["image_dir"]
	if imageDir == "" {
		return content, processedResources, fmt.Errorf("image_dir not found in metadata")
	}

	// Create images directory in the repository
	assetsImagePath := filepath.Join(repoPath, "assets", "img", imageDir)
	if err := os.MkdirAll(assetsImagePath, 0755); err != nil {
		return content, processedResources, fmt.Errorf("failed to create assets image directory: %w", err)
	}

	// Find all images in the content
	imageURLs := p.extractImageURLs(content)
	p.logger.Info("Found images in content", zap.Int("count", len(imageURLs)))

	// Download and process each image
	imageMap := make(map[string]string) // original URL -> new path

	for _, url := range imageURLs {
		resource, err := p.downloadAndProcessImage(ctx, url, assetsImagePath, imageDir)
		if err != nil {
			p.logger.Error("Failed to process image", zap.String("url", url), zap.Error(err))
			continue
		}

		processedResources = append(processedResources, *resource)
		imageMap[url] = resource.URL // New Jekyll path

		// Also map the normalized URL (without query parameters) for better matching
		normalizedURL := p.normalizeImageURL(url)
		if normalizedURL != url {
			imageMap[normalizedURL] = resource.URL
		}
	}

	// Replace images in content with Jekyll format
	processedContent := p.replaceImagesInContent(content, imageMap, imageURLs)

	return processedContent, processedResources, nil
}

func (p *AlFolioImageProcessor) extractImageURLs(content string) []string {
	var urls []string

	// Match markdown images: ![alt](url)
	markdownImageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	matches := markdownImageRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			url := strings.TrimSpace(match[2])
			if p.isImageURL(url) {
				urls = append(urls, url)
			}
		}
	}

	// Match Jekyll liquid template images: {% include figure.liquid ... path="url" ... %}
	alFolioImageRegex := regexp.MustCompile(`{%\s*include\s+figure\.liquid[^%]*path="([^"]+)"[^%]*%}`)
	alFolioMatches := alFolioImageRegex.FindAllStringSubmatch(content, -1)

	for _, match := range alFolioMatches {
		if len(match) >= 2 {
			url := strings.TrimSpace(match[1])
			if p.isImageURL(url) {
				urls = append(urls, url)
			}
		}
	}

	// Also match HTML img tags if any
	htmlImageRegex := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`)
	htmlMatches := htmlImageRegex.FindAllStringSubmatch(content, -1)

	for _, match := range htmlMatches {
		if len(match) >= 2 {
			url := strings.TrimSpace(match[1])
			if p.isImageURL(url) {
				urls = append(urls, url)
			}
		}
	}

	return p.deduplicateURLs(urls)
}

func (p *AlFolioImageProcessor) isImageURL(url string) bool {
	// Check if URL points to an image
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg"}
	urlLower := strings.ToLower(url)

	for _, ext := range imageExtensions {
		if strings.Contains(urlLower, ext) {
			return true
		}
	}

	// Check if it's a Notion AWS S3 image URL
	if strings.Contains(url, "prod-files-secure.s3.us-west-2.amazonaws.com") {
		return true
	}

	// Also check if it's a Notion image URL
	if strings.Contains(url, "notion") && strings.Contains(url, "image") {
		return true
	}

	return false
}

func (p *AlFolioImageProcessor) downloadAndProcessImage(ctx context.Context, url, assetsPath, imageDir string) (*publisher.Resource, error) {
	// Generate unique filename using timestamp
	p.imageCounter++
	extension := p.getFileExtension(url)
	if extension == "" {
		extension = ".png" // Default for Notion images
	}

	// Use timestamp + counter for unique filenames
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%d_%d%s", timestamp, p.imageCounter, extension)
	localPath := filepath.Join(assetsPath, filename)

	// Download the image
	if err := p.downloadImage(ctx, url, localPath); err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	// Create Al-Folio-compatible path
	alFolioPath := fmt.Sprintf("/assets/img/%s/%s", imageDir, filename)

	resource := &publisher.Resource{
		ID:        fmt.Sprintf("img_%d", p.imageCounter),
		Type:      publisher.ResourceTypeImage,
		URL:       alFolioPath,
		LocalPath: localPath,
		Metadata: map[string]string{
			"original_url": url,
			"filename":     filename,
			"image_dir":    imageDir,
		},
	}

	p.logger.Info("Image processed",
		zap.String("original_url", url),
		zap.String("al_folio_path", alFolioPath),
		zap.String("local_path", localPath))

	return resource, nil
}

func (p *AlFolioImageProcessor) downloadImage(ctx context.Context, url, localPath string) error {
	// Check if file already exists
	if _, err := os.Stat(localPath); err == nil {
		p.logger.Debug("Image already exists locally", zap.String("path", localPath))
		return nil
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	// Create the file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy content
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	return nil
}

func (p *AlFolioImageProcessor) replaceImagesInContent(content string, imageMap map[string]string, imageURLs []string) string {
	// Group consecutive images for layout decisions
	imageGroups := p.groupConsecutiveImages(content, imageURLs)

	for _, group := range imageGroups {
		layout := p.determineLayout(len(group.URLs))
		alFolioHTML := p.generateAlFolioImageHTML(group.URLs, imageMap, layout)

		// Replace the group in content
		content = p.replaceImageGroup(content, group, alFolioHTML)
	}

	return content
}

type ImageGroup struct {
	URLs      []string
	StartPos  int
	EndPos    int
	FullMatch string
}

func (p *AlFolioImageProcessor) groupConsecutiveImages(content string, imageURLs []string) []ImageGroup {
	var groups []ImageGroup

	// Find positions of all images
	imagePositions := make(map[string][]int)

	for _, url := range imageURLs {
		// Find all occurrences of this image in markdown format
		markdownPattern := fmt.Sprintf(`!\[[^\]]*\]\(%s\)`, regexp.QuoteMeta(url))
		markdownRegex := regexp.MustCompile(markdownPattern)

		markdownMatches := markdownRegex.FindAllStringIndex(content, -1)
		for _, match := range markdownMatches {
			imagePositions[url] = append(imagePositions[url], match[0])
		}

		// Also find occurrences in Al-Folio liquid template format
		alFolioPattern := fmt.Sprintf(`{%%\s*include\s+figure\.liquid[^%%]*path="%s"[^%%]*%%}`, regexp.QuoteMeta(url))
		alFolioRegex := regexp.MustCompile(alFolioPattern)

		alFolioMatches := alFolioRegex.FindAllStringIndex(content, -1)
		for _, match := range alFolioMatches {
			imagePositions[url] = append(imagePositions[url], match[0])
		}
	}

	// Simple grouping: treat consecutive images (within 100 characters) as a group
	var currentGroup ImageGroup
	lastPos := -1

	// Sort URLs by their first occurrence position
	sortedURLs := p.sortURLsByPosition(imageURLs, imagePositions)

	for _, url := range sortedURLs {
		positions := imagePositions[url]
		if len(positions) == 0 {
			continue
		}

		pos := positions[0] // Use first occurrence

		if lastPos >= 0 && pos-lastPos > 200 { // More than 200 chars apart
			// Start new group
			if len(currentGroup.URLs) > 0 {
				groups = append(groups, currentGroup)
			}
			currentGroup = ImageGroup{URLs: []string{url}}
		} else {
			// Add to current group
			currentGroup.URLs = append(currentGroup.URLs, url)
		}

		lastPos = pos
	}

	// Add the last group
	if len(currentGroup.URLs) > 0 {
		groups = append(groups, currentGroup)
	}

	// If no groups formed, create individual groups
	if len(groups) == 0 {
		for _, url := range imageURLs {
			groups = append(groups, ImageGroup{URLs: []string{url}})
		}
	}

	return groups
}

func (p *AlFolioImageProcessor) sortURLsByPosition(urls []string, positions map[string][]int) []string {
	type urlPos struct {
		url string
		pos int
	}

	var urlPositions []urlPos
	for _, url := range urls {
		if pos, exists := positions[url]; exists && len(pos) > 0 {
			urlPositions = append(urlPositions, urlPos{url: url, pos: pos[0]})
		}
	}

	// Simple bubble sort by position
	for i := 0; i < len(urlPositions); i++ {
		for j := i + 1; j < len(urlPositions); j++ {
			if urlPositions[i].pos > urlPositions[j].pos {
				urlPositions[i], urlPositions[j] = urlPositions[j], urlPositions[i]
			}
		}
	}

	var sortedURLs []string
	for _, up := range urlPositions {
		sortedURLs = append(sortedURLs, up.url)
	}

	return sortedURLs
}

func (p *AlFolioImageProcessor) determineLayout(imageCount int) ImageLayout {
	switch imageCount {
	case 1:
		return SingleImage
	case 2:
		return TwoColumnRow
	case 3:
		return ThreeColumnRow
	case 4:
		return FourColumnRow
	default:
		// For more than 4 images, use multiple rows
		return TwoColumnRow
	}
}

func (p *AlFolioImageProcessor) generateAlFolioImageHTML(urls []string, imageMap map[string]string, layout ImageLayout) string {
	p.logger.Debug("Generating Jekyll HTML",
		zap.Strings("urls", urls),
		zap.Any("imageMap", imageMap),
		zap.Int("layout", int(layout)))

	switch layout {
	case SingleImage:
		if len(urls) >= 1 {
			alFolioPath := imageMap[urls[0]]
			if alFolioPath == "" {
				// Try normalized URL
				normalizedURL := p.normalizeImageURL(urls[0])
				alFolioPath = imageMap[normalizedURL]
			}
			if alFolioPath == "" {
				p.logger.Warn("No Al-Folio path found for URL", zap.String("url", urls[0]))
				alFolioPath = urls[0] // Fallback to original URL
			}
			return fmt.Sprintf(`<div class="row mt-3">
    <div class="col-sm mt-0 mb-0">
        {%% include figure.liquid loading="eager" path="%s" class="img-fluid rounded z-depth-1" zoomable=true %%}
    </div>
</div>`, alFolioPath)
		}

	case TwoColumnRow:
		if len(urls) >= 2 {
			path1 := imageMap[urls[0]]
			if path1 == "" {
				path1 = imageMap[p.normalizeImageURL(urls[0])]
			}
			if path1 == "" {
				path1 = urls[0]
			}

			path2 := imageMap[urls[1]]
			if path2 == "" {
				path2 = imageMap[p.normalizeImageURL(urls[1])]
			}
			if path2 == "" {
				path2 = urls[1]
			}
			return fmt.Sprintf(`<div class="row mt-3">
    <div class="col-sm mt-0 mb-0">
        {%% include figure.liquid loading="eager" path="%s" class="img-fluid rounded z-depth-1" zoomable=true %%}
    </div>
    <div class="col-sm mt-0 mb-0">
        {%% include figure.liquid loading="eager" path="%s" class="img-fluid rounded z-depth-1" zoomable=true %%}
    </div>
</div>`, path1, path2)
		}

	case ThreeColumnRow:
		if len(urls) >= 3 {
			path1 := imageMap[urls[0]]
			if path1 == "" {
				path1 = imageMap[p.normalizeImageURL(urls[0])]
			}
			if path1 == "" {
				path1 = urls[0]
			}

			path2 := imageMap[urls[1]]
			if path2 == "" {
				path2 = imageMap[p.normalizeImageURL(urls[1])]
			}
			if path2 == "" {
				path2 = urls[1]
			}

			path3 := imageMap[urls[2]]
			if path3 == "" {
				path3 = imageMap[p.normalizeImageURL(urls[2])]
			}
			if path3 == "" {
				path3 = urls[2]
			}
			return fmt.Sprintf(`<div class="row mt-3">
    <div class="col-sm mt-0 mb-0">
        {%% include figure.liquid loading="eager" path="%s" class="img-fluid rounded z-depth-1" zoomable=true %%}
    </div>
    <div class="col-sm mt-0 mb-0">
        {%% include figure.liquid loading="eager" path="%s" class="img-fluid rounded z-depth-1" zoomable=true %%}
    </div>
    <div class="col-sm mt-0 mb-0">
        {%% include figure.liquid loading="eager" path="%s" class="img-fluid rounded z-depth-1" zoomable=true %%}
    </div>
</div>`, path1, path2, path3)
		}
	}

	// Fallback: single image layout for first image
	if len(urls) > 0 {
		alFolioPath := imageMap[urls[0]]
		if alFolioPath == "" {
			// Try normalized URL
			normalizedURL := p.normalizeImageURL(urls[0])
			alFolioPath = imageMap[normalizedURL]
		}
		if alFolioPath == "" {
			p.logger.Warn("No Al-Folio path found for fallback URL", zap.String("url", urls[0]))
			alFolioPath = urls[0] // Fallback to original URL
		}
		return fmt.Sprintf(`<div class="row mt-3">
    <div class="col-sm mt-0 mb-0">
        {%% include figure.liquid loading="eager" path="%s" class="img-fluid rounded z-depth-1" zoomable=true %%}
    </div>
</div>`, alFolioPath)
	}

	return ""
}

func (p *AlFolioImageProcessor) replaceImageGroup(content string, group ImageGroup, alFolioHTML string) string {
	// Replace all images in the group with the Jekyll HTML
	for i, url := range group.URLs {
		// Get the normalized URL (without query parameters)
		normalizedURL := p.normalizeImageURL(url)

		if i == 0 {
			// Replace first image with the complete Al-Folio HTML (with updated local path)
			content = p.replaceImageURL(content, url, alFolioHTML)
			content = p.replaceImageURL(content, normalizedURL, alFolioHTML)
		} else {
			// Remove other images in the group (they're included in the first replacement)
			content = p.replaceImageURL(content, url, "")
			content = p.replaceImageURL(content, normalizedURL, "")
		}
	}

	return content
}

func (p *AlFolioImageProcessor) replaceImageURL(content, targetURL, replacement string) string {
	// Replace markdown format: ![alt](url)
	markdownPattern := fmt.Sprintf(`!\[[^\]]*\]\(%s\)`, regexp.QuoteMeta(targetURL))
	markdownRegex := regexp.MustCompile(markdownPattern)
	content = markdownRegex.ReplaceAllString(content, replacement)

	// Replace Jekyll liquid template format with flexible matching for AWS S3 URLs
	// This handles URLs with different credentials/timestamps
	normalizedURL := p.normalizeImageURL(targetURL)
	urlParts := strings.Split(normalizedURL, "/")

	if len(urlParts) >= 4 && strings.Contains(normalizedURL, "prod-files-secure.s3.us-west-2.amazonaws.com") {
		// Extract the unique file identifier (last part of the path)
		fileName := urlParts[len(urlParts)-1]
		// Also get the directory part (second-to-last part)
		dirName := urlParts[len(urlParts)-2]

		// Create a flexible pattern that matches any AWS S3 URL with the same file in the same directory
		// This pattern will match regardless of credentials/timestamps
		alFolioPattern := fmt.Sprintf(`{%%\s*include\s+figure\.liquid[^%%]*path="[^"]*prod-files-secure\.s3\.us-west-2\.amazonaws\.com[^"]*/%s/%s[^"]*"[^%%]*%%}`, regexp.QuoteMeta(dirName), regexp.QuoteMeta(fileName))
		alFolioRegex := regexp.MustCompile(alFolioPattern)
		content = alFolioRegex.ReplaceAllString(content, replacement)

		p.logger.Debug("Replacing Al-Folio image URL",
			zap.String("pattern", alFolioPattern),
			zap.String("replacement", replacement),
			zap.String("fileName", fileName))
	} else {
		// Fallback: exact URL matching for non-AWS S3 URLs
		alFolioPattern := fmt.Sprintf(`{%%\s*include\s+figure\.liquid[^%%]*path="%s"[^%%]*%%}`, regexp.QuoteMeta(targetURL))
		alFolioRegex := regexp.MustCompile(alFolioPattern)
		content = alFolioRegex.ReplaceAllString(content, replacement)
	}

	return content
}

func (p *AlFolioImageProcessor) getFileExtension(url string) string {
	// Extract extension from URL
	parts := strings.Split(url, ".")
	if len(parts) > 1 {
		ext := strings.ToLower(parts[len(parts)-1])
		// Remove query parameters
		if idx := strings.Index(ext, "?"); idx != -1 {
			ext = ext[:idx]
		}
		// Common image extensions
		validExts := []string{"jpg", "jpeg", "png", "gif", "webp", "svg"}
		for _, validExt := range validExts {
			if ext == validExt {
				return "." + ext
			}
		}
	}
	return ""
}

func (p *AlFolioImageProcessor) normalizeImageURL(url string) string {
	// Remove query parameters from URLs for better matching
	if idx := strings.Index(url, "?"); idx != -1 {
		return url[:idx]
	}
	return url
}

func (p *AlFolioImageProcessor) deduplicateURLs(urls []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, url := range urls {
		if !seen[url] {
			seen[url] = true
			result = append(result, url)
		}
	}

	return result
}
