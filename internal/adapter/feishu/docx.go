package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// GetDocument fetches basic document metadata.
func (c *Client) GetDocument(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	documentID, ok := asString(input["document_id"])
	if !ok || strings.TrimSpace(documentID) == "" {
		return nil, apperr.New("INVALID_INPUT", "document_id is required")
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/docx/v1/documents/"+url.PathEscape(strings.TrimSpace(documentID)),
		nil,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response documentGetResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode document get response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}
	if response.Data.Document.DocumentID == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "document_id is empty in Feishu response")
	}

	return map[string]any{
		"document_id": response.Data.Document.DocumentID,
		"revision_id": response.Data.Document.RevisionID,
		"title":       response.Data.Document.Title,
		"raw": map[string]any{
			"display_setting": response.Data.Document.DisplaySetting,
			"cover":           response.Data.Document.Cover,
		},
	}, nil
}

// ListDocumentBlocks returns document blocks with pagination support.
func (c *Client) ListDocumentBlocks(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	documentID, ok := asString(input["document_id"])
	if !ok || strings.TrimSpace(documentID) == "" {
		return nil, apperr.New("INVALID_INPUT", "document_id is required")
	}

	query := url.Values{}
	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		query.Set("page_token", strings.TrimSpace(pageToken))
	}
	if revisionID, ok := asInt(input["document_revision_id"]); ok {
		query.Set("document_revision_id", strconv.Itoa(revisionID))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/docx/v1/documents/"+url.PathEscape(strings.TrimSpace(documentID))+"/blocks",
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response documentBlockListResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode document block list response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}

	items := make([]map[string]any, 0, len(response.Data.Items))
	for _, item := range response.Data.Items {
		items = append(items, normalizeDocxBlock(item))
	}

	return map[string]any{
		"document_id":     strings.TrimSpace(documentID),
		"items":           items,
		"next_page_token": response.Data.PageToken,
		"has_more":        response.Data.HasMore,
	}, nil
}

// GetDocumentBlock fetches the structured content of a single block.
func (c *Client) GetDocumentBlock(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	documentID, ok := asString(input["document_id"])
	if !ok || strings.TrimSpace(documentID) == "" {
		return nil, apperr.New("INVALID_INPUT", "document_id is required")
	}
	blockID, ok := asString(input["block_id"])
	if !ok || strings.TrimSpace(blockID) == "" {
		return nil, apperr.New("INVALID_INPUT", "block_id is required")
	}

	query := url.Values{}
	if revisionID, ok := asInt(input["document_revision_id"]); ok {
		query.Set("document_revision_id", strconv.Itoa(revisionID))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/docx/v1/documents/"+url.PathEscape(strings.TrimSpace(documentID))+"/blocks/"+url.PathEscape(strings.TrimSpace(blockID)),
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response documentBlockGetResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode document block get response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}
	if response.Data.Block.BlockID == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "block_id is empty in Feishu response")
	}

	return normalizeDocxBlock(response.Data.Block), nil
}

// GetDocumentBlockChildren returns child blocks under the given block with pagination support.
func (c *Client) GetDocumentBlockChildren(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	documentID, ok := asString(input["document_id"])
	if !ok || strings.TrimSpace(documentID) == "" {
		return nil, apperr.New("INVALID_INPUT", "document_id is required")
	}
	blockID, ok := asString(input["block_id"])
	if !ok || strings.TrimSpace(blockID) == "" {
		return nil, apperr.New("INVALID_INPUT", "block_id is required")
	}

	query := url.Values{}
	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		query.Set("page_token", strings.TrimSpace(pageToken))
	}
	if revisionID, ok := asInt(input["document_revision_id"]); ok {
		query.Set("document_revision_id", strconv.Itoa(revisionID))
	}
	if withDescendants, ok := input["with_descendants"].(bool); ok {
		query.Set("with_descendants", strconv.FormatBool(withDescendants))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/docx/v1/documents/"+url.PathEscape(strings.TrimSpace(documentID))+"/blocks/"+url.PathEscape(strings.TrimSpace(blockID))+"/children",
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response documentBlockChildrenResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode document block children response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}

	items := make([]map[string]any, 0, len(response.Data.Items))
	for _, item := range response.Data.Items {
		items = append(items, normalizeDocxBlock(item))
	}

	return map[string]any{
		"document_id":     strings.TrimSpace(documentID),
		"block_id":        strings.TrimSpace(blockID),
		"items":           items,
		"next_page_token": response.Data.PageToken,
		"has_more":        response.Data.HasMore,
	}, nil
}

