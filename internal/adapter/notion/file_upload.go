package notion

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// CreateFileUpload 创建一个 Notion file upload。
func (c *Client) CreateFileUpload(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	payload, appErr := buildCreateFileUploadPayload(input)
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/v1/file_uploads",
		nil,
		payload,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	return decodeFileUploadObject(responseBody, "failed to decode Notion file upload create response")
}

// GetFileUpload 读取单个 Notion file upload。
func (c *Client) GetFileUpload(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	fileUploadID, appErr := requireIDField(input, "file_upload_id")
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/v1/file_uploads/"+url.PathEscape(fileUploadID),
		nil,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	return decodeFileUploadObject(responseBody, "failed to decode Notion file upload response")
}

// ListFileUploads 列出当前集成可见的 file upload。
func (c *Client) ListFileUploads(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	query := url.Values{}
	if status, ok := asString(input["status"]); ok && strings.TrimSpace(status) != "" {
		query.Set("status", strings.TrimSpace(status))
	}
	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		query.Set("start_cursor", strings.TrimSpace(pageToken))
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/v1/file_uploads",
		query,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response notionFileUploadListResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion file upload list response: %v", err))
	}

	items := make([]map[string]any, 0, len(response.Results))
	for _, item := range response.Results {
		items = append(items, normalizeFileUploadObject(item))
	}

	nextPageToken := ""
	if response.NextCursor != nil {
		nextPageToken = strings.TrimSpace(*response.NextCursor)
	}

	return map[string]any{
		"items":           items,
		"next_page_token": nextPageToken,
		"has_more":        response.HasMore,
	}, nil
}

// SendFileUpload 通过 multipart/form-data 向 Notion 发送文件内容。
func (c *Client) SendFileUpload(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	fileUploadID, appErr := requireIDField(input, "file_upload_id")
	if appErr != nil {
		return nil, appErr
	}

	fileName, fileBytes, contentType, partNumber, debugSummary, appErr := buildSendFileUploadRequest(input)
	if appErr != nil {
		return nil, appErr
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if partNumber > 0 {
		if err := writer.WriteField("part_number", strconv.Itoa(partNumber)); err != nil {
			return nil, apperr.New("REQUEST_ENCODE_FAILED", fmt.Sprintf("failed to write part_number field: %v", err))
		}
	}

	fileHeader := make(textproto.MIMEHeader)
	fileHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeMultipartFileName(fileName)))
	if strings.TrimSpace(contentType) != "" {
		fileHeader.Set("Content-Type", strings.TrimSpace(contentType))
	}
	partWriter, err := writer.CreatePart(fileHeader)
	if err != nil {
		return nil, apperr.New("REQUEST_ENCODE_FAILED", fmt.Sprintf("failed to create multipart file part: %v", err))
	}
	if _, err := partWriter.Write(fileBytes); err != nil {
		return nil, apperr.New("REQUEST_ENCODE_FAILED", fmt.Sprintf("failed to write multipart file bytes: %v", err))
	}
	if err := writer.Close(); err != nil {
		return nil, apperr.New("REQUEST_ENCODE_FAILED", fmt.Sprintf("failed to finalize multipart request: %v", err))
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doMultipartRequest(
		ctx,
		http.MethodPost,
		"/v1/file_uploads/"+url.PathEscape(fileUploadID)+"/send",
		nil,
		&body,
		writer.FormDataContentType(),
		"Bearer "+accessToken,
		notionVersion,
		debugSummary,
	)
	if appErr != nil {
		return nil, appErr
	}

	return decodeFileUploadObject(responseBody, "failed to decode Notion file upload send response")
}

// CompleteFileUpload 完成一个 multi-part file upload。
func (c *Client) CompleteFileUpload(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	fileUploadID, appErr := requireIDField(input, "file_upload_id")
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/v1/file_uploads/"+url.PathEscape(fileUploadID)+"/complete",
		nil,
		map[string]any{},
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	return decodeFileUploadObject(responseBody, "failed to decode Notion file upload complete response")
}

