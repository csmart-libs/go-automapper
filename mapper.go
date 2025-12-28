// Package automapper provides high-performance object-to-object mapping for Go,
//
// Key features:
//   - Automatic mapping between structs based on field name matching
//   - Support for nested struct mapping and slice/array mapping
//   - Custom type converters for type-to-type transformations
//   - Custom value resolvers for field-level transformations
//   - Flattening/unflattening support
//   - High performance through reflection caching
//
// Basic usage:
//
//	mapper := automapper.New()
//	mapper.CreateMap[Source, Dest]()
//	dest, err := automapper.Map[Dest](mapper, source)
package automapper

import (
	"reflect"
	"sync"
)

// Mapper is the main interface for object-to-object mapping.
// It provides methods to configure mappings and perform mapping operations.
type Mapper struct {
	config *MapperConfiguration
}

// MapperConfiguration holds all mapping configurations.
type MapperConfiguration struct {
	mu           sync.RWMutex
	typeMaps     map[typeMapKey]*TypeMap
	typeCache    *typeCache
	converters   map[typeMapKey]TypeConverter
	allowNilColl bool

	// Optimization settings
	optLevel      OptimizationLevel
	useUnsafe     bool
	optimizedMaps map[typeMapKey]*TypeMapOptimized
}

// typeMapKey uniquely identifies a source-destination type pair.
type typeMapKey struct {
	srcType  reflect.Type
	destType reflect.Type
}

// TypeMap represents the mapping configuration between two types.
type TypeMap struct {
	srcType      reflect.Type
	destType     reflect.Type
	memberMaps   []*MemberMap
	customMapper CustomMapperFunc
	beforeMap    []BeforeAfterMapFunc
	afterMap     []BeforeAfterMapFunc
	ignoreFields map[string]bool
}

// MemberMap represents the mapping configuration for a single member/field.
type MemberMap struct {
	destField     string
	destFieldIdx  []int
	srcField      string
	srcFieldIdx   []int
	resolver      ValueResolver
	converter     TypeConverter
	condition     ConditionFunc
	ignore        bool
	useFlattening bool
	flattenPath   []string
}

// TypeConverter is a function that converts from one type to another.
type TypeConverter func(src any, destType reflect.Type) (any, error)

// ValueResolver is a function that resolves a value for a destination field.
type ValueResolver func(src any, dest any) (any, error)

// CustomMapperFunc is a function that performs custom mapping between types.
type CustomMapperFunc func(src any, dest any) error

// BeforeAfterMapFunc is a function called before or after mapping.
type BeforeAfterMapFunc func(src any, dest any) error

// ConditionFunc determines if a member should be mapped.
type ConditionFunc func(src any) bool

// New creates a new Mapper with default configuration.
func New() *Mapper {
	return &Mapper{
		config: &MapperConfiguration{
			typeMaps:      make(map[typeMapKey]*TypeMap),
			typeCache:     newTypeCache(),
			converters:    make(map[typeMapKey]TypeConverter),
			optimizedMaps: make(map[typeMapKey]*TypeMapOptimized),
		},
	}
}

// NewWithConfig creates a new Mapper with custom configuration options.
func NewWithConfig(opts ...ConfigOption) *Mapper {
	m := New()
	for _, opt := range opts {
		opt(m.config)
	}
	return m
}

// ConfigOption is a function that configures the mapper.
type ConfigOption func(*MapperConfiguration)

// WithAllowNullCollections allows null collections in mapping output.
func WithAllowNullCollections() ConfigOption {
	return func(c *MapperConfiguration) {
		c.allowNilColl = true
	}
}

// WithOptimizationLevel sets the optimization level for the mapper.
func WithOptimizationLevel(level OptimizationLevel) ConfigOption {
	return func(c *MapperConfiguration) {
		c.optLevel = level
		if level >= OptimizationUnsafe {
			c.useUnsafe = true
		}
	}
}

// WithUnsafeOptimizations enables unsafe pointer optimizations for primitive types.
// This provides significant performance improvements but uses unsafe operations.
// Only use this when you understand the implications of unsafe code.
func WithUnsafeOptimizations() ConfigOption {
	return func(c *MapperConfiguration) {
		c.useUnsafe = true
		if c.optLevel < OptimizationUnsafe {
			c.optLevel = OptimizationUnsafe
		}
	}
}

