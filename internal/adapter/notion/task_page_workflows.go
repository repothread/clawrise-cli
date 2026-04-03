package notion

import (
	"context"
	"fmt"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// EnsurePageSections makes sure the requested heading sections exist and creates only the missing ones.
// 这个 task 只补缺失 section，不会覆盖已有 section 内容。
func (c *Client) EnsurePageSections(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	pageID, appErr := requireIDField(input, "page_id")
	if appErr != nil {
		return nil, appErr
	}

	sections, appErr := normalizeEnsureSectionsInput(input["sections"])
	if appErr != nil {
		return nil, appErr
	}
	markdownData, currentMarkdown, unknownBlockIDs, appErr := c.readTaskPageMarkdownWithSafety(ctx, profile, input, pageID)
	if appErr != nil {
		return nil, appErr
	}

	updatedMarkdown := currentMarkdown
	items := make([]map[string]any, 0, len(sections))
	createdCount := 0
	existingCount := 0
	for _, section := range sections {
		match, appErr := findMarkdownSectionMatch(updatedMarkdown, section.HeadingPath, section.HeadingLevel)
		if appErr != nil {
			return nil, appErr
		}
		if match != nil {
			existingCount++
			items = append(items, map[string]any{
				"heading_path":  section.HeadingPath,
				"heading_level": section.HeadingLevel,
				"action":        "exists",
			})
			continue
		}

		createdCount++
		updatedMarkdown = appendMissingMarkdownSection(updatedMarkdown, section.HeadingPath, section.HeadingLevel, section.Markdown)
		items = append(items, map[string]any{
			"heading_path":  section.HeadingPath,
			"heading_level": section.HeadingLevel,
			"action":        "created",
			"source":        section.Source,
		})
	}

	if updatedMarkdown == currentMarkdown {
		return map[string]any{
			"action":            "noop",
			"page_id":           pageID,
			"created_count":     createdCount,
			"existing_count":    existingCount,
			"unknown_block_ids": unknownBlockIDs,
			"markdown":          currentMarkdown,
			"page_markdown":     cloneMap(markdownData),
			"items":             items,
		}, nil
	}

	updatedPage, appErr := c.UpdatePageMarkdown(ctx, profile, map[string]any{
		"page_id": pageID,
		"type":    "replace_content",
		"replace_content": map[string]any{
			"new_str": updatedMarkdown,
		},
	})
	if appErr != nil {
		return nil, appErr
	}

	return map[string]any{
		"action":            "updated",
		"page_id":           pageID,
		"created_count":     createdCount,
		"existing_count":    existingCount,
		"unknown_block_ids": unknownBlockIDs,
		"markdown_page":     cloneMap(updatedPage),
		"items":             items,
	}, nil
}

// AppendUnderHeading appends markdown content under one existing heading section or creates that section first when requested.
// 追加而不是替换，更适合 AI 累积日志、结论、行动项。
func (c *Client) AppendUnderHeading(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	pageID, appErr := requireIDField(input, "page_id")
	if appErr != nil {
		return nil, appErr
	}

	headingPath, headingLevel, appErr := normalizePatchSectionTarget(input)
	if appErr != nil {
		return nil, appErr
	}
	appendBody, source, appErr := resolveMarkdownTaskSource(input)
	if appErr != nil {
		return nil, appErr
	}

	_, currentMarkdown, unknownBlockIDs, appErr := c.readTaskPageMarkdownWithSafety(ctx, profile, input, pageID)
	if appErr != nil {
		return nil, appErr
	}

	match, appErr := findMarkdownSectionMatch(currentMarkdown, headingPath, headingLevel)
	if appErr != nil {
		return nil, appErr
	}

	action := "appended"
	updatedMarkdown := currentMarkdown
	if match == nil {
		createIfMissing := false
		if value, ok := asBool(input["create_if_missing"]); ok {
			createIfMissing = value
		}
		if !createIfMissing {
			return nil, apperr.New("OBJECT_NOT_FOUND", fmt.Sprintf("no section matched %q", strings.Join(headingPath, " / ")))
		}
		action = "created_and_appended"
		updatedMarkdown = appendMissingMarkdownSection(currentMarkdown, headingPath, headingLevel, appendBody)
	} else {
		updatedMarkdown = appendMarkdownUnderSection(currentMarkdown, *match, appendBody)
	}

	if updatedMarkdown == currentMarkdown {
		return map[string]any{
			"action":            "noop",
			"page_id":           pageID,
			"heading_path":      headingPath,
			"heading_level":     headingLevel,
			"unknown_block_ids": unknownBlockIDs,
			"source":            source,
		}, nil
	}

	updatedPage, appErr := c.UpdatePageMarkdown(ctx, profile, map[string]any{
		"page_id": pageID,
		"type":    "replace_content",
		"replace_content": map[string]any{
			"new_str": updatedMarkdown,
		},
	})
	if appErr != nil {
		return nil, appErr
	}

	return map[string]any{
		"action":            action,
		"page_id":           pageID,
		"heading_path":      headingPath,
		"heading_level":     headingLevel,
		"unknown_block_ids": unknownBlockIDs,
		"source":            source,
		"markdown_page":     cloneMap(updatedPage),
	}, nil
}

// FindOrCreatePageByPath resolves one root context and then walks the requested page path, creating missing pages along the way.
// 这相当于给 AI 一个“按人类路径找页/建页”的稳定原语。
func (c *Client) FindOrCreatePageByPath(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	segments, appErr := normalizePagePathSegments(input)
	if appErr != nil {
		return nil, appErr
	}

	rootContext, appErr := c.resolveFindOrCreatePathRoot(ctx, profile, input)
	if appErr != nil {
		return nil, appErr
	}

	markdown, source, hasMarkdown, appErr := resolveOptionalMarkdownTaskSource(input)
	if appErr != nil {
		return nil, appErr
	}

	searchPageSize := 100
	if value, ok := asInt(input["search_page_size"]); ok && value > 0 {
		searchPageSize = value
	}
	createIfMissing := true
	if value, ok := asBool(input["create_if_missing"]); ok {
		createIfMissing = value
	}

	currentParentType := rootContext.ParentType
	currentParentID := rootContext.ParentID
	currentTitleProperty := rootContext.TitleProperty
	items := make([]map[string]any, 0, len(segments))
	createdCount := 0
	foundCount := 0
	leafPage := map[string]any{}

	for index, segment := range segments {
		match, appErr := c.findChildPageByTitle(ctx, profile, currentParentType, currentParentID, currentTitleProperty, segment, searchPageSize)
		if appErr != nil {
			return nil, appErr
		}

		if len(match) == 0 {
			if !createIfMissing {
				return nil, apperr.New("OBJECT_NOT_FOUND", fmt.Sprintf("page path segment %q was not found under %s %s", segment, currentParentType, currentParentID))
			}

			createInput := map[string]any{
				"parent": map[string]any{
					"type": currentParentType,
					"id":   currentParentID,
				},
				"title": segment,
			}
			if currentParentType == "data_source_id" && currentTitleProperty != "" {
				createInput["title_property"] = currentTitleProperty
			}
			if index == len(segments)-1 && hasMarkdown {
				createInput["markdown"] = markdown
			}
			if currentParentType == "page_id" {
				copyOptionalTaskFields(input, createInput, "position", "after", "template")
			}

			createdPage, appErr := c.CreatePage(ctx, profile, createInput)
			if appErr != nil {
				return nil, appErr
			}
			leafPage = createdPage
			createdCount++
			items = append(items, map[string]any{
				"index":   index,
				"title":   segment,
				"action":  "created",
				"page_id": createdPage["page_id"],
			})
		} else {
			leafPage = cloneMap(match[0])
			foundCount++
			items = append(items, map[string]any{
				"index":   index,
				"title":   segment,
				"action":  "found",
				"page_id": leafPage["id"],
			})
		}

		nextPageID, _ := asString(firstNonEmptyValue(leafPage["page_id"], leafPage["id"]))
		currentParentType = "page_id"
		currentParentID = strings.TrimSpace(nextPageID)
		currentTitleProperty = ""
	}

	result := map[string]any{
		"action":         "resolved",
		"leaf_page_id":   currentParentID,
		"created_count":  createdCount,
		"found_count":    foundCount,
		"path":           segments,
		"root_type":      rootContext.RootType,
		"root_parent_id": rootContext.ParentID,
		"items":          items,
		"page":           cloneMap(leafPage),
	}
	if hasMarkdown {
		result["source"] = source
	}
	return result, nil
}

type ensureSectionInput struct {
	HeadingPath  []string
	HeadingLevel int
	Markdown     string
	Source       string
}

type findOrCreatePathRoot struct {
	RootType      string
	ParentType    string
	ParentID      string
	TitleProperty string
}

func normalizeEnsureSectionsInput(raw any) ([]ensureSectionInput, *apperr.AppError) {
	items, ok := asArray(raw)
	if !ok || len(items) == 0 {
		return nil, apperr.New("INVALID_INPUT", "sections must be a non-empty array")
	}

	sections := make([]ensureSectionInput, 0, len(items))
	for index, item := range items {
		record, ok := asMap(item)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", fmt.Sprintf("sections[%d] must be an object", index))
		}

		headingPath, headingLevel, appErr := normalizePatchSectionTarget(record)
		if appErr != nil {
			return nil, appErr
		}
		markdown, source, hasMarkdown, appErr := resolveOptionalMarkdownTaskSource(record)
		if appErr != nil {
			return nil, appErr
		}
		section := ensureSectionInput{
			HeadingPath:  headingPath,
			HeadingLevel: headingLevel,
		}
		if hasMarkdown {
			section.Markdown = markdown
			section.Source = source
		}
		sections = append(sections, section)
	}
	return sections, nil
}

