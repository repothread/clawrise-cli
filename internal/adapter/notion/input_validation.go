package notion

import (
	"fmt"
	"sort"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
)

func validateTopLevelInputFields(operation string, input map[string]any, spec adapter.InputSpec, fieldGuidance map[string]string) *apperr.AppError {
	if len(input) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(spec.Required)+len(spec.Optional))
	for _, field := range spec.Required {
		field = strings.TrimSpace(field)
		if field != "" {
			allowed[field] = struct{}{}
		}
	}
	for _, field := range spec.Optional {
		field = strings.TrimSpace(field)
		if field != "" {
			allowed[field] = struct{}{}
		}
	}

	unsupported := make([]string, 0)
	for field := range input {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, ok := allowed[field]; ok {
			continue
		}
		unsupported = append(unsupported, field)
	}
	if len(unsupported) == 0 {
		return nil
	}

	sort.Strings(unsupported)
	if len(unsupported) == 1 {
		field := unsupported[0]
		if guidance := strings.TrimSpace(fieldGuidance[field]); guidance != "" {
			return apperr.New("INVALID_INPUT", fmt.Sprintf("%s is not supported by %s; %s", field, operation, guidance))
		}
		return apperr.New("INVALID_INPUT", fmt.Sprintf("%s is not supported by %s", field, operation))
	}

	return apperr.New("INVALID_INPUT", fmt.Sprintf("unsupported fields for %s: %s", operation, strings.Join(unsupported, ", ")))
}
