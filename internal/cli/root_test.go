package cli

import (
	"bytes"
	"testing"
)

func TestRunOperationDryRun(t *testing.T) {
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")
	t.Setenv("CLAWRISE_CONFIG", "../../examples/config.example.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{
		"feishu.calendar.event.create",
		"--dry-run",
		"--json",
		`{"calendar_id":"cal_demo","summary":"Demo Event","start_at":"2026-03-30T10:00:00+08:00","end_at":"2026-03-30T11:00:00+08:00"}`,
	}, Dependencies{
		Version: "test",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"ok": true`)) {
		t.Fatalf("expected success output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"context"`)) {
		t.Fatalf("expected context output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "bot"`)) {
		t.Fatalf("expected bot subject in output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSubjectUse(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"subject", "use", "bot"}, Dependencies{
		Version: "test",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "bot"`)) {
		t.Fatalf("expected subject output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSpecList(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"spec", "list"}, Dependencies{
		Version: "test",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"full_path": "feishu"`)) {
		t.Fatalf("expected feishu in spec list output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"full_path": "notion"`)) {
		t.Fatalf("expected notion in spec list output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSpecGet(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"spec", "get", "notion.page.create"}, Dependencies{
		Version: "test",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"operation": "notion.page.create"`)) {
		t.Fatalf("expected operation output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"implemented": true`)) {
		t.Fatalf("expected implemented flag, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}
