package adapter

import (
	"context"
	"time"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

// Call 描述一次 operation 执行时传入 adapter handler 的上下文。
type Call struct {
	Profile        config.Profile
	Input          map[string]any
	IdempotencyKey string
}

// HandlerFunc 是 adapter 层的 operation 执行函数。
type HandlerFunc func(ctx context.Context, call Call) (map[string]any, *apperr.AppError)

// Definition stores the base metadata for one operation.
type Definition struct {
	Operation       string
	Platform        string
	Mutating        bool
	DefaultTimeout  time.Duration
	AllowedSubjects []string
	Handler         HandlerFunc
}

// Registry manages metadata for the currently registered operations.
type Registry struct {
	definitions map[string]Definition
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		definitions: map[string]Definition{},
	}
}

// Register 注册一个 operation 定义。
func (r *Registry) Register(def Definition) {
	r.definitions[def.Operation] = def
}

// Resolve looks up metadata by normalized operation name.
func (r *Registry) Resolve(operation string) (Definition, bool) {
	definition, ok := r.definitions[operation]
	return definition, ok
}
