package notion

import (
	"context"
	"fmt"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

type pageGraphQueueItem struct {
	PageID string
	Depth  int
}

// ReadPageGraph reads one page plus related pages discovered from relation properties.
// 先把单页读全，再沿 relation 属性做有界扩展，适合 AI 做页面级知识图读取。
func (c *Client) ReadPageGraph(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	rootPageID, appErr := requireIDField(input, "page_id")
	if appErr != nil {
		return nil, appErr
	}

	maxDepth := 1
	if value, ok := asInt(input["max_depth"]); ok && value >= 0 {
		maxDepth = value
	}
	maxNodes := 20
	if value, ok := asInt(input["max_nodes"]); ok && value > 0 {
		maxNodes = value
	}
	stopOnError := false
	if value, ok := asBool(input["stop_on_error"]); ok {
		stopOnError = value
	}

	relationProperties, appErr := normalizeRelationPropertyNames(input["relation_properties"])
	if appErr != nil {
		return nil, appErr
	}
	if appErr := validateReadGraphSelection(input["filter_properties"], relationProperties); appErr != nil {
		return nil, appErr
	}

	queue := []pageGraphQueueItem{{PageID: rootPageID, Depth: 0}}
	visited := map[string]struct{}{}
	nodes := make([]map[string]any, 0)
	edges := make([]map[string]any, 0)
	errors := make([]map[string]any, 0)
	truncated := false

	for len(queue) > 0 {
		if len(visited) >= maxNodes {
			truncated = true
			break
		}

		current := queue[0]
		queue = queue[1:]
		current.PageID = strings.TrimSpace(current.PageID)
		if current.PageID == "" {
			continue
		}
		if _, exists := visited[current.PageID]; exists {
			continue
		}
		visited[current.PageID] = struct{}{}

		pageInput := buildReadGraphPageInput(current.PageID, input)
		pageData, appErr := c.ReadCompletePage(ctx, profile, pageInput)
		if appErr != nil {
			errors = append(errors, map[string]any{
				"page_id": current.PageID,
				"depth":   current.Depth,
				"error": map[string]any{
					"code":          appErr.Code,
					"message":       appErr.Message,
					"retryable":     appErr.Retryable,
					"upstream_code": appErr.UpstreamCode,
					"http_status":   appErr.HTTPStatus,
				},
			})
			if stopOnError {
				return nil, appErr
			}
			continue
		}

		nodes = append(nodes, map[string]any{
			"page_id": current.PageID,
			"depth":   current.Depth,
			"data":    pageData,
		})

		if current.Depth >= maxDepth {
			continue
		}

		itemsByName := extractReadGraphPropertyItemsByName(pageData)
		for propertyName, propertyItem := range itemsByName {
			if len(relationProperties) > 0 {
				if _, exists := relationProperties[propertyName]; !exists {
					continue
				}
			} else {
				propertyType, _ := asString(propertyItem["property_type"])
				if strings.TrimSpace(propertyType) != "relation" {
					continue
				}
			}

			relationIDs := extractRelationPageIDsFromCompletePropertyItem(propertyItem)
			for _, relatedPageID := range relationIDs {
				relatedPageID = strings.TrimSpace(relatedPageID)
				if relatedPageID == "" {
					continue
				}
				edges = append(edges, map[string]any{
					"from_page_id":  current.PageID,
					"to_page_id":    relatedPageID,
					"property_name": propertyName,
				})
				if _, exists := visited[relatedPageID]; exists {
					continue
				}
				queue = append(queue, pageGraphQueueItem{
					PageID: relatedPageID,
					Depth:  current.Depth + 1,
				})
			}
		}
	}

	if len(queue) > 0 {
		truncated = true
	}

	return map[string]any{
		"root_page_id": rootPageID,
		"max_depth":    maxDepth,
		"max_nodes":    maxNodes,
		"truncated":    truncated,
		"node_count":   len(nodes),
		"edge_count":   len(edges),
		"error_count":  len(errors),
		"nodes":        nodes,
		"edges":        edges,
		"errors":       errors,
	}, nil
}

func buildReadGraphPageInput(pageID string, input map[string]any) map[string]any {
	pageInput := map[string]any{
		"page_id":                pageID,
		"include_property_items": true,
	}
	copyOptionalTaskFields(input, pageInput, "filter_properties", "property_item_page_size", "include_markdown", "include_transcript", "expand_unknown_blocks", "unknown_block_limit")
	return pageInput
}

func normalizeRelationPropertyNames(raw any) (map[string]struct{}, *apperr.AppError) {
	if raw == nil {
		return nil, nil
	}
	items, ok := asArray(raw)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "relation_properties must be an array")
	}

	result := map[string]struct{}{}
	for _, item := range items {
		name, ok := asString(item)
		if !ok || strings.TrimSpace(name) == "" {
			return nil, apperr.New("INVALID_INPUT", "each relation_properties item must be a non-empty string")
		}
		result[strings.TrimSpace(name)] = struct{}{}
	}
	return result, nil
}