// doMultipartRequest 专门处理 file upload send 这类 multipart 请求。
func (c *Client) doMultipartRequest(ctx context.Context, method, rawPath string, query url.Values, body io.Reader, contentType string, authorization string, notionVersion string, debugSummary map[string]any) ([]byte, *apperr.AppError) {
	endpoint := *c.baseURL
	endpoint.Path = filepath.Clean(strings.TrimRight(c.baseURL.Path, "/") + "/" + strings.TrimLeft(rawPath, "/"))
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return nil, apperr.New("REQUEST_BUILD_FAILED", fmt.Sprintf("failed to build Notion multipart request: %v", err))
	}

	request.Header.Set("Accept", "application/json")
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	if strings.TrimSpace(notionVersion) != "" {
		request.Header.Set("Notion-Version", notionVersion)
	}
	request.Header.Set("Content-Type", contentType)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, apperr.New("HTTP_REQUEST_FAILED", fmt.Sprintf("failed to call Notion API: %v", err)).WithRetryable(true)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, apperr.New("HTTP_READ_FAILED", fmt.Sprintf("failed to read Notion response: %v", err))
	}
	if providerDebugEnabled(ctx) {
		adapter.AddProviderDebugEvent(ctx, map[string]any{
			"provider":        "notion",
			"method":          method,
			"path":            endpoint.Path,
			"query":           endpoint.RawQuery,
			"request_body":    adapter.RedactDebugValue(cloneDebugValue(debugSummary)),
			"response_status": response.StatusCode,
			"response_body":   adapter.RedactDebugValue(decodeDebugResponseBody(responseBody)),
		})
	}

	if response.StatusCode >= 400 {
		return nil, normalizeNotionHTTPError(response.StatusCode, response.Header, responseBody)
	}
	return responseBody, nil
}

func buildCreateFileUploadPayload(input map[string]any) (map[string]any, *apperr.AppError) {
	payload := map[string]any{}

	mode := "single_part"
	if value, ok := asString(input["mode"]); ok && strings.TrimSpace(value) != "" {
		mode = strings.TrimSpace(value)
	}
	switch mode {
	case "single_part", "multi_part", "external_url":
		payload["mode"] = mode
	default:
		return nil, apperr.New("INVALID_INPUT", "mode must be one of single_part, multi_part, or external_url")
	}

	if filename, ok := asString(input["filename"]); ok && strings.TrimSpace(filename) != "" {
		payload["filename"] = strings.TrimSpace(filename)
	}
	if contentType, ok := asString(input["content_type"]); ok && strings.TrimSpace(contentType) != "" {
		payload["content_type"] = strings.TrimSpace(contentType)
	}

	if mode == "multi_part" {
		filename, ok := asString(payload["filename"])
		if !ok || strings.TrimSpace(filename) == "" {
			return nil, apperr.New("INVALID_INPUT", "filename is required when mode is multi_part")
		}
		numberOfParts, ok := asInt(input["number_of_parts"])
		if !ok || numberOfParts <= 0 {
			return nil, apperr.New("INVALID_INPUT", "number_of_parts is required when mode is multi_part")
		}
		payload["number_of_parts"] = numberOfParts
	}

	if mode == "external_url" {
		externalURL, ok := asString(input["external_url"])
		if !ok || strings.TrimSpace(externalURL) == "" {
			return nil, apperr.New("INVALID_INPUT", "external_url is required when mode is external_url")
		}
		payload["external_url"] = strings.TrimSpace(externalURL)
	}

	return payload, nil
}