// WithPooling is a configuration option placeholder for future object pooling support.
// Currently, this option only sets the optimization level but does not enable actual pooling.
// It is kept for API compatibility and future implementation.
func WithPooling() ConfigOption {
	return func(c *MapperConfiguration) {
		if c.optLevel < OptimizationPooled {
			c.optLevel = OptimizationPooled
		}
	}
}

// WithSpecializedMappers enables pre-compiled specialized mappers for primitive-only structs.
func WithSpecializedMappers() ConfigOption {
	return func(c *MapperConfiguration) {
		c.optLevel = OptimizationSpecialized
		c.useUnsafe = true
	}
}

// CreateMap creates a mapping configuration between source and destination types.
// Returns a TypeMapBuilder for further configuration.
func CreateMap[TSrc, TDest any](m *Mapper) *TypeMapBuilder[TSrc, TDest] {
	var src TSrc
	var dest TDest
	srcType := reflect.TypeOf(src)
	destType := reflect.TypeOf(dest)

	// Handle pointer types
	if srcType.Kind() == reflect.Ptr {
		srcType = srcType.Elem()
	}
	if destType.Kind() == reflect.Ptr {
		destType = destType.Elem()
	}

	key := typeMapKey{srcType: srcType, destType: destType}

	m.config.mu.Lock()
	defer m.config.mu.Unlock()

	tm := &TypeMap{
		srcType:      srcType,
		destType:     destType,
		memberMaps:   make([]*MemberMap, 0),
		ignoreFields: make(map[string]bool),
	}

	// Auto-configure member maps based on field matching
	tm.autoConfigureMembers(m.config.typeCache)

	m.config.typeMaps[key] = tm

	// Compile optimized version if optimization is enabled
	if m.config.optLevel > OptimizationNone {
		optMap := compileOptimizedTypeMap(tm, m.config.optLevel)
		m.config.optimizedMaps[key] = optMap
	}

	return &TypeMapBuilder[TSrc, TDest]{
		mapper:  m,
		typeMap: tm,
	}
}

// autoConfigureMembers automatically configures member mappings based on field names.
func (tm *TypeMap) autoConfigureMembers(cache *typeCache) {
	destInfo := cache.getTypeInfo(tm.destType)

	for _, destField := range destInfo.fields {
		mm := tm.findSourceMember(destField, cache)
		if mm != nil {
			tm.memberMaps = append(tm.memberMaps, mm)
		}
	}
}

// findSourceMember finds a matching source member for a destination field.
func (tm *TypeMap) findSourceMember(destField *fieldInfo, cache *typeCache) *MemberMap {
	srcInfo := cache.getTypeInfo(tm.srcType)

	// Direct name match
	if srcField, ok := srcInfo.fieldsByName[destField.name]; ok {
		return &MemberMap{
			destField:    destField.name,
			destFieldIdx: destField.index,
			srcField:     srcField.name,
			srcFieldIdx:  srcField.index,
		}
	}

	// Try flattening: CustomerName -> Customer.Name
	flattenPath := splitPascalCase(destField.name)
	if len(flattenPath) > 1 {
		if mm := tm.tryFlattenMatch(flattenPath, srcInfo, destField, cache); mm != nil {
			return mm
		}
	}

	return nil
}

// tryFlattenMatch attempts to match a flattened destination field to nested source fields.
func (tm *TypeMap) tryFlattenMatch(path []string, _ *typeInfo, destField *fieldInfo, cache *typeCache) *MemberMap {
	currentType := tm.srcType
	var indices []int

	for i, part := range path {
		info := cache.getTypeInfo(currentType)
		field, ok := info.fieldsByName[part]
		if !ok {
			return nil
		}
		indices = append(indices, field.index...)

		if i < len(path)-1 {
			// Navigate to nested type
			fieldType := field.fieldType
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() != reflect.Struct {
				return nil
			}
			currentType = fieldType
		}
	}

	return &MemberMap{
		destField:     destField.name,
		destFieldIdx:  destField.index,
		srcField:      path[0],
		srcFieldIdx:   indices,
		useFlattening: true,
		flattenPath:   path,
	}
}