// GetDocumentBlockDescendants returns all descendant blocks under the given block.
func (c *Client) GetDocumentBlockDescendants(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	descendantsInput := cloneFeishuInputMap(input)
	descendantsInput["with_descendants"] = true
	return c.GetDocumentBlockChildren(ctx, profile, descendantsInput)
}

// UpdateDocumentBlock updates the text content of a single block.
func (c *Client) UpdateDocumentBlock(ctx context.Context, profile ExecutionProfile, input map[string]any, clientToken string) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	request, appErr := buildUpdateDocumentBlockRequest(input)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	query.Set("document_revision_id", strconv.Itoa(request.DocumentRevisionID))
	if strings.TrimSpace(clientToken) != "" {
		query.Set("client_token", strings.TrimSpace(clientToken))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPatch,
		"/open-apis/docx/v1/documents/"+url.PathEscape(request.DocumentID)+"/blocks/"+url.PathEscape(request.BlockID),
		query,
		request.Body,
		"Bearer "+accessToken,
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		return nil, appErr
	}

	var response patchDocumentBlockResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode document block patch response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}

	data := normalizeDocxBlock(response.Data.Block)
	data["document_id"] = request.DocumentID
	data["document_revision_id"] = response.Data.DocumentRevisionID
	if strings.TrimSpace(response.Data.ClientToken) != "" {
		data["client_token"] = response.Data.ClientToken
	}
	return data, nil
}

// BatchDeleteDocumentBlockChildren deletes a contiguous range of child blocks from a parent block.
func (c *Client) BatchDeleteDocumentBlockChildren(ctx context.Context, profile ExecutionProfile, input map[string]any, clientToken string) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	request, appErr := buildBatchDeleteDocumentBlockChildrenRequest(input)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	query.Set("document_revision_id", strconv.Itoa(request.DocumentRevisionID))
	if strings.TrimSpace(clientToken) != "" {
		query.Set("client_token", strings.TrimSpace(clientToken))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodDelete,
		"/open-apis/docx/v1/documents/"+url.PathEscape(request.DocumentID)+"/blocks/"+url.PathEscape(request.BlockID)+"/children/batch_delete",
		query,
		request.Body,
		"Bearer "+accessToken,
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		return nil, appErr
	}

	var response batchDeleteDocumentBlockChildrenResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode document block children delete response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}

	data := map[string]any{
		"document_id":          request.DocumentID,
		"block_id":             request.BlockID,
		"start_index":          request.Body.StartIndex,
		"end_index":            request.Body.EndIndex,
		"document_revision_id": response.Data.DocumentRevisionID,
	}
	if strings.TrimSpace(response.Data.ClientToken) != "" {
		data["client_token"] = response.Data.ClientToken
	}
	return data, nil
}

// AppendDocumentBlocks appends text-oriented blocks to a docx document.
func (c *Client) AppendDocumentBlocks(ctx context.Context, profile ExecutionProfile, input map[string]any, clientToken string) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	request, appErr := buildAppendBlocksRequest(input)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	query.Set("document_revision_id", "-1")
	if strings.TrimSpace(clientToken) != "" {
		query.Set("client_token", clientToken)
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/open-apis/docx/v1/documents/"+url.PathEscape(request.DocumentID)+"/blocks/"+url.PathEscape(request.BlockID)+"/children",
		query,
		request.Body,
		"Bearer "+accessToken,
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		return nil, appErr
	}

	var response appendBlocksResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode append blocks response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}

	items := make([]map[string]any, 0, len(response.Data.Children))
	for _, child := range response.Data.Children {
		items = append(items, map[string]any{
			"block_id":   child.BlockID,
			"block_type": child.BlockType,
		})
	}

	return map[string]any{
		"document_id":    request.DocumentID,
		"block_id":       request.BlockID,
		"appended_count": len(response.Data.Children),
		"children":       items,
	}, nil
}

// GetDocumentRawContent fetches pure text content from a docx document.
func (c *Client) GetDocumentRawContent(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	documentID, ok := asString(input["document_id"])
	if !ok || strings.TrimSpace(documentID) == "" {
		return nil, apperr.New("INVALID_INPUT", "document_id is required")
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/docx/v1/documents/"+url.PathEscape(strings.TrimSpace(documentID))+"/raw_content",
		nil,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response rawContentResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode raw content response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}

	return map[string]any{
		"document_id": strings.TrimSpace(documentID),
		"content":     response.Data.Content,
	}, nil
}

