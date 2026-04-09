package notion

import (
	"context"
	"mime"
	"path/filepath"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// AttachFileBlock 上传一个文件并把它作为 image/file block 追加到目标 block 下。
// AttachFileBlock uploads one file payload and appends it under the target block as an image or file block.
func (c *Client) AttachFileBlock(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	blockID, appErr := requireIDField(input, "block_id")
	if appErr != nil {
		return nil, appErr
	}

	fileSpec, appErr := buildTaskBlockFileSpec(input)
	if appErr != nil {
		return nil, appErr
	}

	createData, appErr := c.CreateFileUpload(ctx, profile, map[string]any{
		"mode":         "single_part",
		"filename":     fileSpec.FileName,
		"content_type": fileSpec.ContentType,
	})
	if appErr != nil {
		return nil, appErr
	}

	fileUploadID, _ := asString(createData["file_upload_id"])
	sendInput := map[string]any{
		"file_upload_id": strings.TrimSpace(fileUploadID),
		"filename":       fileSpec.FileName,
	}
	if fileSpec.ContentType != "" {
		sendInput["content_type"] = fileSpec.ContentType
	}
	if fileSpec.Source == "file_path" {
		sendInput["file_path"] = fileSpec.FilePath
	} else {
		sendInput["content_base64"] = fileSpec.ContentBase64
	}

	sendData, appErr := c.SendFileUpload(ctx, profile, sendInput)
	if appErr != nil {
		return nil, appErr
	}

	child := map[string]any{
		"type":           fileSpec.BlockType,
		"file_upload_id": strings.TrimSpace(fileUploadID),
	}
	if fileSpec.Caption != "" {
		child["caption"] = fileSpec.Caption
	}
	if fileSpec.CaptionRichText != nil {
		child["caption_rich_text"] = cloneDebugValue(fileSpec.CaptionRichText)
	}

	appendInput := map[string]any{
		"block_id": blockID,
		"children": []any{child},
	}
	copyOptionalTaskFields(input, appendInput, "position", "after")

	appendData, appErr := c.AppendBlockChildren(ctx, profile, appendInput)
	if appErr != nil {
		return nil, appErr
	}

	return map[string]any{
		"block_id":    blockID,
		"block_type":  fileSpec.BlockType,
		"source":      fileSpec.Source,
		"file_upload": cloneMap(createData),
		"upload_send": cloneMap(sendData),
		"append":      cloneMap(appendData),
		"child_ids":   appendData["child_ids"],
	}, nil
}

type taskBlockFileSpec struct {
	Source          string
	FilePath        string
	ContentBase64   string
	FileName        string
	ContentType     string
	BlockType       string
	Caption         string
	CaptionRichText any
}

// buildTaskBlockFileSpec 统一任务命令对文件输入、文件名、内容类型和目标 block 类型的推断。
// buildTaskBlockFileSpec centralizes task-level inference for file source, file name, content type, and target block type.
func buildTaskBlockFileSpec(input map[string]any) (taskBlockFileSpec, *apperr.AppError) {
	spec := taskBlockFileSpec{}

	rawFilePath, hasFilePath := input["file_path"]
	rawContentBase64, hasContentBase64 := input["content_base64"]
	if hasFilePath == hasContentBase64 {
		return spec, apperr.New("INVALID_INPUT", "provide exactly one of file_path or content_base64")
	}

	if hasFilePath {
		filePath, ok := asString(rawFilePath)
		if !ok || strings.TrimSpace(filePath) == "" {
			return spec, apperr.New("INVALID_INPUT", "file_path must be a non-empty string")
		}
		spec.Source = "file_path"
		spec.FilePath = strings.TrimSpace(filePath)
	}
	if hasContentBase64 {
		contentBase64, ok := asString(rawContentBase64)
		if !ok || strings.TrimSpace(contentBase64) == "" {
			return spec, apperr.New("INVALID_INPUT", "content_base64 must be a non-empty string")
		}
		spec.Source = "content_base64"
		spec.ContentBase64 = strings.TrimSpace(contentBase64)
	}

	if value, ok := asString(input["filename"]); ok && strings.TrimSpace(value) != "" {
		spec.FileName = strings.TrimSpace(value)
	} else if spec.Source == "file_path" {
		spec.FileName = filepath.Base(spec.FilePath)
	}
	if spec.FileName == "" {
		return spec, apperr.New("INVALID_INPUT", "filename is required when content_base64 is used")
	}

	if value, ok := asString(input["content_type"]); ok && strings.TrimSpace(value) != "" {
		spec.ContentType = strings.TrimSpace(value)
	} else if inferredType := inferTaskBlockContentType(spec.FileName); inferredType != "" {
		spec.ContentType = inferredType
	}

	if value, ok := asString(input["block_type"]); ok && strings.TrimSpace(value) != "" {
		spec.BlockType = strings.TrimSpace(value)
		switch spec.BlockType {
		case "image", "file":
		default:
			return spec, apperr.New("INVALID_INPUT", "block_type must be image or file")
		}
	} else {
		spec.BlockType = inferTaskBlockType(spec.ContentType)
	}

	if value, ok := asString(input["caption"]); ok && strings.TrimSpace(value) != "" {
		spec.Caption = strings.TrimSpace(value)
	}
	if captionRichText, exists := input["caption_rich_text"]; exists && captionRichText != nil {
		spec.CaptionRichText = captionRichText
	}

	return spec, nil
}

func inferTaskBlockContentType(fileName string) string {
	fileExtension := strings.TrimSpace(filepath.Ext(fileName))
	if fileExtension == "" {
		return ""
	}
	return strings.TrimSpace(mime.TypeByExtension(fileExtension))
}

func inferTaskBlockType(contentType string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "image/") {
		return "image"
	}
	return "file"
}
