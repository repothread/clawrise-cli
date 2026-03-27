package spec

import (
	"fmt"
	"sort"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

// Service exposes the registry as a structured `spec` view.
type Service struct {
	registry       *adapter.Registry
	catalogEntries []speccatalog.Entry
}

// NewService creates a `spec` service.
func NewService(registry *adapter.Registry) *Service {
	return newServiceWithCatalog(registry, speccatalog.All())
}

// NewServiceWithCatalog creates a `spec` service with an explicit catalog source.
func NewServiceWithCatalog(registry *adapter.Registry, catalogEntries []speccatalog.Entry) *Service {
	return newServiceWithCatalog(registry, catalogEntries)
}

func newServiceWithCatalog(registry *adapter.Registry, catalogEntries []speccatalog.Entry) *Service {
	return &Service{
		registry:       registry,
		catalogEntries: append([]speccatalog.Entry(nil), catalogEntries...),
	}
}

// List returns the next-level nodes under the given path.
func (s *Service) List(path string) (ListResult, error) {
	path, parts, err := normalizePath(path)
	if err != nil {
		return ListResult{}, err
	}

	definitions := s.registry.Definitions()
	if path != "" {
		if _, ok := s.registry.Resolve(path); ok {
			return ListResult{}, newError("SPEC_PATH_IS_OPERATION", fmt.Sprintf("%s is an operation; use `clawrise spec get %s` instead", path, path))
		}
	}

	itemsByPath := map[string]*ListItem{}
	childrenByPath := map[string]map[string]struct{}{}
	hasMatch := path == ""

	for _, definition := range definitions {
		opParts := strings.Split(definition.Operation, ".")
		if !hasPrefix(opParts, parts) {
			continue
		}
		hasMatch = true

		if len(opParts) <= len(parts) {
			continue
		}

		childDepth := len(parts) + 1
		childFullPath := strings.Join(opParts[:childDepth], ".")
		childName := opParts[childDepth-1]

		item, exists := itemsByPath[childFullPath]
		if !exists {
			item = &ListItem{
				Name:     childName,
				FullPath: childFullPath,
				NodeType: nodeTypeForChild(parts, childDepth, len(opParts)),
			}
			itemsByPath[childFullPath] = item
		}

		if item.NodeType == "operation" {
			implemented := definition.Handler != nil
			mutating := definition.Mutating
			item.Implemented = &implemented
			item.Mutating = &mutating
			item.Summary = definition.Spec.Summary
			continue
		}

		item.OperationCount++
		if len(opParts) > childDepth {
			if _, ok := childrenByPath[childFullPath]; !ok {
				childrenByPath[childFullPath] = map[string]struct{}{}
			}
			childrenByPath[childFullPath][strings.Join(opParts[:childDepth+1], ".")] = struct{}{}
		}
	}

	if !hasMatch {
		return ListResult{}, newError("SPEC_PATH_NOT_FOUND", fmt.Sprintf("spec path %s does not exist", path))
	}

	items := make([]ListItem, 0, len(itemsByPath))
	for _, item := range itemsByPath {
		if item.NodeType != "operation" {
			item.ChildCount = len(childrenByPath[item.FullPath])
		}
		items = append(items, *item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].FullPath < items[j].FullPath
	})

	return ListResult{
		Path:     path,
		NodeType: nodeTypeForPath(parts),
		Depth:    1,
		Items:    items,
	}, nil
}

// Get returns the details for one operation.
func (s *Service) Get(operation string) (OperationView, error) {
	operation, parts, err := normalizePath(operation)
	if err != nil {
		return OperationView{}, err
	}
	if len(parts) < 3 {
		return OperationView{}, newError("SPEC_PATH_NOT_OPERATION", fmt.Sprintf("%s is not a complete operation", operation))
	}

	definition, ok := s.registry.Resolve(operation)
	if !ok {
		return OperationView{}, newError("OPERATION_NOT_FOUND", fmt.Sprintf("operation %s is not registered", operation))
	}

	return buildOperationView(definition), nil
}

// CompletionData 返回 completion 所需的结构化事实集。
func (s *Service) CompletionData() CompletionData {
	operations := make([]string, 0)
	operationIndex := map[string]struct{}{}
	pathIndex := map[string]struct{}{}

	// 运行时 operation 是真正可执行的命令集合，直接用于主命令补全。
	for _, definition := range s.registry.Definitions() {
		if _, exists := operationIndex[definition.Operation]; !exists {
			operationIndex[definition.Operation] = struct{}{}
			operations = append(operations, definition.Operation)
		}
		addSpecPaths(pathIndex, definition.Operation)
	}

	// catalog 中声明但暂未实现的路径也纳入 spec 路径补全，便于浏览和导出。
	for _, entry := range s.catalogEntries {
		addSpecPaths(pathIndex, entry.Operation)
	}

	paths := make([]string, 0, len(pathIndex))
	for path := range pathIndex {
		paths = append(paths, path)
	}

	sort.Strings(operations)
	sort.Strings(paths)
	return CompletionData{
		Operations: operations,
		SpecPaths:  paths,
	}
}

func normalizePath(path string) (string, []string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil, nil
	}
	if strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".") || strings.Contains(path, "..") {
		return "", nil, newError("INVALID_SPEC_PATH", fmt.Sprintf("invalid spec path: %s", path))
	}

	parts := strings.Split(path, ".")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return "", nil, newError("INVALID_SPEC_PATH", fmt.Sprintf("invalid spec path: %s", path))
		}
	}
	return path, parts, nil
}

func hasPrefix(values, prefix []string) bool {
	if len(prefix) > len(values) {
		return false
	}
	for index := range prefix {
		if values[index] != prefix[index] {
			return false
		}
	}
	return true
}

func nodeTypeForPath(parts []string) string {
	switch len(parts) {
	case 0:
		return "root"
	case 1:
		return "platform"
	default:
		return "group"
	}
}

func nodeTypeForChild(parentParts []string, childDepth int, operationPartCount int) string {
	if childDepth == operationPartCount {
		return "operation"
	}
	if len(parentParts) == 0 && childDepth == 1 {
		return "platform"
	}
	return "group"
}

func runtimeStatusForDefinition(definition adapter.Definition) string {
	if definition.Handler == nil {
		return "registered_but_stubbed"
	}
	return "registered_and_implemented"
}

func buildOperationView(definition adapter.Definition) OperationView {
	parts := strings.Split(definition.Operation, ".")
	implemented := definition.Handler != nil
	return OperationView{
		Operation:        definition.Operation,
		Platform:         parts[0],
		ResourcePath:     strings.Join(parts[1:len(parts)-1], "."),
		Action:           parts[len(parts)-1],
		Summary:          definition.Spec.Summary,
		Description:      definition.Spec.Description,
		AllowedSubjects:  append([]string(nil), definition.AllowedSubjects...),
		Mutating:         definition.Mutating,
		Implemented:      implemented,
		DryRunSupported:  definition.Spec.DryRunSupported,
		DefaultTimeoutMS: definition.DefaultTimeout.Milliseconds(),
		Idempotency:      definition.Spec.Idempotency,
		Input:            definition.Spec.Input,
		Examples:         append([]adapter.ExampleSpec(nil), definition.Spec.Examples...),
		RuntimeStatus:    runtimeStatusForDefinition(definition),
	}
}

func addSpecPaths(index map[string]struct{}, operation string) {
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return
	}

	parts := strings.Split(operation, ".")
	for depth := 1; depth <= len(parts); depth++ {
		index[strings.Join(parts[:depth], ".")] = struct{}{}
	}
}
