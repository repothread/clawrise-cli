package notion

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/adapter"
)

func TestCreateFileUploadSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/file_uploads" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode create file upload payload: %v", err)
			}
			if payload["mode"] != "single_part" || payload["filename"] != "demo.txt" || payload["content_type"] != "text/plain" {
				t.Fatalf("unexpected create file upload payload: %+v", payload)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"object":       "file_upload",
				"id":           "fu_123",
				"status":       "pending",
				"filename":     "demo.txt",
				"content_type": "text/plain",
			}), nil
		},
	})

	data, appErr := client.CreateFileUpload(context.Background(), testStaticProfile(), map[string]any{
		"mode":         "single_part",
		"filename":     "demo.txt",
		"content_type": "text/plain",
	})
	if appErr != nil {
		t.Fatalf("CreateFileUpload returned error: %+v", appErr)
	}
	if data["file_upload_id"] != "fu_123" || data["status"] != "pending" {
		t.Fatalf("unexpected create file upload result: %+v", data)
	}
}

func TestListFileUploadsSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/file_uploads" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", request.Method)
			}
			if request.URL.Query().Get("status") != "uploaded" {
				t.Fatalf("unexpected status query: %s", request.URL.Query().Get("status"))
			}
			if request.URL.Query().Get("page_size") != "20" {
				t.Fatalf("unexpected page_size query: %s", request.URL.Query().Get("page_size"))
			}
			if request.URL.Query().Get("start_cursor") != "cursor_demo" {
				t.Fatalf("unexpected start_cursor query: %s", request.URL.Query().Get("start_cursor"))
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"results": []map[string]any{
					{
						"object":       "file_upload",
						"id":           "fu_1",
						"status":       "uploaded",
						"filename":     "demo.txt",
						"content_type": "text/plain",
					},
				},
				"has_more":    true,
				"next_cursor": "cursor_next",
			}), nil
		},
	})

	data, appErr := client.ListFileUploads(context.Background(), testStaticProfile(), map[string]any{
		"status":     "uploaded",
		"page_size":  20,
		"page_token": "cursor_demo",
	})
	if appErr != nil {
		t.Fatalf("ListFileUploads returned error: %+v", appErr)
	}
	items := data["items"].([]map[string]any)
	if len(items) != 1 || items[0]["file_upload_id"] != "fu_1" {
		t.Fatalf("unexpected file upload list result: %+v", data)
	}
}

func TestGetFileUploadSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/file_uploads/fu_123" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			return jsonResponse(t, http.StatusOK, map[string]any{
				"object":       "file_upload",
				"id":           "fu_123",
				"status":       "uploaded",
				"filename":     "demo.txt",
				"content_type": "text/plain",
				"in_trash":     false,
			}), nil
		},
	})

	data, appErr := client.GetFileUpload(context.Background(), testStaticProfile(), map[string]any{
		"file_upload_id": "fu_123",
	})
	if appErr != nil {
		t.Fatalf("GetFileUpload returned error: %+v", appErr)
	}
	if data["file_upload_id"] != "fu_123" || data["status"] != "uploaded" {
		t.Fatalf("unexpected get file upload result: %+v", data)
	}
}

func TestSendFileUploadWithFilePathAndDebugSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	filePath := filepath.Join(t.TempDir(), "demo.txt")
	if err := os.WriteFile(filePath, []byte("hello file upload"), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	ctx := adapter.WithRuntimeOptions(context.Background(), adapter.RuntimeOptions{
		DebugProviderPayload: true,
	})
	ctx, _ = adapter.WithProviderDebugCapture(ctx)

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/file_uploads/fu_123/send" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
			if err != nil {
				t.Fatalf("failed to parse content type: %v", err)
			}
			if mediaType != "multipart/form-data" {
				t.Fatalf("unexpected media type: %s", mediaType)
			}

			reader := multipart.NewReader(request.Body, params["boundary"])
			partNumberSeen := false
			fileSeen := false
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("failed to iterate multipart parts: %v", err)
				}
				if part.FormName() == "part_number" {
					body, _ := io.ReadAll(part)
					if string(body) != "2" {
						t.Fatalf("unexpected part_number body: %s", string(body))
					}
					partNumberSeen = true
					continue
				}
				if part.FormName() == "file" {
					if part.FileName() != "demo.txt" {
						t.Fatalf("unexpected multipart file name: %s", part.FileName())
					}
					body, _ := io.ReadAll(part)
					if string(body) != "hello file upload" {
						t.Fatalf("unexpected multipart file body: %s", string(body))
					}
					fileSeen = true
				}
			}
			if !partNumberSeen || !fileSeen {
				t.Fatalf("expected multipart part_number and file, got part_number=%v file=%v", partNumberSeen, fileSeen)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"object":       "file_upload",
				"id":           "fu_123",
				"status":       "uploaded",
				"filename":     "demo.txt",
				"content_type": "text/plain",
			}), nil
		},
	})

	data, appErr := client.SendFileUpload(ctx, testStaticProfile(), map[string]any{
		"file_upload_id": "fu_123",
		"file_path":      filePath,
		"content_type":   "text/plain",
		"part_number":    2,
	})
	if appErr != nil {
		t.Fatalf("SendFileUpload returned error: %+v", appErr)
	}
	if data["status"] != "uploaded" {
		t.Fatalf("unexpected send file upload result: %+v", data)
	}
	debugData := adapter.ProviderDebugFromContext(ctx)
	if debugData == nil {
		t.Fatal("expected provider debug payload to be present")
	}
	requests := debugData["provider_requests"].([]map[string]any)
	if len(requests) != 1 {
		t.Fatalf("unexpected debug payload: %+v", debugData)
	}
	requestBody := requests[0]["request_body"].(map[string]any)
	if requestBody["content_length"] != float64(len("hello file upload")) && requestBody["content_length"] != len("hello file upload") {
		t.Fatalf("unexpected debug request body: %+v", requestBody)
	}
}

func TestSendFileUploadWithBase64ContentSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
			if err != nil {
				t.Fatalf("failed to parse content type: %v", err)
			}
			if mediaType != "multipart/form-data" {
				t.Fatalf("unexpected media type: %s", mediaType)
			}

			reader := multipart.NewReader(request.Body, params["boundary"])
			part, err := reader.NextPart()
			if err != nil {
				t.Fatalf("failed to read multipart file part: %v", err)
			}
			body, _ := io.ReadAll(part)
			if string(body) != "base64 payload" {
				t.Fatalf("unexpected multipart body: %s", string(body))
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"object":   "file_upload",
				"id":       "fu_456",
				"status":   "uploaded",
				"filename": "demo.txt",
			}), nil
		},
	})

	data, appErr := client.SendFileUpload(context.Background(), testStaticProfile(), map[string]any{
		"file_upload_id": "fu_456",
		"filename":       "demo.txt",
		"content_base64": base64.StdEncoding.EncodeToString([]byte("base64 payload")),
	})
	if appErr != nil {
		t.Fatalf("SendFileUpload returned error: %+v", appErr)
	}
	if data["file_upload_id"] != "fu_456" {
		t.Fatalf("unexpected send file upload result: %+v", data)
	}
}

func TestCompleteFileUploadSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/file_uploads/fu_123/complete" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode complete file upload payload: %v", err)
			}
			if len(payload) != 0 {
				t.Fatalf("expected empty complete payload, got: %+v", payload)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"object":   "file_upload",
				"id":       "fu_123",
				"status":   "uploaded",
				"filename": "demo.txt",
			}), nil
		},
	})

	data, appErr := client.CompleteFileUpload(context.Background(), testStaticProfile(), map[string]any{
		"file_upload_id": "fu_123",
	})
	if appErr != nil {
		t.Fatalf("CompleteFileUpload returned error: %+v", appErr)
	}
	if data["status"] != "uploaded" {
		t.Fatalf("unexpected complete file upload result: %+v", data)
	}
}
