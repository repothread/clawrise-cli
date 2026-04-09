package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestAttachFileBlockUploadsAndAppendsImageBlock(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "screenshot.png")
	if err := os.WriteFile(filePath, []byte("png-bytes"), 0o644); err != nil {
		t.Fatalf("failed to write temp image file: %v", err)
	}

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/file_uploads":
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode create file upload payload: %v", err)
				}
				if payload["mode"] != "single_part" || payload["filename"] != "screenshot.png" {
					t.Fatalf("unexpected create upload payload: %+v", payload)
				}
				if payload["content_type"] != "image/png" {
					t.Fatalf("expected inferred image/png content type, got: %+v", payload)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":       "file_upload",
					"id":           "fu_attach_123",
					"status":       "pending",
					"filename":     "screenshot.png",
					"content_type": "image/png",
				}), nil
			case "/v1/file_uploads/fu_attach_123/send":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":       "file_upload",
					"id":           "fu_attach_123",
					"status":       "uploaded",
					"filename":     "screenshot.png",
					"content_type": "image/png",
				}), nil
			case "/v1/blocks/page_demo/children":
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode append block payload: %v", err)
				}
				position := payload["position"].(map[string]any)
				if position["type"] != "start" {
					t.Fatalf("unexpected append position payload: %+v", position)
				}
				children := payload["children"].([]any)
				if len(children) != 1 {
					t.Fatalf("unexpected child count: %+v", children)
				}
				child := children[0].(map[string]any)
				if child["type"] != "image" {
					t.Fatalf("expected inferred image block type: %+v", child)
				}
				imageBody := child["image"].(map[string]any)
				if imageBody["type"] != "file_upload" || imageBody["file_upload"].(map[string]any)["id"] != "fu_attach_123" {
					t.Fatalf("unexpected image block payload: %+v", imageBody)
				}
				caption := imageBody["caption"].([]any)
				if len(caption) != 1 || caption[0].(map[string]any)["text"].(map[string]any)["content"] != "最新截图" {
					t.Fatalf("unexpected caption payload: %+v", imageBody["caption"])
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{"id": "blk_uploaded_1"},
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.AttachFileBlock(context.Background(), testStaticProfile(), map[string]any{
		"block_id":  "page_demo",
		"file_path": filePath,
		"caption":   "最新截图",
		"position": map[string]any{
			"type": "start",
		},
	})
	if appErr != nil {
		t.Fatalf("AttachFileBlock returned error: %+v", appErr)
	}
	if data["block_type"] != "image" {
		t.Fatalf("unexpected block_type: %+v", data["block_type"])
	}
	childIDs := data["child_ids"].([]string)
	if len(childIDs) != 1 || childIDs[0] != "blk_uploaded_1" {
		t.Fatalf("unexpected child ids: %+v", childIDs)
	}
}

func TestAttachFileBlockSupportsBase64FileBlock(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/file_uploads":
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode create file upload payload: %v", err)
				}
				if payload["filename"] != "demo.txt" || payload["content_type"] != "text/plain" {
					t.Fatalf("unexpected create upload payload: %+v", payload)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":       "file_upload",
					"id":           "fu_attach_456",
					"status":       "pending",
					"filename":     "demo.txt",
					"content_type": "text/plain",
				}), nil
			case "/v1/file_uploads/fu_attach_456/send":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":       "file_upload",
					"id":           "fu_attach_456",
					"status":       "uploaded",
					"filename":     "demo.txt",
					"content_type": "text/plain",
				}), nil
			case "/v1/blocks/page_demo/children":
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode append block payload: %v", err)
				}
				child := payload["children"].([]any)[0].(map[string]any)
				if child["type"] != "file" {
					t.Fatalf("expected explicit file block type: %+v", child)
				}
				fileBody := child["file"].(map[string]any)
				if fileBody["type"] != "file_upload" || fileBody["file_upload"].(map[string]any)["id"] != "fu_attach_456" {
					t.Fatalf("unexpected file block payload: %+v", fileBody)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{"id": "blk_uploaded_2"},
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.AttachFileBlock(context.Background(), testStaticProfile(), map[string]any{
		"block_id":       "page_demo",
		"content_base64": "SGVsbG8=",
		"filename":       "demo.txt",
		"content_type":   "text/plain",
		"block_type":     "file",
	})
	if appErr != nil {
		t.Fatalf("AttachFileBlock returned error: %+v", appErr)
	}
	if data["block_type"] != "file" {
		t.Fatalf("unexpected block_type: %+v", data["block_type"])
	}
}

func TestAttachFileBlockRejectsMissingSource(t *testing.T) {
	_, appErr := buildTaskBlockFileSpec(map[string]any{
		"filename": "demo.txt",
	})
	if appErr == nil {
		t.Fatal("expected buildTaskBlockFileSpec to reject missing source")
	}
	if appErr.Code != "INVALID_INPUT" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}
