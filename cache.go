package automapper

import (
	"reflect"
	"sync"
	"unicode"
)

// typeCache caches type information for faster reflection operations.
type typeCache struct {
	mu    sync.RWMutex
	cache map[reflect.Type]*typeInfo
}

// typeInfo holds cached information about a type.
type typeInfo struct {
	typ          reflect.Type
	fields       []*fieldInfo
	fieldsByName map[string]*fieldInfo
}

// fieldInfo holds cached information about a struct field.
type fieldInfo struct {
	name      string
	index     []int
	fieldType reflect.Type
	canSet    bool
}

// newTypeCache creates a new type cache.
func newTypeCache() *typeCache {
	return &typeCache{
		cache: make(map[reflect.Type]*typeInfo),
	}
}

// getTypeInfo retrieves or builds type information for a given type.
func (tc *typeCache) getTypeInfo(t reflect.Type) *typeInfo {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	tc.mu.RLock()
	info, ok := tc.cache[t]
	tc.mu.RUnlock()
	if ok {
		return info
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Double-check after acquiring write lock
	if info, ok = tc.cache[t]; ok {
		return info
	}

	info = tc.buildTypeInfo(t)
	tc.cache[t] = info
	return info
}

// buildTypeInfo builds type information for a struct type.
func (tc *typeCache) buildTypeInfo(t reflect.Type) *typeInfo {
	info := &typeInfo{
		typ:          t,
		fields:       make([]*fieldInfo, 0),
		fieldsByName: make(map[string]*fieldInfo),
	}

	if t.Kind() != reflect.Struct {
		return info
	}

	tc.collectFields(t, nil, info)
	return info
}

// collectFields recursively collects fields from a struct type.
func (tc *typeCache) collectFields(t reflect.Type, index []int, info *typeInfo) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldIdx := append(append([]int{}, index...), i)

		// Handle embedded structs
		if field.Anonymous {
			fieldType := field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() == reflect.Struct {
				tc.collectFields(fieldType, fieldIdx, info)
				continue
			}
		}

		// Only include exported fields
		if !field.IsExported() {
			continue
		}

		fi := &fieldInfo{
			name:      field.Name,
			index:     fieldIdx,
			fieldType: field.Type,
			canSet:    true,
		}
		info.fields = append(info.fields, fi)
		info.fieldsByName[field.Name] = fi
	}
}

// splitPascalCase splits a PascalCase string into individual words.
// Example: "CustomerName" -> ["Customer", "Name"]
func splitPascalCase(s string) []string {
	if len(s) == 0 {
		return nil
	}

	var words []string
	var current []rune

	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		}
		current = append(current, r)
	}

	if len(current) > 0 {
		words = append(words, string(current))
	}

	return words
}
