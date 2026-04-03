package notion

import (
	"context"
	"fmt"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

type markdownHeading struct {
	LineIndex int
	Level     int
	Title     string
	Path      []string
}

type markdownSectionMatch struct {
	Heading markdownHeading
	EndLine int
}

// PatchPageSection replaces one markdown section by exact heading or heading path.
// 这个 task 先安全读取整页 markdown，再只替换目标 section，避免 AI 误覆盖整页其它内容。
func (c *Client) PatchPageSection(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	pageID, appErr := requireIDField(input, "page_id")
	if appErr != nil {
		return nil, appErr
	}

	headingPath, headingLevel, appErr := normalizePatchSectionTarget(input)
	if appErr != nil {
		return nil, appErr
	}

	sectionBody, source, appErr := resolveMarkdownTaskSource(input)
	if appErr != nil {
		return nil, appErr
	}

	markdownData, appErr := c.GetPageMarkdown(ctx, profile, map[string]any{
		"page_id": pageID,
	})
	if appErr != nil {
		return nil, appErr
	}

	if truncated, _ := asBool(markdownData["truncated"]); truncated {
		if allowed, _ := asBool(input["allow_truncated"]); !allowed {
			return nil, apperr.New("UNSAFE_PAGE_CONTENT", "page markdown is truncated; set allow_truncated=true to patch it anyway")
		}
	}

	unknownBlockIDs := toStringSlice(markdownData["unknown_block_ids"])
	if len(unknownBlockIDs) > 0 {
		if allowed, _ := asBool(input["allow_unknown_blocks"]); !allowed {
			return nil, apperr.New("UNSAFE_PAGE_CONTENT", "page markdown contains unknown_block_ids; set allow_unknown_blocks=true to patch it anyway")
		}
	}

	currentMarkdown, _ := asString(markdownData["markdown"])
	match, appErr := findMarkdownSectionMatch(currentMarkdown, headingPath, headingLevel)
	if appErr != nil {
		return nil, appErr
	}

	action := "updated"
	updatedMarkdown := currentMarkdown
	if match == nil {
		createIfMissing := false
		if value, ok := asBool(input["create_if_missing"]); ok {
			createIfMissing = value
		}
		if !createIfMissing {
			return nil, apperr.New("OBJECT_NOT_FOUND", fmt.Sprintf("no section matched %q", strings.Join(headingPath, " / ")))
		}

		action = "created"
		updatedMarkdown = appendMissingMarkdownSection(currentMarkdown, headingPath, headingLevel, sectionBody)
	} else {
		updatedMarkdown = replaceMarkdownSectionBody(currentMarkdown, *match, sectionBody)
	}

	if updatedMarkdown == currentMarkdown {
		return map[string]any{
			"action":        "noop",
			"page_id":       pageID,
			"heading_path":  headingPath,
			"heading_level": headingLevel,
			"source":        source,
		}, nil
	}

	updateData, appErr := c.UpdatePageMarkdown(ctx, profile, map[string]any{
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
		"markdown_page":     cloneMap(updateData),
	}, nil
}

func normalizePatchSectionTarget(input map[string]any) ([]string, int, *apperr.AppError) {
	heading, hasHeading := asString(input["heading"])
	rawHeadingPath, hasHeadingPath := input["heading_path"]
	if hasHeading == hasHeadingPath {
		return nil, 0, apperr.New("INVALID_INPUT", "provide exactly one of heading or heading_path")
	}

	var headingPath []string
	if hasHeading {
		heading = strings.TrimSpace(heading)
		if heading == "" {
			return nil, 0, apperr.New("INVALID_INPUT", "heading must be a non-empty string")
		}
		headingPath = []string{heading}
	} else {
		items, ok := asArray(rawHeadingPath)
		if !ok || len(items) == 0 {
			return nil, 0, apperr.New("INVALID_INPUT", "heading_path must be a non-empty array")
		}
		headingPath = make([]string, 0, len(items))
		for _, item := range items {
			text, ok := asString(item)
			if !ok || strings.TrimSpace(text) == "" {
				return nil, 0, apperr.New("INVALID_INPUT", "each heading_path item must be a non-empty string")
			}
			headingPath = append(headingPath, strings.TrimSpace(text))
		}
	}

	headingLevel := 0
	if value, ok := asInt(input["heading_level"]); ok {
		if value < 1 || value > 6 {
			return nil, 0, apperr.New("INVALID_INPUT", "heading_level must be between 1 and 6")
		}
		headingLevel = value
	}

	return headingPath, headingLevel, nil
}

func findMarkdownSectionMatch(markdown string, headingPath []string, headingLevel int) (*markdownSectionMatch, *apperr.AppError) {
	lines, headings := parseMarkdownHeadings(markdown)
	matches := make([]markdownSectionMatch, 0)
	for index, heading := range headings {
		if !headingMatchesTarget(heading, headingPath, headingLevel) {
			continue
		}

		endLine := len(lines)
		for nextIndex := index + 1; nextIndex < len(headings); nextIndex++ {
			if headings[nextIndex].Level <= heading.Level {
				endLine = headings[nextIndex].LineIndex
				break
			}
		}
		matches = append(matches, markdownSectionMatch{
			Heading: heading,
			EndLine: endLine,
		})
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return &matches[0], nil
	default:
		descriptions := make([]string, 0, len(matches))
		for _, match := range matches {
			descriptions = append(descriptions, fmt.Sprintf("line %d (%s)", match.Heading.LineIndex+1, strings.Join(match.Heading.Path, " / ")))
		}
		return nil, apperr.New("AMBIGUOUS_TARGET", fmt.Sprintf("found %d matching sections for %q: %s", len(matches), strings.Join(headingPath, " / "), strings.Join(descriptions, ", ")))
	}
}

func headingMatchesTarget(heading markdownHeading, headingPath []string, headingLevel int) bool {
	if len(headingPath) == 1 {
		if strings.TrimSpace(heading.Title) != strings.TrimSpace(headingPath[0]) {
			return false
		}
		return headingLevel == 0 || heading.Level == headingLevel
	}

	if len(heading.Path) != len(headingPath) {
		return false
	}
	for index := range headingPath {
		if strings.TrimSpace(heading.Path[index]) != strings.TrimSpace(headingPath[index]) {
			return false
		}
	}
	if headingLevel == 0 {
		return true
	}
	expectedLevel := headingLevel + len(headingPath) - 1
	if expectedLevel > 6 {
		expectedLevel = 6
	}
	return heading.Level == expectedLevel
}

func parseMarkdownHeadings(markdown string) ([]string, []markdownHeading) {
	lines := splitMarkdownLines(markdown)
	headings := make([]markdownHeading, 0)
	path := make([]string, 0)

	inFence := false
	fenceMarker := ""
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if marker, ok := markdownFenceMarker(trimmed); ok {
			if !inFence {
				inFence = true
				fenceMarker = marker
			} else if marker == fenceMarker {
				inFence = false
				fenceMarker = ""
			}
			continue
		}
		if inFence {
			continue
		}

		level, title, ok := parseMarkdownHeadingLine(trimmed)
		if !ok {
			continue
		}
		if len(path) >= level {
			path = path[:level-1]
		}
		path = append(path, title)

		headings = append(headings, markdownHeading{
			LineIndex: index,
			Level:     level,
			Title:     title,
			Path:      append([]string{}, path...),
		})
	}
	return lines, headings
}

func markdownFenceMarker(line string) (string, bool) {
	switch {
	case strings.HasPrefix(line, "```"):
		return "```", true
	case strings.HasPrefix(line, "~~~"):
		return "~~~", true
	default:
		return "", false
	}
}

func parseMarkdownHeadingLine(line string) (int, string, bool) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false
	}

	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0, "", false
	}
	if level >= len(line) || (line[level] != ' ' && line[level] != '\t') {
		return 0, "", false
	}

	title := strings.TrimSpace(line[level:])
	title = strings.TrimSpace(strings.TrimRight(title, "#"))
	if title == "" {
		return 0, "", false
	}
	return level, title, true
}

