package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestPatchPageSectionUpdatesMatchedHeading(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/pages/page_demo/markdown":
				switch request.Method {
				case http.MethodGet:
					return jsonResponse(t, http.StatusOK, map[string]any{
						"object":            "page_markdown",
						"id":                "page_demo",
						"markdown":          "# Weekly Review\n\n## Risks\n\nOld risk\n\n## Notes\n\nKeep me",
						"truncated":         false,
						"unknown_block_ids": []string{},
					}), nil
				case http.MethodPatch:
					var payload map[string]any
					if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
						t.Fatalf("failed to decode markdown patch payload: %v", err)
					}
					replaceContent := payload["replace_content"].(map[string]any)
					expected := "# Weekly Review\n\n## Risks\n\n- New risk\n\n## Notes\n\nKeep me"
					if replaceContent["new_str"] != expected {
						t.Fatalf("unexpected patched markdown: %+v", replaceContent["new_str"])
					}
					return jsonResponse(t, http.StatusOK, map[string]any{
						"object":            "page_markdown",
						"id":                "page_demo",
						"markdown":          expected,
						"truncated":         false,
						"unknown_block_ids": []string{},
					}), nil
				default:
					t.Fatalf("unexpected method: %s", request.Method)
					return nil, nil
				}
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.PatchPageSection(context.Background(), testStaticProfile(), map[string]any{
		"page_id":       "page_demo",
		"heading":       "Risks",
		"heading_level": 2,
		"markdown":      "- New risk",
	})
	if appErr != nil {
		t.Fatalf("PatchPageSection returned error: %+v", appErr)
	}
	if data["action"] != "updated" {
		t.Fatalf("unexpected action: %+v", data)
	}
}

func TestPatchPageSectionRejectsUnknownBlocksByDefault(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/pages/page_demo/markdown" || request.Method != http.MethodGet {
				t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
			}
			return jsonResponse(t, http.StatusOK, map[string]any{
				"object":            "page_markdown",
				"id":                "page_demo",
				"markdown":          "# Weekly Review",
				"truncated":         false,
				"unknown_block_ids": []string{"blk_unknown_1"},
			}), nil
		},
	})

	_, appErr := client.PatchPageSection(context.Background(), testStaticProfile(), map[string]any{
		"page_id":  "page_demo",
		"heading":  "Risks",
		"markdown": "- New risk",
	})
	if appErr == nil {
		t.Fatal("expected PatchPageSection to reject unknown blocks by default")
	}
	if appErr.Code != "UNSAFE_PAGE_CONTENT" {
		t.Fatalf("unexpected error code: %+v", appErr)
	}
}