func extractReadGraphPropertyItemsByName(pageData map[string]any) map[string]map[string]any {
	propertyItems, _ := asMap(pageData["property_items"])
	itemsByNameRaw, _ := asMap(propertyItems["items_by_name"])
	if len(itemsByNameRaw) == 0 {
		return nil
	}

	itemsByName := make(map[string]map[string]any, len(itemsByNameRaw))
	for propertyName, rawItem := range itemsByNameRaw {
		item, ok := asMap(rawItem)
		if !ok || len(item) == 0 {
			continue
		}
		itemsByName[propertyName] = item
	}
	return itemsByName
}

func extractRelationPageIDsFromCompletePropertyItem(item map[string]any) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0)

	if propertyItem, ok := asMap(item["property_item"]); ok {
		collectRelationPageIDs(propertyItem, seen, &result)
	}
	if rawItems, ok := item["items"].([]map[string]any); ok {
		for _, child := range rawItems {
			collectRelationPageIDs(child, seen, &result)
		}
	}
	if rawItems, ok := asArray(item["items"]); ok {
		for _, child := range rawItems {
			childMap, ok := asMap(child)
			if !ok {
				continue
			}
			collectRelationPageIDs(childMap, seen, &result)
		}
	}
	return result
}

func collectRelationPageIDs(record map[string]any, seen map[string]struct{}, result *[]string) {
	if len(record) == 0 {
		return
	}

	if relationRecord, ok := asMap(record["relation"]); ok {
		appendRelationPageID(relationRecord, seen, result)
	}
	if relationList, ok := asArray(record["relation"]); ok {
		for _, item := range relationList {
			relationRecord, ok := asMap(item)
			if !ok {
				continue
			}
			appendRelationPageID(relationRecord, seen, result)
		}
	}
	if nestedPropertyItem, ok := asMap(record["property_item"]); ok {
		collectRelationPageIDs(nestedPropertyItem, seen, result)
	}
	if results, ok := asArray(record["results"]); ok {
		for _, item := range results {
			child, ok := asMap(item)
			if !ok {
				continue
			}
			collectRelationPageIDs(child, seen, result)
		}
	}
}

func appendRelationPageID(record map[string]any, seen map[string]struct{}, result *[]string) {
	pageID := extractFirstString(record, "id", "page_id")
	if strings.TrimSpace(pageID) == "" {
		return
	}
	pageID = strings.TrimSpace(pageID)
	if _, exists := seen[pageID]; exists {
		return
	}
	seen[pageID] = struct{}{}
	*result = append(*result, pageID)
}

func validateReadGraphSelection(filterProperties any, relationProperties map[string]struct{}) *apperr.AppError {
	if filterProperties == nil || len(relationProperties) == 0 {
		return nil
	}
	filterItems, ok := asArray(filterProperties)
	if !ok {
		return apperr.New("INVALID_INPUT", "filter_properties must be an array")
	}
	allowed := map[string]struct{}{}
	for _, item := range filterItems {
		name, ok := asString(item)
		if ok && strings.TrimSpace(name) != "" {
			allowed[strings.TrimSpace(name)] = struct{}{}
		}
	}
	for propertyName := range relationProperties {
		if _, exists := allowed[propertyName]; !exists {
			return apperr.New("INVALID_INPUT", fmt.Sprintf("relation_properties contains %q but filter_properties excludes it", propertyName))
		}
	}
	return nil
}
