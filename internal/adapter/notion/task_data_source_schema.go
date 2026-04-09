package notion

import (
	"context"
	"fmt"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// EnsureDataSourceSchema safely adds missing schema fields or options without rewriting the whole data source definition.
// 这个 task 默认只做增量补齐，避免 AI 因为一次 schema 对齐误改现有表结构。
func (c *Client) EnsureDataSourceSchema(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	resolved, appErr := c.resolveTaskDataSourceTarget(ctx, profile, input)
	if appErr != nil {
		return nil, appErr
	}

	dataSourceData, _ := asMap(resolved["data_source"])
	if len(dataSourceData) == 0 {
		return nil, apperr.New("OBJECT_NOT_FOUND", "failed to resolve one target data source")
	}

	currentProperties, _ := asMap(dataSourceData["properties"])
	updateBody := map[string]any{}
	changedProperties := map[string]any{}
	addedProperties := make([]string, 0)
	extendedProperties := make([]string, 0)
	skippedProperties := make([]string, 0)
	intentProvided := false

	if title, provided, appErr := buildOptionalRichTextInput(input["title"], "title"); appErr != nil {
		return nil, appErr
	} else if provided {
		intentProvided = true
		currentTitle, _ := asString(dataSourceData["title"])
		desiredTitle := extractRichTextPlainText(toAnySlice(title))
		if strings.TrimSpace(currentTitle) != strings.TrimSpace(desiredTitle) {
			updateBody["title"] = title
		}
	}

	if description, provided, appErr := buildOptionalRichTextInput(input["description"], "description"); appErr != nil {
		return nil, appErr
	} else if provided {
		intentProvided = true
		currentDescription, _ := asString(dataSourceData["description"])
		desiredDescription := extractRichTextPlainText(toAnySlice(description))
		if strings.TrimSpace(currentDescription) != strings.TrimSpace(desiredDescription) {
			updateBody["description"] = description
		}
	}

	if rawProperties, exists := input["properties"]; exists {
		intentProvided = true
		desiredProperties, ok := asMap(rawProperties)
		if !ok || len(desiredProperties) == 0 {
			return nil, apperr.New("INVALID_INPUT", "properties must be a non-empty object")
		}

		currentTitlePropertyName := findDataSourceTitlePropertyName(currentProperties)
		for propertyName, rawDesired := range desiredProperties {
			desiredProperty, ok := asMap(rawDesired)
			if !ok || len(desiredProperty) == 0 {
				return nil, apperr.New("INVALID_INPUT", fmt.Sprintf("properties.%s must be a non-empty object", propertyName))
			}

			desiredType := detectSchemaPropertyType(desiredProperty)
			if desiredType == "" {
				return nil, apperr.New("INVALID_INPUT", fmt.Sprintf("properties.%s must include a valid property type", propertyName))
			}
			if desiredType == "title" && currentTitlePropertyName != "" && currentTitlePropertyName != propertyName {
				return nil, apperr.New("SCHEMA_CONFLICT", fmt.Sprintf("data source already has title property %q; cannot add another title property %q", currentTitlePropertyName, propertyName))
			}

			currentProperty, exists := currentProperties[propertyName]
			if !exists {
				changedProperties[propertyName] = cloneMap(desiredProperty)
				addedProperties = append(addedProperties, propertyName)
				continue
			}

			currentPropertyMap, ok := asMap(currentProperty)
			if !ok || len(currentPropertyMap) == 0 {
				changedProperties[propertyName] = cloneMap(desiredProperty)
				addedProperties = append(addedProperties, propertyName)
				continue
			}

			updateProperty, changed, extended, appErr := buildSchemaEnsurePropertyUpdate(propertyName, currentPropertyMap, desiredProperty)
			if appErr != nil {
				return nil, appErr
			}
			if changed {
				changedProperties[propertyName] = updateProperty
				if extended {
					extendedProperties = append(extendedProperties, propertyName)
				}
				continue
			}
			skippedProperties = append(skippedProperties, propertyName)
		}
	}
	if !intentProvided {
		return nil, apperr.New("INVALID_INPUT", "at least one of title, description, or properties is required")
	}

	if len(changedProperties) > 0 {
		updateBody["properties"] = changedProperties
	}
	if len(updateBody) == 0 {
		return map[string]any{
			"action":              "noop",
			"data_source_id":      resolved["data_source_id"],
			"database_id":         resolved["database_id"],
			"added_properties":    addedProperties,
			"extended_properties": extendedProperties,
			"skipped_properties":  skippedProperties,
			"data_source":         cloneMap(dataSourceData),
		}, nil
	}

	updateInput := map[string]any{
		"data_source_id": resolved["data_source_id"],
		"body":           updateBody,
	}
	updated, appErr := c.UpdateDataSource(ctx, profile, updateInput)
	if appErr != nil {
		return nil, appErr
	}

	return map[string]any{
		"action":              "updated",
		"data_source_id":      resolved["data_source_id"],
		"database_id":         resolved["database_id"],
		"added_properties":    addedProperties,
		"extended_properties": extendedProperties,
		"skipped_properties":  skippedProperties,
		"update_body":         updateBody,
		"data_source":         cloneMap(updated),
	}, nil
}

func buildSchemaEnsurePropertyUpdate(propertyName string, current map[string]any, desired map[string]any) (map[string]any, bool, bool, *apperr.AppError) {
	currentType := detectSchemaPropertyType(current)
	desiredType := detectSchemaPropertyType(desired)
	if currentType == "" || desiredType == "" {
		return nil, false, false, apperr.New("SCHEMA_CONFLICT", fmt.Sprintf("failed to determine schema type for property %q", propertyName))
	}
	if currentType != desiredType {
		return nil, false, false, apperr.New("SCHEMA_CONFLICT", fmt.Sprintf("property %q already exists with type %q; desired type is %q", propertyName, currentType, desiredType))
	}

	switch desiredType {
	case "select", "multi_select", "status":
		merged, changed, appErr := buildMergedSchemaOptionUpdate(current, desired, desiredType)
		if appErr != nil {
			return nil, false, false, appErr
		}
		return merged, changed, changed, nil
	default:
		return nil, false, false, nil
	}
}

func buildMergedSchemaOptionUpdate(current map[string]any, desired map[string]any, propertyType string) (map[string]any, bool, *apperr.AppError) {
	currentBody, _ := asMap(current[propertyType])
	desiredBody, ok := asMap(desired[propertyType])
	if !ok {
		return nil, false, apperr.New("INVALID_INPUT", fmt.Sprintf("%s property definition must include a %s body", propertyType, propertyType))
	}

	currentOptions, appErr := extractSchemaOptions(currentBody)
	if appErr != nil {
		return nil, false, appErr
	}
	desiredOptions, appErr := extractSchemaOptions(desiredBody)
	if appErr != nil {
		return nil, false, appErr
	}
	if len(desiredOptions) == 0 {
		return nil, false, nil
	}

	optionNames := map[string]struct{}{}
	mergedOptions := make([]map[string]any, 0, len(currentOptions)+len(desiredOptions))
	for _, option := range currentOptions {
		name := extractFirstString(option, "name")
		if name != "" {
			optionNames[name] = struct{}{}
		}
		mergedOptions = append(mergedOptions, cloneMap(option))
	}

	changed := false
	for _, option := range desiredOptions {
		name := extractFirstString(option, "name")
		if name == "" {
			return nil, false, apperr.New("INVALID_INPUT", fmt.Sprintf("%s options must include name", propertyType))
		}
		if _, exists := optionNames[name]; exists {
			continue
		}
		changed = true
		optionNames[name] = struct{}{}
		mergedOptions = append(mergedOptions, cloneMap(option))
	}
	if !changed {
		return nil, false, nil
	}

	update := cloneMap(desired)
	updateBody, _ := asMap(update[propertyType])
	if updateBody == nil {
		updateBody = map[string]any{}
	}
	updateBody = cloneMap(updateBody)
	updateBody["options"] = mergedOptions
	update[propertyType] = updateBody
	return update, true, nil
}

func extractSchemaOptions(body map[string]any) ([]map[string]any, *apperr.AppError) {
	if len(body) == 0 {
		return nil, nil
	}
	rawOptions, exists := body["options"]
	if !exists {
		return nil, nil
	}
	items, ok := asArray(rawOptions)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "property options must be an array")
	}

	options := make([]map[string]any, 0, len(items))
	for _, item := range items {
		record, ok := asMap(item)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "each property option must be an object")
		}
		options = append(options, cloneMap(record))
	}
	return options, nil
}

