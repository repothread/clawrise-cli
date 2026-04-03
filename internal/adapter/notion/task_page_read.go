package notion

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// ReadCompletePage 尽量读全一个页面，聚合基础 page、完整属性项和递归补拉的 markdown 子树。
// ReadCompletePage reads one page as completely as possible by aggregating base page data, complete property items, and recursively fetched markdown subtrees.
func (c *Client) ReadCompletePage(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	pageID, appErr := requireIDField(input, "page_id")
	if appErr != nil {
		return nil, appErr
	}

	pageInput := map[string]any{
		"page_id": pageID,
	}
	if filterProperties, exists := input["filter_properties"]; exists {
		pageInput["filter_properties"] = cloneDebugValue(filterProperties)
	}

	pageData, appErr := c.GetPage(ctx, profile, pageInput)
	if appErr != nil {
		return nil, appErr
	}

	includePropertyItems := true
	if value, ok := asBool(input["include_property_items"]); ok {
		includePropertyItems = value
	}

	propertyItemsResult := map[string]any{
		"enabled": includePropertyItems,
	}
	if includePropertyItems {
		pageSize := 100
		if value, ok := asInt(input["property_item_page_size"]); ok && value > 0 {
			pageSize = value
		}

		properties, _ := asMap(pageData["properties"])
		itemsByName, appErr := c.readCompletePropertyItems(ctx, profile, pageID, properties, input["filter_properties"], pageSize)
		if appErr != nil {
			return nil, appErr
		}
		propertyItemsResult["items_by_name"] = itemsByName
		propertyItemsResult["completed_count"] = len(itemsByName)
		propertyItemsResult["page_size"] = pageSize
	}

	includeMarkdown := true
	if value, ok := asBool(input["include_markdown"]); ok {
		includeMarkdown = value
	}

	markdownResult := map[string]any{
		"enabled": includeMarkdown,
	}
	if includeMarkdown {
		includeTranscript, _ := asBool(input["include_transcript"])
		expandUnknownBlocks := true
		if value, ok := asBool(input["expand_unknown_blocks"]); ok {
			expandUnknownBlocks = value
		}
		unknownBlockLimit := 20
		if value, ok := asInt(input["unknown_block_limit"]); ok && value > 0 {
			unknownBlockLimit = value
		}

		markdownData, appErr := c.readCompleteMarkdown(ctx, profile, pageID, includeTranscript, expandUnknownBlocks, unknownBlockLimit)
		if appErr != nil {
			return nil, appErr
		}
		markdownResult = markdownData
	}

	return map[string]any{
		"page":           cloneMap(pageData),
		"property_items": propertyItemsResult,
		"markdown":       markdownResult,
	}, nil
}

// readCompletePropertyItems 把页面属性逐个补拉为完整 property item 结果，并处理分页。
// readCompletePropertyItems completes each selected page property through the property item endpoint and exhausts pagination when needed.
func (c *Client) readCompletePropertyItems(ctx context.Context, profile ExecutionProfile, pageID string, properties map[string]any, rawFilterProperties any, pageSize int) (map[string]any, *apperr.AppError) {
	selectedNames, appErr := normalizeTaskPagePropertySelection(properties, rawFilterProperties)
	if appErr != nil {
		return nil, appErr
	}

	itemsByName := map[string]any{}
	for _, propertyName := range selectedNames {
		propertyRecord, _ := asMap(properties[propertyName])
		propertyID, ok := asString(propertyRecord["id"])
		if !ok || strings.TrimSpace(propertyID) == "" {
			continue
		}

		propertyType, _ := asString(propertyRecord["type"])
		completedItem, appErr := c.collectCompletePagePropertyItem(ctx, profile, pageID, strings.TrimSpace(propertyID), pageSize)
		if appErr != nil {
			return nil, appErr
		}
		completedItem["property_name"] = propertyName
		completedItem["property_type"] = strings.TrimSpace(propertyType)
		itemsByName[propertyName] = completedItem
	}
	return itemsByName, nil
}