func (c *Client) readTaskPageMarkdownWithSafety(ctx context.Context, profile ExecutionProfile, input map[string]any, pageID string) (map[string]any, string, []string, *apperr.AppError) {
	markdownData, appErr := c.GetPageMarkdown(ctx, profile, map[string]any{
		"page_id": pageID,
	})
	if appErr != nil {
		return nil, "", nil, appErr
	}

	if truncated, _ := asBool(markdownData["truncated"]); truncated {
		if allowed, _ := asBool(input["allow_truncated"]); !allowed {
			return nil, "", nil, apperr.New("UNSAFE_PAGE_CONTENT", "page markdown is truncated; set allow_truncated=true to continue")
		}
	}

	unknownBlockIDs := toStringSlice(markdownData["unknown_block_ids"])
	if len(unknownBlockIDs) > 0 {
		if allowed, _ := asBool(input["allow_unknown_blocks"]); !allowed {
			return nil, "", nil, apperr.New("UNSAFE_PAGE_CONTENT", "page markdown contains unknown_block_ids; set allow_unknown_blocks=true to continue")
		}
	}

	currentMarkdown, _ := asString(markdownData["markdown"])
	return markdownData, currentMarkdown, unknownBlockIDs, nil
}

func appendMarkdownUnderSection(markdown string, match markdownSectionMatch, newBody string) string {
	existingBody := extractMarkdownSectionBody(markdown, match)
	appendedBody := strings.Trim(newBody, "\n")
	trimmedExistingBody := strings.Trim(existingBody, "\n")
	if strings.TrimSpace(trimmedExistingBody) != "" && strings.TrimSpace(appendedBody) != "" {
		appendedBody = trimmedExistingBody + "\n\n" + appendedBody
	} else if strings.TrimSpace(trimmedExistingBody) != "" {
		appendedBody = trimmedExistingBody
	}
	return replaceMarkdownSectionBody(markdown, match, appendedBody)
}

