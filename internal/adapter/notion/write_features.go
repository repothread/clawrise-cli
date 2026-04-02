package notion

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
)

func providerDebugEnabled(ctx context.Context) bool {
	return adapter.RuntimeOptionsFromContext(ctx).DebugProviderPayload
}

func verifyAfterWriteEnabled(ctx context.Context) bool {
	return adapter.RuntimeOptionsFromContext(ctx).VerifyAfterWrite
}

func attachVerification(data map[string]any, verification map[string]any) map[string]any {
	if len(data) == 0 || len(verification) == 0 {
		return data
	}
	data["verification"] = verification
	return data
}

func buildVerificationResult(target string) map[string]any {
	return map[string]any{
		"enabled": true,
		"mode":    "read_after_write",
		"target":  target,
		"ok":      true,
		"checks":  []map[string]any{},
	}
}

func appendVerificationCheck(result map[string]any, check map[string]any) {
	if len(result) == 0 || len(check) == 0 {
		return
	}

	items, _ := result["checks"].([]map[string]any)
	items = append(items, check)
	result["checks"] = items

	ok, _ := asBool(check["ok"])
	if !ok {
		result["ok"] = false
	}
}

func appendVerificationError(result map[string]any, appErr *apperr.AppError) {
	if len(result) == 0 || appErr == nil {
		return
	}
	result["ok"] = false
	result["error"] = map[string]any{
		"code":          appErr.Code,
		"message":       appErr.Message,
		"retryable":     appErr.Retryable,
		"upstream_code": appErr.UpstreamCode,
		"http_status":   appErr.HTTPStatus,
	}
}

func blockSnapshots(raw any) []map[string]any {
	items, ok := raw.([]map[string]any)
	if !ok || len(items) == 0 {
		return nil
	}

	snapshots := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if len(item) == 0 {
			continue
		}
		blockType, _ := asString(item["type"])
		snapshots = append(snapshots, map[string]any{
			"type":       strings.TrimSpace(blockType),
			"plain_text": extractBlockPlainText(item),
		})
	}
	return snapshots
}

func cloneDebugValue(raw any) any {
	if raw == nil {
		return nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return raw
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return string(encoded)
	}
	return decoded
}

func decodeDebugResponseBody(data []byte) any {
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err == nil {
		return decoded
	}
	return string(data)
}

func (c *Client) verifyPageCreate(ctx context.Context, profile ExecutionProfile, payload map[string]any, data map[string]any) map[string]any {
	pageID, _ := asString(data["page_id"])
	result := buildVerificationResult("page")
	result["page_id"] = strings.TrimSpace(pageID)

	pageData, appErr := c.GetPage(ctx, profile, map[string]any{
		"page_id": pageID,
	})
	if appErr != nil {
		appendVerificationError(result, appErr)
		return result
	}

	appendVerificationCheck(result, map[string]any{
		"name":    "page_exists",
		"ok":      true,
		"page_id": pageID,
	})

	expectedTitle, _ := asString(data["title"])
	actualTitle, _ := asString(pageData["title"])
	appendVerificationCheck(result, map[string]any{
		"name":     "title_matches",
		"ok":       strings.TrimSpace(expectedTitle) == strings.TrimSpace(actualTitle),
		"expected": expectedTitle,
		"actual":   actualTitle,
	})

	children := blockSnapshots(payload["children"])
	if len(children) == 0 {
		return result
	}

	markdownData, appErr := c.GetPageMarkdown(ctx, profile, map[string]any{
		"page_id": pageID,
	})
	if appErr != nil {
		appendVerificationError(result, appErr)
		return result
	}

	markdown, _ := asString(markdownData["markdown"])
	for index, snapshot := range children {
		expectedText, _ := asString(snapshot["plain_text"])
		if strings.TrimSpace(expectedText) == "" {
			continue
		}
		appendVerificationCheck(result, map[string]any{
			"name":        "markdown_contains_child_text",
			"ok":          strings.Contains(markdown, expectedText),
			"child_index": index,
			"expected":    expectedText,
		})
	}
	return result
}