type appendBlocksRequest struct {
	DocumentID string
	BlockID    string
	Body       appendBlocksPayload
}

type updateDocumentBlockRequest struct {
	DocumentID         string
	BlockID            string
	DocumentRevisionID int
	Body               updateDocumentBlockPayload
}

type batchDeleteDocumentBlockChildrenRequest struct {
	DocumentID         string
	BlockID            string
	DocumentRevisionID int
	Body               batchDeleteDocumentBlockChildrenPayload
}

type documentGetResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Document docxDocument `json:"document"`
	} `json:"data"`
}

type documentBlockListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items     []docxBlockNode `json:"items"`
		PageToken string          `json:"page_token"`
		HasMore   bool            `json:"has_more"`
	} `json:"data"`
}

type documentBlockGetResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Block docxBlockNode `json:"block"`
	} `json:"data"`
}

type documentBlockChildrenResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items     []docxBlockNode `json:"items"`
		PageToken string          `json:"page_token"`
		HasMore   bool            `json:"has_more"`
	} `json:"data"`
}

type patchDocumentBlockResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Block              docxBlockNode `json:"block"`
		DocumentRevisionID int           `json:"document_revision_id"`
		ClientToken        string        `json:"client_token"`
	} `json:"data"`
}

type batchDeleteDocumentBlockChildrenResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		DocumentRevisionID int    `json:"document_revision_id"`
		ClientToken        string `json:"client_token"`
	} `json:"data"`
}

type docxDocument struct {
	DocumentID     string         `json:"document_id"`
	RevisionID     int            `json:"revision_id"`
	Title          string         `json:"title"`
	DisplaySetting map[string]any `json:"display_setting"`
	Cover          map[string]any `json:"cover"`
}

type appendBlocksPayload struct {
	Children []docxBlock `json:"children"`
}

type updateDocumentBlockPayload struct {
	BlockID    string                 `json:"block_id,omitempty"`
	UpdateText *updateDocxTextRequest `json:"update_text,omitempty"`
	UpdateTask *updateDocxTaskRequest `json:"update_task,omitempty"`
}

type updateDocxTextRequest struct {
	Elements []docxTextElement `json:"elements,omitempty"`
}

type updateDocxTaskRequest struct {
	TaskID string `json:"task_id,omitempty"`
	Folded *bool  `json:"folded,omitempty"`
}

type batchDeleteDocumentBlockChildrenPayload struct {
	StartIndex int `json:"start_index"`
	EndIndex   int `json:"end_index"`
}

type docxBlock struct {
	BlockType int           `json:"block_type"`
	Text      *docxTextBody `json:"text,omitempty"`
	Heading1  *docxTextBody `json:"heading1,omitempty"`
	Heading2  *docxTextBody `json:"heading2,omitempty"`
	Heading3  *docxTextBody `json:"heading3,omitempty"`
	Bullet    *docxTextBody `json:"bullet,omitempty"`
	Ordered   *docxTextBody `json:"ordered,omitempty"`
	Quote     *docxTextBody `json:"quote,omitempty"`
	Code      *docxCodeBody `json:"code,omitempty"`
	Todo      *docxTodoBody `json:"todo,omitempty"`
	Divider   *struct{}     `json:"divider,omitempty"`
}

type docxTextBody struct {
	Elements []docxTextElement `json:"elements"`
}

type docxTextElement struct {
	TextRun *docxTextRun `json:"text_run,omitempty"`
}

type docxTextRun struct {
	Content string `json:"content"`
}

type docxCodeBody struct {
	Elements []docxTextElement `json:"elements"`
	Language int               `json:"language,omitempty"`
	Wrap     bool              `json:"wrap,omitempty"`
}

type docxTodoBody struct {
	Elements []docxTextElement `json:"elements"`
	Done     bool              `json:"done,omitempty"`
}