func extractMarkdownSectionBody(markdown string, match markdownSectionMatch) string {
	lines := splitMarkdownLines(markdown)
	if match.Heading.LineIndex+1 >= len(lines) || match.EndLine > len(lines) {
		return ""
	}
	bodyLines := trimLeadingBlankLines(lines[match.Heading.LineIndex+1 : match.EndLine])
	bodyLines = trimTrailingBlankLines(bodyLines)
	return strings.Join(bodyLines, "\n")
}

func normalizePagePathSegments(input map[string]any) ([]string, *apperr.AppError) {
	if rawPath, exists := input["path"]; exists {
		switch value := rawPath.(type) {
		case string:
			return splitPagePathString(value)
		default:
			items, ok := asArray(value)
			if !ok || len(items) == 0 {
				return nil, apperr.New("INVALID_INPUT", "path must be a non-empty string or array")
			}
			segments := make([]string, 0, len(items))
			for _, item := range items {
				segment, ok := asString(item)
				if !ok || strings.TrimSpace(segment) == "" {
					return nil, apperr.New("INVALID_INPUT", "each path segment must be a non-empty string")
				}
				segments = append(segments, strings.TrimSpace(segment))
			}
			return segments, nil
		}
	}
	if rawPath, ok := asString(input["path_string"]); ok && strings.TrimSpace(rawPath) != "" {
		return splitPagePathString(rawPath)
	}
	return nil, apperr.New("INVALID_INPUT", "path is required")
}

