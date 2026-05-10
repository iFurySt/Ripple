package notion

import (
	"testing"

	"github.com/ifuryst/ripple/internal/models"
)

func TestRoutePlatformsForHNDailyReport(t *testing.T) {
	platforms := routePlatformsForContentType(
		models.StringArray{"微信公众号"},
		models.StringArray{"HNDailyReport"},
	)

	if len(platforms) != 1 || platforms[0] != "Email" {
		t.Fatalf("expected HNDailyReport to route to Email, got %#v", platforms)
	}
}

func TestRoutePlatformsForOtherContent(t *testing.T) {
	platforms := routePlatformsForContentType(
		models.StringArray{"微信公众号"},
		models.StringArray{"Article"},
	)

	if len(platforms) != 1 || platforms[0] != "微信公众号" {
		t.Fatalf("expected platforms to remain unchanged, got %#v", platforms)
	}
}
