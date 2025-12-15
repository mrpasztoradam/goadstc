package symbols

import (
	"fmt"
	"strings"
	"sync"
)

// Table manages a cached symbol table with concurrent access.
type Table struct {
	symbols map[string]*Symbol
	mu      sync.RWMutex
	loaded  bool
}

// NewTable creates a new empty symbol table.
func NewTable() *Table {
	return &Table{
		symbols: make(map[string]*Symbol),
	}
}

// Load parses and loads symbols from raw upload data.
func (t *Table) Load(data []byte) error {
	symbols, err := ParseSymbolTable(data)
	if err != nil {
		return fmt.Errorf("parse symbol table: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.symbols = make(map[string]*Symbol, len(symbols))

	for i := range symbols {
		sym := &symbols[i]
		t.symbols[sym.Name] = sym
	}

	t.loaded = true
	return nil
}

// Get retrieves a symbol by name.
func (t *Table) Get(name string) (*Symbol, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.loaded {
		return nil, fmt.Errorf("symbol table not loaded")
	}

	sym, exists := t.symbols[name]
	if !exists {
		return nil, fmt.Errorf("symbol %q not found", name)
	}

	return sym, nil
}

// List returns all symbols in the table.
func (t *Table) List() ([]*Symbol, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.loaded {
		return nil, fmt.Errorf("symbol table not loaded")
	}

	symbols := make([]*Symbol, 0, len(t.symbols))
	for _, sym := range t.symbols {
		symbols = append(symbols, sym)
	}

	return symbols, nil
}

// IsLoaded returns true if the symbol table has been loaded.
func (t *Table) IsLoaded() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.loaded
}

// Find searches for symbols by name pattern.
func (t *Table) Find(pattern string) ([]*Symbol, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.loaded {
		return nil, fmt.Errorf("symbol table not loaded")
	}

	pattern = strings.ToLower(pattern)
	var matches []*Symbol

	for name, sym := range t.symbols {
		if strings.Contains(strings.ToLower(name), pattern) {
			matches = append(matches, sym)
		}
	}

	return matches, nil
}