func splitPagePathString(raw string) ([]string, *apperr.AppError) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, apperr.New("INVALID_INPUT", "path must not be empty")
	}
	raw = strings.ReplaceAll(raw, ">", "/")
	parts := strings.Split(raw, "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segments = append(segments, part)
	}
	if len(segments) == 0 {
		return nil, apperr.New("INVALID_INPUT", "path must contain at least one non-empty segment")
	}
	return segments, nil
}

func (c *Client) resolveFindOrCreatePathRoot(ctx context.Context, profile ExecutionProfile, input map[string]any) (findOrCreatePathRoot, *apperr.AppError) {
	if parentPageID, ok := asString(input["parent_page_id"]); ok && strings.TrimSpace(parentPageID) != "" {
		return findOrCreatePathRoot{
			RootType:   "page",
			ParentType: "page_id",
			ParentID:   strings.TrimSpace(parentPageID),
		}, nil
	}

	resolved, appErr := c.ResolveDatabaseTarget(ctx, profile, input)
	if appErr != nil {
		return findOrCreatePathRoot{}, appErr
	}
	resolvedType, _ := asString(resolved["resolved_type"])
	switch strings.TrimSpace(resolvedType) {
	case "page":
		pageID, _ := asString(resolved["page_id"])
		return findOrCreatePathRoot{
			RootType:   "page",
			ParentType: "page_id",
			ParentID:   strings.TrimSpace(pageID),
		}, nil
	case "data_source":
		dataSource, _ := asMap(resolved["data_source"])
		properties, _ := asMap(dataSource["properties"])
		titleProperty := findDataSourceTitlePropertyName(properties)
		if titleProperty == "" {
			return findOrCreatePathRoot{}, apperr.New("SCHEMA_CONFLICT", "failed to infer title property name from target data source")
		}
		dataSourceID, _ := asString(resolved["data_source_id"])
		return findOrCreatePathRoot{
			RootType:      "data_source",
			ParentType:    "data_source_id",
			ParentID:      strings.TrimSpace(dataSourceID),
			TitleProperty: titleProperty,
		}, nil
	case "database":
		if dataSourceID, ok := asString(resolved["data_source_id"]); ok && strings.TrimSpace(dataSourceID) != "" {
			dataSource, _ := asMap(resolved["data_source"])
			properties, _ := asMap(dataSource["properties"])
			titleProperty := findDataSourceTitlePropertyName(properties)
			if titleProperty == "" {
				return findOrCreatePathRoot{}, apperr.New("SCHEMA_CONFLICT", "failed to infer title property name from resolved database data source")
			}
			return findOrCreatePathRoot{
				RootType:      "database",
				ParentType:    "data_source_id",
				ParentID:      strings.TrimSpace(dataSourceID),
				TitleProperty: titleProperty,
			}, nil
		}
		return findOrCreatePathRoot{}, apperr.New("OBJECT_NOT_FOUND", "resolved database does not have one usable data source; provide data_source_name to disambiguate")
	default:
		return findOrCreatePathRoot{}, apperr.New("INVALID_INPUT", "failed to resolve one page or data source root")
	}
}

