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

// GetUser reads a user object and supports both a concrete user_id and user_id=me.
func (c *Client) GetUser(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	userID, appErr := requireIDField(input, "user_id")
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	requestPath := "/v1/users/" + url.PathEscape(userID)
	if userID == "me" {
		requestPath = "/v1/users/me"
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		requestPath,
		nil,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response notionUser
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion user response: %v", err))
	}
	if strings.TrimSpace(response.ID) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "user id is empty in Notion response")
	}

	data := map[string]any{
		"user_id":    response.ID,
		"type":       response.Type,
		"name":       response.Name,
		"avatar_url": response.AvatarURL,
	}
	if response.Person != nil && strings.TrimSpace(response.Person.Email) != "" {
		data["email"] = response.Person.Email
	}
	return data, nil
}