type docxBlockNode struct {
	BlockID   string        `json:"block_id"`
	ParentID  string        `json:"parent_id"`
	Children  []string      `json:"children"`
	BlockType int           `json:"block_type"`
	Page      *docxTextBody `json:"page,omitempty"`
	Text      *docxTextBody `json:"text,omitempty"`
	Heading1  *docxTextBody `json:"heading1,omitempty"`
	Heading2  *docxTextBody `json:"heading2,omitempty"`
	Heading3  *docxTextBody `json:"heading3,omitempty"`
	Bullet    *docxTextBody `json:"bullet,omitempty"`
	Ordered   *docxTextBody `json:"ordered,omitempty"`
	Code      *docxTextBody `json:"code,omitempty"`
	Quote     *docxTextBody `json:"quote,omitempty"`
	Equation  *docxTextBody `json:"equation,omitempty"`
	Todo      *docxTextBody `json:"todo,omitempty"`
}

type appendBlocksResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ClientToken string           `json:"client_token"`
		Children    []docxBlockBrief `json:"children"`
	} `json:"data"`
}

type docxBlockBrief struct {
	BlockID   string `json:"block_id"`
	BlockType int    `json:"block_type"`
}

type rawContentResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Content string `json:"content"`
	} `json:"data"`
}

func buildAppendBlocksRequest(input map[string]any) (*appendBlocksRequest, *apperr.AppError) {
	documentID, ok := asString(input["document_id"])
	if !ok || strings.TrimSpace(documentID) == "" {
		return nil, apperr.New("INVALID_INPUT", "document_id is required")
	}

	blockID := strings.TrimSpace(documentID)
	if value, ok := asString(input["block_id"]); ok && strings.TrimSpace(value) != "" {
		blockID = strings.TrimSpace(value)
	}

	rawBlocks, ok := input["blocks"].([]any)
	if !ok || len(rawBlocks) == 0 {
		return nil, apperr.New("INVALID_INPUT", "blocks must be a non-empty array")
	}

	children := make([]docxBlock, 0, len(rawBlocks))
	for _, raw := range rawBlocks {
		blockMap, ok := raw.(map[string]any)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "each block must be an object")
		}
		block, appErr := buildDocxBlock(blockMap)
		if appErr != nil {
			return nil, appErr
		}
		children = append(children, block)
	}

	return &appendBlocksRequest{
		DocumentID: strings.TrimSpace(documentID),
		BlockID:    blockID,
		Body: appendBlocksPayload{
			Children: children,
		},
	}, nil
}

func buildUpdateDocumentBlockRequest(input map[string]any) (*updateDocumentBlockRequest, *apperr.AppError) {
	documentID, ok := asString(input["document_id"])
	if !ok || strings.TrimSpace(documentID) == "" {
		return nil, apperr.New("INVALID_INPUT", "document_id is required")
	}
	blockID, ok := asString(input["block_id"])
	if !ok || strings.TrimSpace(blockID) == "" {
		return nil, apperr.New("INVALID_INPUT", "block_id is required")
	}

	documentRevisionID := -1
	if value, ok := asInt(input["document_revision_id"]); ok {
		documentRevisionID = value
	}

	payload := updateDocumentBlockPayload{
		BlockID: strings.TrimSpace(blockID),
	}

	if updateTask, exists := input["update_task"]; exists {
		taskRequest, appErr := buildUpdateDocxTaskRequest(updateTask)
		if appErr != nil {
			return nil, appErr
		}
		payload.UpdateTask = taskRequest
		return &updateDocumentBlockRequest{
			DocumentID:         strings.TrimSpace(documentID),
			BlockID:            strings.TrimSpace(blockID),
			DocumentRevisionID: documentRevisionID,
			Body:               payload,
		}, nil
	}

	blockInput := input
	if rawBlock, exists := input["block"]; exists {
		record, ok := rawBlock.(map[string]any)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "block must be an object")
		}
		blockInput = record
	}

	if _, exists := blockInput["checked"]; exists {
		return nil, apperr.New("UNSUPPORTED_FIELD", "checked updates are not supported in feishu.docs.block.update yet")
	}
	if _, exists := blockInput["language"]; exists {
		return nil, apperr.New("UNSUPPORTED_FIELD", "language updates are not supported in feishu.docs.block.update yet")
	}

	textBody, appErr := buildDocxTextBody(blockInput["text"])
	if appErr != nil {
		return nil, appErr
	}
	payload.UpdateText = &updateDocxTextRequest{
		Elements: textBody.Elements,
	}

	return &updateDocumentBlockRequest{
		DocumentID:         strings.TrimSpace(documentID),
		BlockID:            strings.TrimSpace(blockID),
		DocumentRevisionID: documentRevisionID,
		Body:               payload,
	}, nil
}