func buildSendFileUploadRequest(input map[string]any) (string, []byte, string, int, map[string]any, *apperr.AppError) {
	fileName := ""
	if value, ok := asString(input["filename"]); ok && strings.TrimSpace(value) != "" {
		fileName = strings.TrimSpace(value)
	}
	contentType, _ := asString(input["content_type"])
	contentType = strings.TrimSpace(contentType)

	var fileBytes []byte
	switch {
	case hasNonEmptyString(input, "file_path"):
		filePath, _ := asString(input["file_path"])
		loadedBytes, err := os.ReadFile(strings.TrimSpace(filePath))
		if err != nil {
			return "", nil, "", 0, nil, apperr.New("INVALID_INPUT", fmt.Sprintf("failed to read file_path: %v", err))
		}
		fileBytes = loadedBytes
		if fileName == "" {
			fileName = filepath.Base(strings.TrimSpace(filePath))
		}
	case hasNonEmptyString(input, "content_base64"):
		rawBase64, _ := asString(input["content_base64"])
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(rawBase64))
		if err != nil {
			return "", nil, "", 0, nil, apperr.New("INVALID_INPUT", fmt.Sprintf("failed to decode content_base64: %v", err))
		}
		fileBytes = decoded
	default:
		return "", nil, "", 0, nil, apperr.New("INVALID_INPUT", "one of file_path or content_base64 is required")
	}

	if fileName == "" {
		return "", nil, "", 0, nil, apperr.New("INVALID_INPUT", "filename is required when sending base64 content")
	}

	partNumber := 0
	if value, ok := asInt(input["part_number"]); ok {
		if value <= 0 {
			return "", nil, "", 0, nil, apperr.New("INVALID_INPUT", "part_number must be greater than 0")
		}
		partNumber = value
	}

	debugSummary := map[string]any{
		"file_name":      fileName,
		"content_type":   contentType,
		"content_length": len(fileBytes),
	}
	if partNumber > 0 {
		debugSummary["part_number"] = partNumber
	}
	if filePath, ok := asString(input["file_path"]); ok && strings.TrimSpace(filePath) != "" {
		debugSummary["file_path"] = strings.TrimSpace(filePath)
	}

	return fileName, fileBytes, contentType, partNumber, debugSummary, nil
}

func decodeFileUploadObject(responseBody []byte, decodeErrorMessage string) (map[string]any, *apperr.AppError) {
	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("%s: %v", decodeErrorMessage, err))
	}
	fileUploadID, ok := asString(response["id"])
	if !ok || strings.TrimSpace(fileUploadID) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "file upload id is empty in Notion response")
	}
	return normalizeFileUploadObject(response), nil
}

func normalizeFileUploadObject(item map[string]any) map[string]any {
	result := map[string]any{
		"file_upload_id": extractFirstString(item, "id"),
		"raw":            cloneMap(item),
	}
	if objectType, ok := asString(item["object"]); ok && strings.TrimSpace(objectType) != "" {
		result["object"] = strings.TrimSpace(objectType)
	}
	if createdTime := extractFirstString(item, "created_time"); createdTime != "" {
		result["created_time"] = createdTime
	}
	if lastEditedTime := extractFirstString(item, "last_edited_time"); lastEditedTime != "" {
		result["last_edited_time"] = lastEditedTime
	}
	if expiryTime := extractFirstString(item, "expiry_time"); expiryTime != "" {
		result["expiry_time"] = expiryTime
	}
	if status := extractFirstString(item, "status"); status != "" {
		result["status"] = status
	}
	if filename := extractFirstString(item, "filename"); filename != "" {
		result["filename"] = filename
	}
	if contentType := extractFirstString(item, "content_type"); contentType != "" {
		result["content_type"] = contentType
	}
	if contentLength, ok := asInt(item["content_length"]); ok {
		result["content_length"] = contentLength
	}
	if uploadURL := extractFirstString(item, "upload_url"); uploadURL != "" {
		result["upload_url"] = uploadURL
	}
	if completeURL := extractFirstString(item, "complete_url"); completeURL != "" {
		result["complete_url"] = completeURL
	}
	if createdBy, ok := asMap(item["created_by"]); ok && len(createdBy) > 0 {
		result["created_by"] = cloneMap(createdBy)
	}
	if fileImportResult, ok := asMap(item["file_import_result"]); ok && len(fileImportResult) > 0 {
		result["file_import_result"] = cloneMap(fileImportResult)
	}
	if numberOfParts, ok := asMap(item["number_of_parts"]); ok && len(numberOfParts) > 0 {
		result["number_of_parts"] = cloneMap(numberOfParts)
	}
	if archived, ok := asBool(item["archived"]); ok {
		result["archived"] = archived
	}
	if inTrash, ok := asBool(item["in_trash"]); ok {
		result["in_trash"] = inTrash
		if archived, exists := result["archived"]; !exists {
			result["archived"] = inTrash
		} else if archivedBool, ok := archived.(bool); ok {
			result["archived"] = archivedBool || inTrash
		}
	}
	return result
}

func hasNonEmptyString(input map[string]any, key string) bool {
	value, ok := asString(input[key])
	return ok && strings.TrimSpace(value) != ""
}

func escapeMultipartFileName(fileName string) string {
	fileName = strings.ReplaceAll(fileName, "\\", "\\\\")
	fileName = strings.ReplaceAll(fileName, `"`, "\\\"")
	return fileName
}