func (c *Client) verifyPageUpdate(ctx context.Context, profile ExecutionProfile, input map[string]any, data map[string]any) map[string]any {
	result := buildVerificationResult("page")
	pageID, _ := asString(data["page_id"])
	result["page_id"] = strings.TrimSpace(pageID)

	pageData, appErr := c.GetPage(ctx, profile, map[string]any{
		"page_id": pageID,
	})
	if appErr != nil {
		appendVerificationError(result, appErr)
		return result
	}

	appendVerificationCheck(result, map[string]any{
		"name":    "page_exists",
		"ok":      true,
		"page_id": pageID,
	})

	if expectedTitle, ok := asString(input["title"]); ok && strings.TrimSpace(expectedTitle) != "" {
		actualTitle, _ := asString(pageData["title"])
		appendVerificationCheck(result, map[string]any{
			"name":     "title_matches",
			"ok":       strings.TrimSpace(expectedTitle) == strings.TrimSpace(actualTitle),
			"expected": expectedTitle,
			"actual":   actualTitle,
		})
	}

	expectedArchived, ok := expectedPageArchivedState(input)
	if ok {
		actualArchived, _ := asBool(pageData["archived"])
		appendVerificationCheck(result, map[string]any{
			"name":     "archived_matches",
			"ok":       expectedArchived == actualArchived,
			"expected": expectedArchived,
			"actual":   actualArchived,
		})
	}

	return result
}

func expectedPageArchivedState(input map[string]any) (bool, bool) {
	if expectedInTrash, ok := asBool(input["in_trash"]); ok {
		return expectedInTrash, true
	}
	if expectedArchived, ok := asBool(input["archived"]); ok {
		return expectedArchived, true
	}
	return false, false
}

func (c *Client) verifyBlockAppend(ctx context.Context, profile ExecutionProfile, payload map[string]any, data map[string]any) map[string]any {
	result := buildVerificationResult("block_children")
	childIDs, _ := data["child_ids"].([]string)
	expectedBlocks := blockSnapshots(payload["children"])
	result["block_id"] = data["block_id"]

	appendVerificationCheck(result, map[string]any{
		"name":     "child_count_matches",
		"ok":       len(childIDs) == len(expectedBlocks),
		"expected": len(expectedBlocks),
		"actual":   len(childIDs),
	})

	limit := len(childIDs)
	if len(expectedBlocks) < limit {
		limit = len(expectedBlocks)
	}
	for index := 0; index < limit; index++ {
		blockData, appErr := c.GetBlock(ctx, profile, map[string]any{
			"block_id": childIDs[index],
		})
		if appErr != nil {
			appendVerificationError(result, appErr)
			return result
		}

		expectedType, _ := asString(expectedBlocks[index]["type"])
		expectedText, _ := asString(expectedBlocks[index]["plain_text"])
		actualType, _ := asString(blockData["type"])
		actualText, _ := asString(blockData["plain_text"])
		ok := strings.TrimSpace(expectedType) == strings.TrimSpace(actualType)
		if strings.TrimSpace(expectedText) != "" {
			ok = ok && expectedText == actualText
		}
		appendVerificationCheck(result, map[string]any{
			"name":                "child_matches",
			"ok":                  ok,
			"child_index":         index,
			"child_id":            childIDs[index],
			"expected_type":       expectedType,
			"actual_type":         actualType,
			"expected_plain_text": expectedText,
			"actual_plain_text":   actualText,
		})
	}
	return result
}

func (c *Client) verifyBlockUpdate(ctx context.Context, profile ExecutionProfile, payload map[string]any, data map[string]any) map[string]any {
	result := buildVerificationResult("block")
	blockID, _ := asString(data["block_id"])
	result["block_id"] = blockID

	blockData, appErr := c.GetBlock(ctx, profile, map[string]any{
		"block_id": blockID,
	})
	if appErr != nil {
		appendVerificationError(result, appErr)
		return result
	}

	expectedType, _ := asString(payload["type"])
	expectedText := extractBlockPlainText(payload)
	actualType, _ := asString(blockData["type"])
	actualText, _ := asString(blockData["plain_text"])
	ok := strings.TrimSpace(expectedType) == strings.TrimSpace(actualType)
	if strings.TrimSpace(expectedText) != "" {
		ok = ok && expectedText == actualText
	}
	appendVerificationCheck(result, map[string]any{
		"name":                "block_matches",
		"ok":                  ok,
		"block_id":            blockID,
		"expected_type":       expectedType,
		"actual_type":         actualType,
		"expected_plain_text": expectedText,
		"actual_plain_text":   actualText,
	})
	return result
}