func buildBatchDeleteDocumentBlockChildrenRequest(input map[string]any) (*batchDeleteDocumentBlockChildrenRequest, *apperr.AppError) {
	documentID, ok := asString(input["document_id"])
	if !ok || strings.TrimSpace(documentID) == "" {
		return nil, apperr.New("INVALID_INPUT", "document_id is required")
	}
	blockID, ok := asString(input["block_id"])
	if !ok || strings.TrimSpace(blockID) == "" {
		return nil, apperr.New("INVALID_INPUT", "block_id is required")
	}
	startIndex, ok := asInt(input["start_index"])
	if !ok || startIndex < 0 {
		return nil, apperr.New("INVALID_INPUT", "start_index must be a non-negative integer")
	}
	endIndex, ok := asInt(input["end_index"])
	if !ok || endIndex <= startIndex {
		return nil, apperr.New("INVALID_INPUT", "end_index must be greater than start_index")
	}

	documentRevisionID := -1
	if value, ok := asInt(input["document_revision_id"]); ok {
		documentRevisionID = value
	}

	return &batchDeleteDocumentBlockChildrenRequest{
		DocumentID:         strings.TrimSpace(documentID),
		BlockID:            strings.TrimSpace(blockID),
		DocumentRevisionID: documentRevisionID,
		Body: batchDeleteDocumentBlockChildrenPayload{
			StartIndex: startIndex,
			EndIndex:   endIndex,
		},
	}, nil
}

func buildUpdateDocxTaskRequest(raw any) (*updateDocxTaskRequest, *apperr.AppError) {
	record, ok := raw.(map[string]any)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "update_task must be an object")
	}

	request := &updateDocxTaskRequest{}
	if taskID, ok := asString(record["task_id"]); ok && strings.TrimSpace(taskID) != "" {
		request.TaskID = strings.TrimSpace(taskID)
	}
	if folded, ok := record["folded"].(bool); ok {
		request.Folded = &folded
	}
	if request.TaskID == "" && request.Folded == nil {
		return nil, apperr.New("INVALID_INPUT", "update_task requires at least one of task_id or folded")
	}
	return request, nil
}

func buildDocxBlock(input map[string]any) (docxBlock, *apperr.AppError) {
	blockType, ok := asString(input["type"])
	if !ok || strings.TrimSpace(blockType) == "" {
		return docxBlock{}, apperr.New("INVALID_INPUT", "block type is required")
	}

	switch strings.TrimSpace(blockType) {
	case "paragraph":
		body, appErr := buildDocxTextBody(input["text"])
		if appErr != nil {
			return docxBlock{}, appErr
		}
		return docxBlock{
			BlockType: 2,
			Text:      body,
		}, nil
	case "heading_1":
		body, appErr := buildDocxTextBody(input["text"])
		if appErr != nil {
			return docxBlock{}, appErr
		}
		return docxBlock{
			BlockType: 3,
			Heading1:  body,
		}, nil
	case "heading_2":
		body, appErr := buildDocxTextBody(input["text"])
		if appErr != nil {
			return docxBlock{}, appErr
		}
		return docxBlock{
			BlockType: 4,
			Heading2:  body,
		}, nil
	case "heading_3":
		body, appErr := buildDocxTextBody(input["text"])
		if appErr != nil {
			return docxBlock{}, appErr
		}
		return docxBlock{
			BlockType: 5,
			Heading3:  body,
		}, nil
	case "bulleted_list_item":
		body, appErr := buildDocxTextBody(input["text"])
		if appErr != nil {
			return docxBlock{}, appErr
		}
		return docxBlock{
			BlockType: 12,
			Bullet:    body,
		}, nil
	case "numbered_list_item":
		body, appErr := buildDocxTextBody(input["text"])
		if appErr != nil {
			return docxBlock{}, appErr
		}
		return docxBlock{
			BlockType: 13,
			Ordered:   body,
		}, nil
	case "quote":
		body, appErr := buildDocxTextBody(input["text"])
		if appErr != nil {
			return docxBlock{}, appErr
		}
		return docxBlock{
			BlockType: 15,
			Quote:     body,
		}, nil
	case "code":
		body, appErr := buildDocxCodeBody(input)
		if appErr != nil {
			return docxBlock{}, appErr
		}
		return docxBlock{
			BlockType: 14,
			Code:      body,
		}, nil
	case "to_do":
		body, appErr := buildDocxTodoBody(input)
		if appErr != nil {
			return docxBlock{}, appErr
		}
		return docxBlock{
			BlockType: 17,
			Todo:      body,
		}, nil
	case "divider":
		return docxBlock{
			BlockType: 22,
			Divider:   &struct{}{},
		}, nil
	default:
		return docxBlock{}, apperr.New("UNSUPPORTED_FIELD", fmt.Sprintf("unsupported docx block type: %s", blockType))
	}
}

