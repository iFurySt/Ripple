package email

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/resend/resend-go/v3"
	"go.uber.org/zap"

	"github.com/ifuryst/ripple/internal/service/publisher"
)

type Publisher struct {
	logger *zap.Logger
	client *resend.Client
}

func NewPublisher(logger *zap.Logger) publisher.Publisher {
	return &Publisher{logger: logger}
}

func (p *Publisher) GetPlatformName() string {
	return "email"
}

func (p *Publisher) Initialize(ctx context.Context, config publisher.PublishConfig) error {
	if err := p.ValidateConfig(config); err != nil {
		return err
	}

	p.client = resend.NewClient(config.Config["resend_api_key"])
	p.logger.Info("Email publisher initialized", zap.String("provider", config.Config["provider"]))
	return nil
}

func (p *Publisher) ValidateConfig(config publisher.PublishConfig) error {
	if !config.Enabled {
		return fmt.Errorf("email publisher is disabled")
	}

	if strings.ToLower(config.Config["provider"]) != "resend" {
		return fmt.Errorf("unsupported email provider: %s", config.Config["provider"])
	}
	if strings.TrimSpace(config.Config["resend_api_key"]) == "" {
		return fmt.Errorf("resend API key is required")
	}
	if strings.TrimSpace(config.Config["from"]) == "" {
		return fmt.Errorf("email from address is required")
	}
	if len(parseRecipients(config.Config["to"])) == 0 {
		return fmt.Errorf("at least one email recipient is required")
	}

	return nil
}

func (p *Publisher) TransformContent(ctx context.Context, content publisher.PublishContent) (*publisher.PublishContent, error) {
	html, err := convertNotionBlocksToEmailHTML(content.Content)
	if err != nil {
		return nil, fmt.Errorf("notion blocks to email HTML conversion failed: %w", err)
	}

	result := content
	result.Content = html
	return &result, nil
}

func (p *Publisher) ProcessResources(ctx context.Context, content *publisher.PublishContent, config publisher.PublishConfig) error {
	return nil
}

func (p *Publisher) SaveToDraft(ctx context.Context, content publisher.PublishContent, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	return p.PublishDirect(ctx, content, config)
}

func (p *Publisher) Publish(ctx context.Context, draftID string, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	return nil, fmt.Errorf("email publisher does not support publishing existing drafts")
}

func (p *Publisher) PublishDirect(ctx context.Context, content publisher.PublishContent, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	if p.client == nil {
		if err := p.Initialize(ctx, config); err != nil {
			return nil, err
		}
	}

	params := &resend.SendEmailRequest{
		From:    config.Config["from"],
		To:      parseRecipients(config.Config["to"]),
		Subject: content.Title,
		Html:    wrapEmailHTML(content.Title, content.Content),
	}

	sent, err := p.client.Emails.Send(params)
	if err != nil {
		return &publisher.PublishResult{
			Success:  false,
			Error:    err,
			ErrorMsg: err.Error(),
		}, err
	}

	now := time.Now()
	return &publisher.PublishResult{
		Success:     true,
		PublishID:   sent.Id,
		PublishedAt: now,
		Metadata: map[string]string{
			"provider": "resend",
			"to":       strings.Join(params.To, ","),
		},
	}, nil
}

func (p *Publisher) GetPublishStatus(ctx context.Context, publishID string, config publisher.PublishConfig) (*publisher.PublishResult, error) {
	return &publisher.PublishResult{
		Success:   true,
		PublishID: publishID,
		Metadata: map[string]string{
			"status": "sent",
		},
	}, nil
}

func (p *Publisher) Cleanup(ctx context.Context, publishID string, config publisher.PublishConfig) error {
	return nil
}

func parseRecipients(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})

	recipients := make([]string, 0, len(fields))
	for _, field := range fields {
		recipient := strings.TrimSpace(field)
		if recipient != "" {
			recipients = append(recipients, recipient)
		}
	}
	return recipients
}
