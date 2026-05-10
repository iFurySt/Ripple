package email

import "testing"

func TestParseRecipients(t *testing.T) {
	recipients := parseRecipients("a@example.com, b@example.com;c@example.com\n")
	if len(recipients) != 3 {
		t.Fatalf("expected 3 recipients, got %d", len(recipients))
	}
	if recipients[0] != "a@example.com" || recipients[1] != "b@example.com" || recipients[2] != "c@example.com" {
		t.Fatalf("unexpected recipients: %#v", recipients)
	}
}