func replaceMarkdownSectionBody(markdown string, match markdownSectionMatch, newBody string) string {
	lines := splitMarkdownLines(markdown)
	if len(lines) == 0 {
		return appendMissingMarkdownSection("", match.Heading.Path, match.Heading.Level, newBody)
	}

	result := make([]string, 0, len(lines))
	result = append(result, lines[:match.Heading.LineIndex+1]...)

	bodyLines := splitMarkdownBodyLines(newBody)
	suffix := trimLeadingBlankLines(lines[match.EndLine:])
	if len(bodyLines) > 0 {
		result = append(result, "")
		result = append(result, bodyLines...)
	}
	if len(suffix) > 0 {
		if len(result) > 0 && result[len(result)-1] != "" {
			result = append(result, "")
		}
		result = append(result, suffix...)
	}
	return strings.Join(trimTrailingBlankLines(result), "\n")
}

func appendMissingMarkdownSection(markdown string, headingPath []string, headingLevel int, body string) string {
	existing := strings.TrimRight(markdown, "\n")
	appended := buildMissingSectionBlock(headingPath, headingLevel, body)
	if strings.TrimSpace(existing) == "" {
		return appended
	}
	if strings.TrimSpace(appended) == "" {
		return existing
	}
	return existing + "\n\n" + appended
}

func buildMissingSectionBlock(headingPath []string, headingLevel int, body string) string {
	if headingLevel == 0 {
		headingLevel = 2
	}

	lines := make([]string, 0, len(headingPath)*2+4)
	for index, title := range headingPath {
		level := headingLevel + index
		if level > 6 {
			level = 6
		}
		lines = append(lines, strings.Repeat("#", level)+" "+strings.TrimSpace(title))
		lines = append(lines, "")
	}

	bodyLines := splitMarkdownBodyLines(body)
	lines = append(lines, bodyLines...)
	return strings.Join(trimTrailingBlankLines(lines), "\n")
}

func splitMarkdownLines(markdown string) []string {
	if markdown == "" {
		return []string{}
	}
	return strings.Split(markdown, "\n")
}

func splitMarkdownBodyLines(markdown string) []string {
	trimmed := strings.Trim(markdown, "\n")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "\n")
}

func trimLeadingBlankLines(lines []string) []string {
	index := 0
	for index < len(lines) && strings.TrimSpace(lines[index]) == "" {
		index++
	}
	return lines[index:]
}

func trimTrailingBlankLines(lines []string) []string {
	index := len(lines)
	for index > 0 && strings.TrimSpace(lines[index-1]) == "" {
		index--
	}
	return lines[:index]
}
