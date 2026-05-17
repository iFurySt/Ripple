package publisher

import (
	"context"
	"errors"
	"testing"

	"github.com/ifuryst/ripple/internal/models"
	"go.uber.org/zap"
)

func TestPublishSinglePlatformPassesRawContentToPublishDirect(t *testing.T) {
	sentinelErr := errors.New("stop before db write")
	fake := &singlePlatformFakePublisher{
		name:      "fake",
		directErr: sentinelErr,
	}
	manager := NewPublishManager(zap.NewNop(), nil)

	if err := manager.RegisterPublisher(fake); err != nil {
		t.Fatalf("register publisher: %v", err)
	}
	manager.SetPlatformConfig("fake", PublishConfig{
		PlatformName: "fake",
		Enabled:      true,
		Config:       map[string]string{},
	})

	page := &models.NotionPage{
		ID:       1,
		NotionID: "page-1",
		Title:    "Title",
		Content:  `[{"type":"paragraph","paragraph":{"rich_text":[]}}]`,
		Status:   "Done",
	}

	result, err := manager.PublishSinglePlatform(context.Background(), page, "fake", false)
	if err != nil {
		t.Fatalf("PublishSinglePlatform returned unexpected error: %v", err)
	}
	if result == nil || result.Error == nil || result.Error.Error() != sentinelErr.Error() {
		t.Fatalf("expected sentinel result error, got %#v", result)
	}
	if fake.transformCalls != 0 {
		t.Fatalf("TransformContent should not be called by manager, got %d calls", fake.transformCalls)
	}
	if fake.directContent != page.Content {
		t.Fatalf("PublishDirect content = %q, want raw content %q", fake.directContent, page.Content)
	}
}

func TestPublishSinglePlatformPassesRawContentToSaveToDraft(t *testing.T) {
	sentinelErr := errors.New("stop before db write")
	fake := &singlePlatformFakePublisher{
		name:     "fake",
		draftErr: sentinelErr,
	}
	manager := NewPublishManager(zap.NewNop(), nil)

	if err := manager.RegisterPublisher(fake); err != nil {
		t.Fatalf("register publisher: %v", err)
	}
	manager.SetPlatformConfig("fake", PublishConfig{
		PlatformName: "fake",
		Enabled:      true,
		Config:       map[string]string{},
	})

	page := &models.NotionPage{
		ID:       1,
		NotionID: "page-1",
		Title:    "Title",
		Content:  `[{"type":"paragraph","paragraph":{"rich_text":[]}}]`,
		Status:   "Done",
	}

	result, err := manager.PublishSinglePlatform(context.Background(), page, "fake", true)
	if err != nil {
		t.Fatalf("PublishSinglePlatform returned unexpected error: %v", err)
	}
	if result == nil || result.Error == nil || result.Error.Error() != sentinelErr.Error() {
		t.Fatalf("expected sentinel result error, got %#v", result)
	}
	if fake.transformCalls != 0 {
		t.Fatalf("TransformContent should not be called by manager, got %d calls", fake.transformCalls)
	}
	if fake.draftContent != page.Content {
		t.Fatalf("SaveToDraft content = %q, want raw content %q", fake.draftContent, page.Content)
	}
}

type singlePlatformFakePublisher struct {
	name           string
	transformCalls int
	directContent  string
	draftContent   string
	directErr      error
	draftErr       error
}

func (p *singlePlatformFakePublisher) GetPlatformName() string {
	return p.name
}

func (p *singlePlatformFakePublisher) Initialize(ctx context.Context, config PublishConfig) error {
	return nil
}

func (p *singlePlatformFakePublisher) ValidateConfig(config PublishConfig) error {
	return nil
}

func (p *singlePlatformFakePublisher) TransformContent(ctx context.Context, content PublishContent) (*PublishContent, error) {
	p.transformCalls++
	result := content
	result.Content = "transformed:" + content.Content
	return &result, nil
}

func (p *singlePlatformFakePublisher) ProcessResources(ctx context.Context, content *PublishContent, config PublishConfig) error {
	return nil
}

func (p *singlePlatformFakePublisher) SaveToDraft(ctx context.Context, content PublishContent, config PublishConfig) (*PublishResult, error) {
	p.draftContent = content.Content
	return nil, p.draftErr
}

func (p *singlePlatformFakePublisher) Publish(ctx context.Context, draftID string, config PublishConfig) (*PublishResult, error) {
	return &PublishResult{Success: true, PublishID: draftID}, nil
}

func (p *singlePlatformFakePublisher) PublishDirect(ctx context.Context, content PublishContent, config PublishConfig) (*PublishResult, error) {
	p.directContent = content.Content
	return nil, p.directErr
}

func (p *singlePlatformFakePublisher) GetPublishStatus(ctx context.Context, publishID string, config PublishConfig) (*PublishResult, error) {
	return &PublishResult{Success: true, PublishID: publishID}, nil
}

func (p *singlePlatformFakePublisher) Cleanup(ctx context.Context, publishID string, config PublishConfig) error {
	return nil
}
