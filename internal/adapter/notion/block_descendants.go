package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

// GetBlockDescendants 递归拉取整个块树，返回扁平后代列表。
func (c *Client) GetBlockDescendants(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	blockID, appErr := requireIDField(input, "block_id")
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	pageSize := 100
	if value, ok := asInt(input["page_size"]); ok && value > 0 {
		pageSize = value
	}

	items := make([]map[string]any, 0)
	visited := map[string]struct{}{}
	if appErr := c.collectBlockDescendants(ctx, accessToken, notionVersion, blockID, pageSize, 0, "", visited, &items); appErr != nil {
		return nil, appErr
	}

	return map[string]any{
		"block_id":           blockID,
		"items":              items,
		"total_descendants":  len(items),
		"traversal_strategy": "depth_first_flat_list",
	}, nil
}

// collectBlockDescendants 在适配器层处理递归和分页，避免调用方感知多层遍历细节。
func (c *Client) collectBlockDescendants(ctx context.Context, accessToken, notionVersion, blockID string, pageSize int, depth int, parentBlockID string, visited map[string]struct{}, items *[]map[string]any) *apperr.AppError {
	if _, exists := visited[blockID]; exists {
		return nil
	}
	visited[blockID] = struct{}{}

	pageToken := ""
	for {
		response, appErr := c.listBlockChildrenPage(ctx, accessToken, notionVersion, blockID, pageSize, pageToken)
		if appErr != nil {
			return appErr
		}

		for _, item := range response.Results {
			normalized := normalizeBlockData(item)
			normalized["depth"] = depth + 1
			normalized["parent_block_id"] = parentBlockID
			if strings.TrimSpace(parentBlockID) == "" {
				normalized["parent_block_id"] = blockID
			}
			*items = append(*items, normalized)

			if hasChildren, ok := asBool(item["has_children"]); ok && hasChildren {
				childID, _ := asString(item["id"])
				childID = strings.TrimSpace(childID)
				if childID != "" {
					if appErr := c.collectBlockDescendants(ctx, accessToken, notionVersion, childID, pageSize, depth+1, childID, visited, items); appErr != nil {
						return appErr
					}
				}
			}
		}

		if response.NextCursor == nil || strings.TrimSpace(*response.NextCursor) == "" || !response.HasMore {
			break
		}
		pageToken = strings.TrimSpace(*response.NextCursor)
	}

	return nil
}

func (c *Client) listBlockChildrenPage(ctx context.Context, accessToken, notionVersion, blockID string, pageSize int, pageToken string) (notionBlockChildrenResponse, *apperr.AppError) {
	query := url.Values{}
	if pageSize > 0 {
		query.Set("page_size", fmt.Sprintf("%d", pageSize))
	}
	if strings.TrimSpace(pageToken) != "" {
		query.Set("start_cursor", strings.TrimSpace(pageToken))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/v1/blocks/"+url.PathEscape(blockID)+"/children",
		query,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return notionBlockChildrenResponse{}, appErr
	}

	var response notionBlockChildrenResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return notionBlockChildrenResponse{}, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion block children response: %v", err))
	}
	return response, nil
}
