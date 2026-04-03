package notion

import (
	"context"
	"fmt"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// GetMeetingNotes 读取一个 meeting_notes/transcription 块，或在页面下发现所有 meeting_notes/transcription 块并补拉它们的分区内容。
// GetMeetingNotes reads one meeting_notes/transcription block, or discovers all meeting_notes/transcription blocks under one page and then fetches their related section content.
func (c *Client) GetMeetingNotes(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	_, hasBlockID := input["block_id"]
	_, hasPageID := input["page_id"]
	if hasBlockID == hasPageID {
		return nil, apperr.New("INVALID_INPUT", "provide exactly one of block_id or page_id")
	}

	includeSummary := true
	if value, ok := asBool(input["include_summary"]); ok {
		includeSummary = value
	}
	includeNotes := true
	if value, ok := asBool(input["include_notes"]); ok {
		includeNotes = value
	}
	includeTranscript := true
	if value, ok := asBool(input["include_transcript"]); ok {
		includeTranscript = value
	}

	items := make([]map[string]any, 0)
	switch {
	case hasBlockID:
		blockID, appErr := requireIDField(input, "block_id")
		if appErr != nil {
			return nil, appErr
		}
		blockData, appErr := c.GetBlock(ctx, profile, map[string]any{
			"block_id": blockID,
		})
		if appErr != nil {
			return nil, appErr
		}
		item, appErr := c.buildMeetingNotesItem(ctx, profile, blockData, includeSummary, includeNotes, includeTranscript)
		if appErr != nil {
			return nil, appErr
		}
		items = append(items, item)
	case hasPageID:
		pageID, appErr := requireIDField(input, "page_id")
		if appErr != nil {
			return nil, appErr
		}
		descendantsData, appErr := c.GetBlockDescendants(ctx, profile, map[string]any{
			"block_id": pageID,
		})
		if appErr != nil {
			return nil, appErr
		}
		descendants, _ := descendantsData["items"].([]map[string]any)
		for _, item := range descendants {
			if !isMeetingNotesBlock(item) {
				continue
			}
			meetingNotesItem, appErr := c.buildMeetingNotesItem(ctx, profile, item, includeSummary, includeNotes, includeTranscript)
			if appErr != nil {
				return nil, appErr
			}
			items = append(items, meetingNotesItem)
		}
		if len(items) == 0 {
			return nil, apperr.New("OBJECT_NOT_FOUND", fmt.Sprintf("no meeting_notes or transcription blocks were found under page %s", pageID))
		}
	}

	return map[string]any{
		"count":              len(items),
		"include_summary":    includeSummary,
		"include_notes":      includeNotes,
		"include_transcript": includeTranscript,
		"items":              items,
	}, nil
}

// buildMeetingNotesItem 解析 meeting_notes/transcription 元数据，并按需补拉 summary、notes、transcript 分区。
// buildMeetingNotesItem parses meeting_notes/transcription metadata and fetches summary, notes, and transcript sections on demand.
func (c *Client) buildMeetingNotesItem(ctx context.Context, profile ExecutionProfile, blockData map[string]any, includeSummary bool, includeNotes bool, includeTranscript bool) (map[string]any, *apperr.AppError) {
	rawBlock, _ := asMap(blockData["raw"])
	body, normalizedType, ok := extractMeetingNotesBody(rawBlock)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "block is not a meeting_notes or transcription block")
	}

	result := map[string]any{
		"block":          cloneMap(blockData),
		"type":           normalizedType,
		"title":          extractMeetingNotesTitle(body),
		"status":         extractMeetingNotesStatus(body),
		"calendar_event": cloneMeetingNotesSubObject(body, "calendar_event"),
		"recording":      cloneMeetingNotesSubObject(body, "recording"),
		"children":       cloneMeetingNotesSubObject(body, "children"),
	}

	children, _ := asMap(body["children"])
	if includeSummary {
		section, appErr := c.readMeetingNotesSection(ctx, profile, extractMeetingNotesChildID(children, "summary_block_id"))
		if appErr != nil {
			return nil, appErr
		}
		result["summary"] = section
	}
	if includeNotes {
		section, appErr := c.readMeetingNotesSection(ctx, profile, extractMeetingNotesChildID(children, "notes_block_id"))
		if appErr != nil {
			return nil, appErr
		}
		result["notes"] = section
	}
	if includeTranscript {
		section, appErr := c.readMeetingNotesSection(ctx, profile, extractMeetingNotesChildID(children, "transcript_block_id"))
		if appErr != nil {
			return nil, appErr
		}
		result["transcript"] = section
	}

	return result, nil
}

// readMeetingNotesSection 读取一个 section 根块及其后代，并聚合一份 plain_text 文本。
// readMeetingNotesSection reads one section root block plus its descendants and builds one aggregated plain_text field.
func (c *Client) readMeetingNotesSection(ctx context.Context, profile ExecutionProfile, blockID string) (map[string]any, *apperr.AppError) {
	blockID = strings.TrimSpace(blockID)
	if blockID == "" {
		return nil, nil
	}

	rootBlock, appErr := c.GetBlock(ctx, profile, map[string]any{
		"block_id": blockID,
	})
	if appErr != nil {
		return nil, appErr
	}

	descendants := make([]map[string]any, 0)
	if hasChildren, ok := asBool(rootBlock["has_children"]); ok && hasChildren {
		descendantsData, appErr := c.GetBlockDescendants(ctx, profile, map[string]any{
			"block_id": blockID,
		})
		if appErr != nil {
			return nil, appErr
		}
		descendants, _ = descendantsData["items"].([]map[string]any)
	}

	plainTextParts := make([]string, 0, len(descendants)+1)
	if plainText, ok := asString(rootBlock["plain_text"]); ok && strings.TrimSpace(plainText) != "" {
		plainTextParts = append(plainTextParts, strings.TrimSpace(plainText))
	}
	for _, item := range descendants {
		plainText, _ := asString(item["plain_text"])
		if strings.TrimSpace(plainText) != "" {
			plainTextParts = append(plainTextParts, strings.TrimSpace(plainText))
		}
	}

	return map[string]any{
		"block_id":    blockID,
		"block":       cloneMap(rootBlock),
		"descendants": descendants,
		"plain_text":  strings.Join(plainTextParts, "\n"),
	}, nil
}

func isMeetingNotesBlock(blockData map[string]any) bool {
	blockType, _ := asString(blockData["type"])
	blockType = strings.TrimSpace(blockType)
	return blockType == "meeting_notes" || blockType == "transcription"
}

func extractMeetingNotesBody(rawBlock map[string]any) (map[string]any, string, bool) {
	if len(rawBlock) == 0 {
		return nil, "", false
	}
	for _, blockType := range []string{"meeting_notes", "transcription"} {
		body, ok := asMap(rawBlock[blockType])
		if !ok || len(body) == 0 {
			continue
		}
		return body, "meeting_notes", true
	}
	return nil, "", false
}

func extractMeetingNotesTitle(body map[string]any) string {
	titleItems, _ := asArray(body["title"])
	return extractRichTextPlainText(titleItems)
}

func extractMeetingNotesStatus(body map[string]any) string {
	status, _ := asString(body["status"])
	return strings.TrimSpace(status)
}

func cloneMeetingNotesSubObject(body map[string]any, key string) map[string]any {
	record, ok := asMap(body[key])
	if !ok || len(record) == 0 {
		return nil
	}
	return cloneMap(record)
}

func extractMeetingNotesChildID(children map[string]any, key string) string {
	value, _ := asString(children[key])
	return strings.TrimSpace(value)
}
