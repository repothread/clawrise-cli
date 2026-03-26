package runtime

import "testing"

func TestParseOperationWithFullPath(t *testing.T) {
	t.Parallel()

	operation, err := ParseOperation("feishu.calendar.event.create", "")
	if err != nil {
		t.Fatalf("ParseOperation returned error: %v", err)
	}

	if operation.Platform != "feishu" {
		t.Fatalf("unexpected platform: %s", operation.Platform)
	}
	if operation.ResourcePath != "calendar.event" {
		t.Fatalf("unexpected resource path: %s", operation.ResourcePath)
	}
	if operation.Action != "create" {
		t.Fatalf("unexpected action: %s", operation.Action)
	}
	if operation.Normalized != "feishu.calendar.event.create" {
		t.Fatalf("unexpected normalized operation: %s", operation.Normalized)
	}
}

func TestParseOperationWithDefaultPlatform(t *testing.T) {
	t.Parallel()

	operation, err := ParseOperation("calendar.event.create", "feishu")
	if err != nil {
		t.Fatalf("ParseOperation returned error: %v", err)
	}

	if operation.Platform != "feishu" {
		t.Fatalf("unexpected platform: %s", operation.Platform)
	}
	if operation.Normalized != "feishu.calendar.event.create" {
		t.Fatalf("unexpected normalized operation: %s", operation.Normalized)
	}
}

func TestParseOperationRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	if _, err := ParseOperation("calendar", ""); err == nil {
		t.Fatal("expected ParseOperation to reject invalid input")
	}
}
