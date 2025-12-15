// Package symbols implements symbol table parsing and caching for TwinCAT 3.
package symbols

import (
	"sync"
)

// TypeRegistry holds registered custom type definitions for automatic struct parsing.
type TypeRegistry struct {
	mu    sync.RWMutex
	types map[string]TypeInfo
}

// NewTypeRegistry creates a new type registry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types: make(map[string]TypeInfo),
	}
}

// Register adds or updates a type definition.
func (r *TypeRegistry) Register(typeName string, typeInfo TypeInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types[typeName] = typeInfo
}

// Get retrieves a type definition by name.
func (r *TypeRegistry) Get(typeName string) (TypeInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.types[typeName]
	return info, ok
}

// Has checks if a type is registered.
func (r *TypeRegistry) Has(typeName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.types[typeName]
	return ok
}

// List returns all registered type names.
func (r *TypeRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.types))
	for name := range r.types {
		names = append(names, name)
	}
	return names
}