// collectCompletePagePropertyItem 循环调用 property item endpoint，直到拿到一个完整属性结果。
// collectCompletePagePropertyItem loops through the property item endpoint until it has one complete property result.
func (c *Client) collectCompletePagePropertyItem(ctx context.Context, profile ExecutionProfile, pageID string, propertyID string, pageSize int) (map[string]any, *apperr.AppError) {
	pageToken := ""
	var aggregated map[string]any
	var aggregatedItems []map[string]any

	for {
		input := map[string]any{
			"page_id":     pageID,
			"property_id": propertyID,
			"page_size":   pageSize,
		}
		if strings.TrimSpace(pageToken) != "" {
			input["page_token"] = pageToken
		}

		itemData, appErr := c.GetPagePropertyItem(ctx, profile, input)
		if appErr != nil {
			return nil, appErr
		}

		if aggregated == nil {
			aggregated = cloneMap(itemData)
			delete(aggregated, "items")
		}

		items, hasItems := itemData["items"].([]map[string]any)
		if !hasItems {
			return cloneMap(itemData), nil
		}
		aggregatedItems = append(aggregatedItems, items...)

		hasMore, _ := asBool(itemData["has_more"])
		nextPageToken, _ := asString(itemData["next_page_token"])
		if !hasMore || strings.TrimSpace(nextPageToken) == "" {
			aggregated["items"] = aggregatedItems
			aggregated["has_more"] = false
			aggregated["next_page_token"] = ""
			aggregated["item_count"] = len(aggregatedItems)
			return aggregated, nil
		}
		pageToken = strings.TrimSpace(nextPageToken)
	}
}

// readCompleteMarkdown 先获取页面 markdown，再递归补拉 unknown_block_ids 并生成带附录的输出。
// readCompleteMarkdown reads page markdown first, then recursively resolves unknown_block_ids and builds an appendix-friendly output.
func (c *Client) readCompleteMarkdown(ctx context.Context, profile ExecutionProfile, pageID string, includeTranscript bool, expandUnknownBlocks bool, unknownBlockLimit int) (map[string]any, *apperr.AppError) {
	rootData, appErr := c.GetPageMarkdown(ctx, profile, map[string]any{
		"page_id":            pageID,
		"include_transcript": includeTranscript,
	})
	if appErr != nil {
		return nil, appErr
	}

	rootMarkdown, _ := asString(rootData["markdown"])
	rootUnknownIDs := toStringSlice(rootData["unknown_block_ids"])

	result := map[string]any{
		"enabled":                   true,
		"page_id":                   pageID,
		"markdown":                  rootMarkdown,
		"truncated":                 rootData["truncated"],
		"unknown_block_ids":         rootUnknownIDs,
		"expanded":                  expandUnknownBlocks,
		"unknown_block_limit":       unknownBlockLimit,
		"resolved_unknown_blocks":   []map[string]any{},
		"unknown_block_errors":      []map[string]any{},
		"skipped_unknown_block_ids": []string{},
		"markdown_with_appendices":  rootMarkdown,
	}
	if !expandUnknownBlocks || len(rootUnknownIDs) == 0 {
		return result, nil
	}

	type markdownQueueItem struct {
		BlockID string
		Depth   int
	}

	queue := make([]markdownQueueItem, 0, len(rootUnknownIDs))
	visited := map[string]struct{}{strings.TrimSpace(pageID): {}}
	for _, blockID := range rootUnknownIDs {
		blockID = strings.TrimSpace(blockID)
		if blockID == "" {
			continue
		}
		if _, exists := visited[blockID]; exists {
			continue
		}
		visited[blockID] = struct{}{}
		queue = append(queue, markdownQueueItem{BlockID: blockID, Depth: 1})
	}

	resolvedSegments := make([]map[string]any, 0)
	errorSegments := make([]map[string]any, 0)
	fetchedCount := 0
	for len(queue) > 0 {
		if unknownBlockLimit > 0 && fetchedCount >= unknownBlockLimit {
			break
		}

		current := queue[0]
		queue = queue[1:]
		fetchedCount++

		segmentData, appErr := c.GetPageMarkdown(ctx, profile, map[string]any{
			"page_id":            current.BlockID,
			"include_transcript": includeTranscript,
		})
		if appErr != nil {
			errorSegments = append(errorSegments, map[string]any{
				"block_id": current.BlockID,
				"depth":    current.Depth,
				"error": map[string]any{
					"code":          appErr.Code,
					"message":       appErr.Message,
					"retryable":     appErr.Retryable,
					"upstream_code": appErr.UpstreamCode,
					"http_status":   appErr.HTTPStatus,
				},
			})
			continue
		}

		segmentUnknownIDs := toStringSlice(segmentData["unknown_block_ids"])
		resolvedSegment := map[string]any{
			"block_id":          current.BlockID,
			"depth":             current.Depth,
			"markdown":          segmentData["markdown"],
			"truncated":         segmentData["truncated"],
			"unknown_block_ids": segmentUnknownIDs,
		}
		resolvedSegments = append(resolvedSegments, resolvedSegment)

		for _, childBlockID := range segmentUnknownIDs {
			childBlockID = strings.TrimSpace(childBlockID)
			if childBlockID == "" {
				continue
			}
			if _, exists := visited[childBlockID]; exists {
				continue
			}
			visited[childBlockID] = struct{}{}
			queue = append(queue, markdownQueueItem{
				BlockID: childBlockID,
				Depth:   current.Depth + 1,
			})
		}
	}

	skippedIDs := make([]string, 0, len(queue))
	for _, item := range queue {
		if strings.TrimSpace(item.BlockID) != "" {
			skippedIDs = append(skippedIDs, strings.TrimSpace(item.BlockID))
		}
	}

	result["resolved_unknown_blocks"] = resolvedSegments
	result["unknown_block_errors"] = errorSegments
	result["skipped_unknown_block_ids"] = skippedIDs
	result["markdown_with_appendices"] = buildMarkdownWithAppendices(rootMarkdown, resolvedSegments)
	return result, nil
}

