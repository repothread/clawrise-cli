package notion

import (
	"context"
	"fmt"
	neturl "net/url"
	"regexp"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

var notionObjectIDPattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}`)

// ResolveDatabaseTarget resolves one user-facing Notion target into database/data source/page context.
// 这个 task 面向 AI 场景：用户常给 URL、page id 或 data source id，而不是一步到位的 database_id。
func (c *Client) ResolveDatabaseTarget(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	selectorType, selectorValue, appErr := normalizeDatabaseTargetSelector(input)
	if appErr != nil {
		return nil, appErr
	}

	dataSourceName := ""
	if value, ok := asString(input["data_source_name"]); ok && strings.TrimSpace(value) != "" {
		dataSourceName = strings.TrimSpace(value)
	}

	result := map[string]any{
		"input_type":  selectorType,
		"input_value": selectorValue,
	}

	switch selectorType {
	case "database_id":
		return c.resolveDatabaseTargetFromDatabase(ctx, profile, selectorValue, dataSourceName, selectorType, result)
	case "data_source_id":
		return c.resolveDatabaseTargetFromDataSource(ctx, profile, selectorValue, dataSourceName, selectorType, result)
	case "page_id":
		return c.resolveDatabaseTargetFromPage(ctx, profile, selectorValue, dataSourceName, selectorType, result)
	case "url":
		candidateID, ok := extractNotionObjectID(selectorValue)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "failed to extract a Notion object id from url")
		}
		result["candidate_id"] = candidateID
		return c.resolveDatabaseTargetFromUnknownID(ctx, profile, candidateID, dataSourceName, "url", result)
	case "target":
		if looksLikeURL(selectorValue) {
			candidateID, ok := extractNotionObjectID(selectorValue)
			if !ok {
				return nil, apperr.New("INVALID_INPUT", "failed to extract a Notion object id from target")
			}
			result["candidate_id"] = candidateID
			return c.resolveDatabaseTargetFromUnknownID(ctx, profile, candidateID, dataSourceName, "target_url", result)
		}

		candidateID, ok := normalizeNotionObjectID(selectorValue)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "target must be a Notion id or a Notion URL")
		}
		result["candidate_id"] = candidateID
		return c.resolveDatabaseTargetFromUnknownID(ctx, profile, candidateID, dataSourceName, "target_id", result)
	default:
		return nil, apperr.New("INVALID_INPUT", "unsupported selector type")
	}
}

func (c *Client) resolveDatabaseTargetFromUnknownID(ctx context.Context, profile ExecutionProfile, objectID string, dataSourceName string, resolutionPath string, result map[string]any) (map[string]any, *apperr.AppError) {
	if databaseData, appErr := c.GetDatabase(ctx, profile, map[string]any{"database_id": objectID}); appErr == nil {
		result["resolution_path"] = resolutionPath + "_database"
		return c.attachResolvedDatabaseContext(ctx, profile, result, databaseData, dataSourceName)
	} else if !isNotionResourceNotFound(appErr) {
		return nil, appErr
	}

	if dataSourceData, appErr := c.GetDataSource(ctx, profile, map[string]any{"data_source_id": objectID}); appErr == nil {
		result["resolution_path"] = resolutionPath + "_data_source"
		return c.attachResolvedDataSourceContext(ctx, profile, result, dataSourceData, dataSourceName)
	} else if !isNotionResourceNotFound(appErr) {
		return nil, appErr
	}

	if pageData, appErr := c.GetPage(ctx, profile, map[string]any{"page_id": objectID}); appErr == nil {
		result["resolution_path"] = resolutionPath + "_page"
		return c.attachResolvedPageContext(ctx, profile, result, pageData, dataSourceName)
	} else if !isNotionResourceNotFound(appErr) {
		return nil, appErr
	}

	return nil, apperr.New("RESOURCE_NOT_FOUND", fmt.Sprintf("failed to resolve Notion target from id %s", objectID))
}

func (c *Client) resolveDatabaseTargetFromDatabase(ctx context.Context, profile ExecutionProfile, databaseID string, dataSourceName string, inputType string, result map[string]any) (map[string]any, *apperr.AppError) {
	databaseData, appErr := c.GetDatabase(ctx, profile, map[string]any{
		"database_id": databaseID,
	})
	if appErr != nil {
		return nil, appErr
	}

	result["resolution_path"] = inputType
	return c.attachResolvedDatabaseContext(ctx, profile, result, databaseData, dataSourceName)
}

func (c *Client) resolveDatabaseTargetFromDataSource(ctx context.Context, profile ExecutionProfile, dataSourceID string, dataSourceName string, inputType string, result map[string]any) (map[string]any, *apperr.AppError) {
	dataSourceData, appErr := c.GetDataSource(ctx, profile, map[string]any{
		"data_source_id": dataSourceID,
	})
	if appErr != nil {
		return nil, appErr
	}

	result["resolution_path"] = inputType
	return c.attachResolvedDataSourceContext(ctx, profile, result, dataSourceData, dataSourceName)
}

func (c *Client) resolveDatabaseTargetFromPage(ctx context.Context, profile ExecutionProfile, pageID string, dataSourceName string, inputType string, result map[string]any) (map[string]any, *apperr.AppError) {
	pageData, appErr := c.GetPage(ctx, profile, map[string]any{
		"page_id": pageID,
	})
	if appErr != nil {
		return nil, appErr
	}

	result["resolution_path"] = inputType
	return c.attachResolvedPageContext(ctx, profile, result, pageData, dataSourceName)
}

func (c *Client) attachResolvedDatabaseContext(ctx context.Context, profile ExecutionProfile, result map[string]any, databaseData map[string]any, dataSourceName string) (map[string]any, *apperr.AppError) {
	result["resolved_type"] = "database"
	result["database_id"] = databaseData["database_id"]
	result["database"] = cloneMap(databaseData)

	selectedDataSource, selectionReason, appErr := c.selectDatabaseDataSource(ctx, profile, databaseData, dataSourceName)
	if appErr != nil {
		return nil, appErr
	}
	if len(selectedDataSource) > 0 {
		result["data_source_id"] = selectedDataSource["data_source_id"]
		result["data_source"] = cloneMap(selectedDataSource)
		result["selected_data_source_reason"] = selectionReason
	}
	return result, nil
}

func (c *Client) attachResolvedDataSourceContext(ctx context.Context, profile ExecutionProfile, result map[string]any, dataSourceData map[string]any, dataSourceName string) (map[string]any, *apperr.AppError) {
	result["resolved_type"] = "data_source"
	result["data_source_id"] = dataSourceData["data_source_id"]
	result["data_source"] = cloneMap(dataSourceData)

	parent, _ := asMap(dataSourceData["parent"])
	if databaseID, ok := asString(parent["database_id"]); ok && strings.TrimSpace(databaseID) != "" {
		databaseData, appErr := c.GetDatabase(ctx, profile, map[string]any{
			"database_id": strings.TrimSpace(databaseID),
		})
		if appErr != nil {
			return nil, appErr
		}
		result["database_id"] = databaseData["database_id"]
		result["database"] = cloneMap(databaseData)

		// Keep the explicitly resolved data source as the selected one instead of replacing it with an inferred default.
		if dataSourceName != "" {
			result["selected_data_source_reason"] = "explicit_data_source"
		}
	}
	return result, nil
}

func (c *Client) attachResolvedPageContext(ctx context.Context, profile ExecutionProfile, result map[string]any, pageData map[string]any, dataSourceName string) (map[string]any, *apperr.AppError) {
	result["resolved_type"] = "page"
	result["page_id"] = pageData["page_id"]
	result["page"] = cloneMap(pageData)

	parent, _ := asMap(pageData["parent"])
	parentType, _ := asString(parent["type"])
	parentType = strings.TrimSpace(parentType)

	switch parentType {
	case "data_source_id":
		dataSourceID, _ := asString(parent["data_source_id"])
		dataSourceID = strings.TrimSpace(dataSourceID)
		if dataSourceID == "" {
			return result, nil
		}
		dataSourceData, appErr := c.GetDataSource(ctx, profile, map[string]any{
			"data_source_id": dataSourceID,
		})
		if appErr != nil {
			return nil, appErr
		}
		return c.attachResolvedDataSourceContext(ctx, profile, result, dataSourceData, dataSourceName)
	case "database_id":
		databaseID, _ := asString(parent["database_id"])
		databaseID = strings.TrimSpace(databaseID)
		if databaseID == "" {
			return result, nil
		}
		databaseData, appErr := c.GetDatabase(ctx, profile, map[string]any{
			"database_id": databaseID,
		})
		if appErr != nil {
			return nil, appErr
		}
		return c.attachResolvedDatabaseContext(ctx, profile, result, databaseData, dataSourceName)
	default:
		return result, nil
	}
}

// selectDatabaseDataSource picks one child data source when the choice is unambiguous or explicitly named.
// 解析 database 时，如果只有一个 data source，或用户显式给了名字，就顺手把 data source 也补出来。
func (c *Client) selectDatabaseDataSource(ctx context.Context, profile ExecutionProfile, databaseData map[string]any, dataSourceName string) (map[string]any, string, *apperr.AppError) {
	summaries, _ := databaseData["data_sources"].([]map[string]any)
	if len(summaries) == 0 {
		return nil, "", nil
	}

	switch {
	case len(summaries) == 1:
		dataSourceID, _ := asString(summaries[0]["data_source_id"])
		dataSourceData, appErr := c.GetDataSource(ctx, profile, map[string]any{
			"data_source_id": dataSourceID,
		})
		if appErr != nil {
			return nil, "", appErr
		}
		return dataSourceData, "single_child_data_source", nil
	case strings.TrimSpace(dataSourceName) != "":
		matches := make([]map[string]any, 0, len(summaries))
		for _, summary := range summaries {
			name, _ := asString(summary["name"])
			if strings.TrimSpace(name) == strings.TrimSpace(dataSourceName) {
				matches = append(matches, summary)
			}
		}
		switch len(matches) {
		case 0:
			return nil, "", apperr.New("OBJECT_NOT_FOUND", fmt.Sprintf("no child data source named %q was found under database %v", dataSourceName, databaseData["database_id"]))
		case 1:
			dataSourceID, _ := asString(matches[0]["data_source_id"])
			dataSourceData, appErr := c.GetDataSource(ctx, profile, map[string]any{
				"data_source_id": dataSourceID,
			})
			if appErr != nil {
				return nil, "", appErr
			}
			return dataSourceData, "matched_by_data_source_name", nil
		default:
			return nil, "", apperr.New("AMBIGUOUS_TARGET", fmt.Sprintf("multiple child data sources named %q were found under database %v", dataSourceName, databaseData["database_id"]))
		}
	default:
		return nil, "", nil
	}
}

func normalizeDatabaseTargetSelector(input map[string]any) (string, string, *apperr.AppError) {
	selectors := make([]struct {
		kind  string
		value string
	}, 0, 5)

	for _, field := range []string{"target", "url", "database_id", "data_source_id", "page_id"} {
		value, ok := asString(input[field])
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		selectors = append(selectors, struct {
			kind  string
			value string
		}{
			kind:  field,
			value: strings.TrimSpace(value),
		})
	}

	if len(selectors) != 1 {
		return "", "", apperr.New("INVALID_INPUT", "provide exactly one of target, url, database_id, data_source_id, or page_id")
	}
	return selectors[0].kind, selectors[0].value, nil
}

func isNotionResourceNotFound(appErr *apperr.AppError) bool {
	if appErr == nil {
		return false
	}
	return appErr.Code == "RESOURCE_NOT_FOUND" || appErr.Code == "OBJECT_NOT_FOUND"
}

func looksLikeURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	parsed, err := neturl.Parse(raw)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func extractNotionObjectID(raw string) (string, bool) {
	matches := notionObjectIDPattern.FindAllString(raw, -1)
	if len(matches) == 0 {
		return "", false
	}
	return normalizeNotionObjectID(matches[len(matches)-1])
}

func normalizeNotionObjectID(raw string) (string, bool) {
	compact := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), "-", ""))
	if len(compact) != 32 {
		return "", false
	}
	for _, ch := range compact {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return "", false
		}
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", compact[0:8], compact[8:12], compact[12:16], compact[16:20], compact[20:32]), true
}
