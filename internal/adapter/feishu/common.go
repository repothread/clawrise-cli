package feishu

import (
	"encoding/json"
	"fmt"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

type feishuEnvelope struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data map[string]any `json:"data"`
}

func decodeFeishuEnvelope(responseBody []byte, decodeErrorMessage string) (map[string]any, *apperr.AppError) {
	var response feishuEnvelope
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("%s: %v", decodeErrorMessage, err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}
	if response.Data == nil {
		response.Data = map[string]any{}
	}
	return response.Data, nil
}

func cloneFeishuMap(raw map[string]any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}
	return cloned
}

func asBool(value any) (bool, bool) {
	boolean, ok := value.(bool)
	return boolean, ok
}

func asMap(value any) (map[string]any, bool) {
	record, ok := value.(map[string]any)
	return record, ok
}

func asArray(value any) ([]any, bool) {
	list, ok := value.([]any)
	return list, ok
}