func detectSchemaPropertyType(property map[string]any) string {
	if property == nil {
		return ""
	}
	if propertyType, ok := asString(property["type"]); ok && strings.TrimSpace(propertyType) != "" {
		return strings.TrimSpace(propertyType)
	}
	for _, candidate := range []string{"title", "rich_text", "number", "select", "multi_select", "status", "date", "people", "files", "checkbox", "url", "email", "phone_number", "formula", "relation", "rollup", "created_time", "created_by", "last_edited_time", "last_edited_by", "button"} {
		if _, exists := property[candidate]; exists {
			return candidate
		}
	}
	return ""
}

func findDataSourceTitlePropertyName(properties map[string]any) string {
	for propertyName, rawProperty := range properties {
		property, ok := asMap(rawProperty)
		if !ok {
			continue
		}
		if detectSchemaPropertyType(property) == "title" {
			return propertyName
		}
	}
	return ""
}

func (c *Client) resolveTaskDataSourceTarget(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	if dataSourceID, ok := asString(input["data_source_id"]); ok && strings.TrimSpace(dataSourceID) != "" {
		dataSource, appErr := c.GetDataSource(ctx, profile, map[string]any{
			"data_source_id": strings.TrimSpace(dataSourceID),
		})
		if appErr != nil {
			return nil, appErr
		}
		result := map[string]any{
			"data_source_id": strings.TrimSpace(dataSourceID),
			"data_source":    cloneMap(dataSource),
		}
		parent, _ := asMap(dataSource["parent"])
		if databaseID, ok := asString(parent["database_id"]); ok && strings.TrimSpace(databaseID) != "" {
			result["database_id"] = strings.TrimSpace(databaseID)
		}
		return result, nil
	}

	resolved, appErr := c.ResolveDatabaseTarget(ctx, profile, input)
	if appErr != nil {
		return nil, appErr
	}
	if dataSourceID, ok := asString(resolved["data_source_id"]); ok && strings.TrimSpace(dataSourceID) != "" {
		return resolved, nil
	}
	return nil, apperr.New("OBJECT_NOT_FOUND", "failed to resolve one target data source")
}

func toAnySlice(items []map[string]any) []any {
	if len(items) == 0 {
		return nil
	}
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}