func buildDocxTextBody(raw any) (*docxTextBody, *apperr.AppError) {
	text, ok := asString(raw)
	if !ok || text == "" {
		return nil, apperr.New("INVALID_INPUT", "text is required for this block type")
	}

	return &docxTextBody{
		Elements: []docxTextElement{
			{
				TextRun: &docxTextRun{
					Content: text,
				},
			},
		},
	}, nil
}

func buildDocxCodeBody(input map[string]any) (*docxCodeBody, *apperr.AppError) {
	textBody, appErr := buildDocxTextBody(input["text"])
	if appErr != nil {
		return nil, appErr
	}

	body := &docxCodeBody{
		Elements: textBody.Elements,
	}
	if language, ok := asString(input["language"]); ok && strings.TrimSpace(language) != "" {
		body.Language = mapCodeLanguage(strings.TrimSpace(language))
	}
	return body, nil
}

func buildDocxTodoBody(input map[string]any) (*docxTodoBody, *apperr.AppError) {
	textBody, appErr := buildDocxTextBody(input["text"])
	if appErr != nil {
		return nil, appErr
	}

	body := &docxTodoBody{
		Elements: textBody.Elements,
	}
	if checked, ok := input["checked"].(bool); ok {
		body.Done = checked
	}
	return body, nil
}

func mapCodeLanguage(language string) int {
	switch language {
	case "go":
		return 22
	case "python":
		return 49
	case "javascript":
		return 30
	case "typescript":
		return 63
	case "json":
		return 28
	case "markdown":
		return 39
	case "sql":
		return 56
	case "bash", "shell":
		return 7
	default:
		return 1
	}
}

func normalizeDocxBlock(block docxBlockNode) map[string]any {
	children := make([]string, 0, len(block.Children))
	for _, childID := range block.Children {
		if strings.TrimSpace(childID) != "" {
			children = append(children, strings.TrimSpace(childID))
		}
	}

	return map[string]any{
		"block_id":        block.BlockID,
		"parent_id":       block.ParentID,
		"children":        children,
		"block_type":      block.BlockType,
		"block_type_name": describeDocxBlockType(block.BlockType),
		"plain_text":      extractDocxBlockPlainText(block),
	}
}

func extractDocxBlockPlainText(block docxBlockNode) string {
	for _, body := range []*docxTextBody{
		block.Page,
		block.Text,
		block.Heading1,
		block.Heading2,
		block.Heading3,
		block.Bullet,
		block.Ordered,
		block.Code,
		block.Quote,
		block.Equation,
		block.Todo,
	} {
		if body == nil {
			continue
		}
		text := extractDocxTextBody(body)
		if text != "" {
			return text
		}
	}
	return ""
}

func extractDocxTextBody(body *docxTextBody) string {
	if body == nil {
		return ""
	}

	var builder strings.Builder
	for _, element := range body.Elements {
		if element.TextRun == nil {
			continue
		}
		builder.WriteString(element.TextRun.Content)
	}
	return builder.String()
}

func describeDocxBlockType(blockType int) string {
	switch blockType {
	case 1:
		return "page"
	case 2:
		return "paragraph"
	case 3:
		return "heading_1"
	case 4:
		return "heading_2"
	case 5:
		return "heading_3"
	case 6:
		return "heading_4"
	case 7:
		return "heading_5"
	case 8:
		return "heading_6"
	case 9:
		return "heading_7"
	case 10:
		return "heading_8"
	case 11:
		return "heading_9"
	case 12:
		return "bulleted_list_item"
	case 13:
		return "numbered_list_item"
	case 14:
		return "code"
	case 15:
		return "quote"
	case 17:
		return "to_do"
	case 22:
		return "divider"
	default:
		return ""
	}
}

func cloneFeishuInputMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
