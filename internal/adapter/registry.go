package adapter

import "time"

// Definition stores the base metadata for one operation.
type Definition struct {
	Operation       string
	Platform        string
	Mutating        bool
	DefaultTimeout  time.Duration
	AllowedSubjects []string
}

// Registry manages metadata for the currently registered operations.
type Registry struct {
	definitions map[string]Definition
}

// NewRegistry creates a registry and registers the current MVP operations.
func NewRegistry() *Registry {
	registry := &Registry{
		definitions: map[string]Definition{},
	}

	registry.register(Definition{
		Operation:       "feishu.calendar.event.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
	})
	registry.register(Definition{
		Operation:       "feishu.calendar.event.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
	})
	registry.register(Definition{
		Operation:       "feishu.docs.document.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
	})
	registry.register(Definition{
		Operation:       "feishu.wiki.space.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
	})
	registry.register(Definition{
		Operation:       "feishu.wiki.node.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
	})
	registry.register(Definition{
		Operation:       "feishu.wiki.node.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
	})
	registry.register(Definition{
		Operation:       "feishu.docs.document.append_blocks",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
	})
	registry.register(Definition{
		Operation:       "feishu.docs.document.get_raw_content",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
	})
	registry.register(Definition{
		Operation:       "feishu.contact.user.get",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
	})
	registry.register(Definition{
		Operation:       "notion.page.create",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
	})
	registry.register(Definition{
		Operation:       "notion.page.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
	})
	registry.register(Definition{
		Operation:       "notion.block.append",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
	})
	registry.register(Definition{
		Operation:       "notion.user.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
	})

	return registry
}

func (r *Registry) register(def Definition) {
	r.definitions[def.Operation] = def
}

// Resolve looks up metadata by normalized operation name.
func (r *Registry) Resolve(operation string) (Definition, bool) {
	definition, ok := r.definitions[operation]
	return definition, ok
}