func (c *Client) findChildPageByTitle(ctx context.Context, profile ExecutionProfile, parentType string, parentID string, titleProperty string, title string, pageSize int) ([]map[string]any, *apperr.AppError) {
	switch parentType {
	case "page_id":
		searchData, appErr := c.Search(ctx, profile, map[string]any{
			"query": title,
			"filter": map[string]any{
				"property": "object",
				"value":    "page",
			},
			"page_size": pageSize,
		})
		if appErr != nil {
			return nil, appErr
		}
		return findExactChildPageMatchesByParent(searchData["items"], "page_id", parentID, title), nil
	case "data_source_id":
		if strings.TrimSpace(titleProperty) == "" {
			return nil, apperr.New("SCHEMA_CONFLICT", "title_property is required when finding pages under a data source")
		}
		queryData, appErr := c.QueryDataSource(ctx, profile, map[string]any{
			"data_source_id": parentID,
			"page_size":      pageSize,
			"filter": map[string]any{
				"property": titleProperty,
				"title": map[string]any{
					"equals": title,
				},
			},
		})
		if appErr != nil {
			return nil, appErr
		}
		return findExactChildPageMatchesByParent(queryData["items"], "data_source_id", parentID, title), nil
	default:
		return nil, apperr.New("INVALID_INPUT", fmt.Sprintf("unsupported path parent type %s", parentType))
	}
}

func findExactChildPageMatchesByParent(raw any, parentType string, parentID string, title string) []map[string]any {
	items, ok := raw.([]map[string]any)
	if !ok || len(items) == 0 {
		return nil
	}

	expectedParentID := strings.TrimSpace(parentID)
	expectedTitle := strings.TrimSpace(title)
	seenPageIDs := map[string]struct{}{}
	matches := make([]map[string]any, 0, len(items))
	for _, item := range items {
		objectType, _ := asString(item["object"])
		if strings.TrimSpace(objectType) != "page" {
			continue
		}
		itemTitle, _ := asString(item["title"])
		if strings.TrimSpace(itemTitle) != expectedTitle {
			continue
		}
		if archived, ok := asBool(item["archived"]); ok && archived {
			continue
		}
		if inTrash, ok := asBool(item["in_trash"]); ok && inTrash {
			continue
		}

		parent, ok := asMap(item["parent"])
		if !ok {
			continue
		}
		parentMatchID, _ := asString(parent[parentType])
		if strings.TrimSpace(parentMatchID) != expectedParentID {
			continue
		}

		pageID, _ := asString(item["id"])
		pageID = strings.TrimSpace(pageID)
		if pageID == "" {
			continue
		}
		if _, exists := seenPageIDs[pageID]; exists {
			continue
		}
		seenPageIDs[pageID] = struct{}{}
		matches = append(matches, cloneMap(item))
	}
	return matches
}

func firstNonEmptyValue(values ...any) string {
	for _, value := range values {
		if text, ok := asString(value); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}