// normalizeTaskPagePropertySelection 把可选的 filter_properties 解析成稳定的属性名列表。
// normalizeTaskPagePropertySelection normalizes optional filter_properties into a stable list of property names.
func normalizeTaskPagePropertySelection(properties map[string]any, rawFilterProperties any) ([]string, *apperr.AppError) {
	if len(properties) == 0 {
		return nil, nil
	}

	if rawFilterProperties == nil {
		names := make([]string, 0, len(properties))
		for propertyName := range properties {
			names = append(names, propertyName)
		}
		sort.Strings(names)
		return names, nil
	}

	filterProperties, ok := asArray(rawFilterProperties)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "filter_properties must be an array")
	}

	selected := make([]string, 0, len(filterProperties))
	for _, item := range filterProperties {
		propertyName, ok := asString(item)
		if !ok || strings.TrimSpace(propertyName) == "" {
			return nil, apperr.New("INVALID_INPUT", "each filter_properties item must be a non-empty string")
		}
		propertyName = strings.TrimSpace(propertyName)
		if _, exists := properties[propertyName]; !exists {
			return nil, apperr.New("INVALID_INPUT", fmt.Sprintf("filter_properties contains unknown property %q", propertyName))
		}
		selected = append(selected, propertyName)
	}
	sort.Strings(selected)
	return selected, nil
}

// buildMarkdownWithAppendices 在主 markdown 后追加递归补拉成功的子树片段，方便 AI 一次性消费。
// buildMarkdownWithAppendices appends successfully fetched markdown subtrees after the root markdown so AI callers can consume one expanded text blob.
func buildMarkdownWithAppendices(rootMarkdown string, resolvedSegments []map[string]any) string {
	result := rootMarkdown
	if len(resolvedSegments) == 0 {
		return result
	}

	var builder strings.Builder
	builder.WriteString(result)
	builder.WriteString("\n\n---\n\n## Fetched Unknown Block Appendices\n")
	for _, segment := range resolvedSegments {
		blockID, _ := asString(segment["block_id"])
		markdown, _ := asString(segment["markdown"])
		builder.WriteString("\n\n### Block ")
		builder.WriteString(strings.TrimSpace(blockID))
		builder.WriteString("\n\n")
		builder.WriteString(markdown)
	}
	return builder.String()
}

func toStringSlice(raw any) []string {
	items, ok := asArray(raw)
	if !ok || len(items) == 0 {
		return nil
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := asString(item)
		if !ok || strings.TrimSpace(text) == "" {
			continue
		}
		result = append(result, strings.TrimSpace(text))
	}
	return result
}
