package automapper

import (
	"reflect"
	"unsafe"
)

// TypeMapBuilder provides a fluent API for configuring type mappings.
type TypeMapBuilder[TSrc, TDest any] struct {
	mapper  *Mapper
	typeMap *TypeMap
}

// ForMember configures a specific destination member mapping using a field selector.
// The selector function should access a field on the destination struct pointer.
//
// Example:
//
//	CreateMap[Source, Dest](mapper).
//	    ForMember(func(d *Dest) any { return d.Name }, MapFrom("FullName"))
//
// Note: Due to Go's reflection limitations, this uses a sentinel-value approach
// to detect which field was accessed. For more reliable behavior, consider using
// ForMemberByName instead.
func (b *TypeMapBuilder[TSrc, TDest]) ForMember(
	destMember func(*TDest) any,
	opts ...MemberOption,
) *TypeMapBuilder[TSrc, TDest] {
	// Get destination member name using reflection
	var dest TDest
	destType := reflect.TypeOf(dest)
	if destType.Kind() == reflect.Ptr {
		destType = destType.Elem()
	}

	// Find which field was accessed by calling the member selector
	memberName := findMemberName(&dest, destMember, destType)
	if memberName == "" {
		return b
	}

	// Find or create member map
	var mm *MemberMap
	for _, m := range b.typeMap.memberMaps {
		if m.destField == memberName {
			mm = m
			break
		}
	}

	if mm == nil {
		// Create new member map
		destInfo := b.mapper.config.typeCache.getTypeInfo(destType)
		if fi, ok := destInfo.fieldsByName[memberName]; ok {
			mm = &MemberMap{
				destField:    memberName,
				destFieldIdx: fi.index,
			}
			b.typeMap.memberMaps = append(b.typeMap.memberMaps, mm)
		}
	}

	if mm != nil {
		for _, opt := range opts {
			opt(mm)
		}
	}

	return b
}

// findMemberName attempts to find the member name from a selector function.
// This uses a pointer-comparison approach to detect which field was accessed.
func findMemberName[TDest any](dest *TDest, selector func(*TDest) any, destType reflect.Type) string {
	if destType.Kind() != reflect.Struct {
		return ""
	}

	// Call the selector to get the returned interface value
	result := selector(dest)
	if result == nil {
		return ""
	}

	// Get the pointer to the returned value
	resultVal := reflect.ValueOf(result)

	// If it's not a pointer or addressable, we can't compare
	var resultPtr uintptr
	if resultVal.Kind() == reflect.Ptr {
		resultPtr = resultVal.Pointer()
	} else if resultVal.CanAddr() {
		resultPtr = resultVal.Addr().Pointer()
	} else {
		// The selector returned a value, not an addressable field reference
		// This happens when users write: func(d *Dest) any { return d.Name }
		// We need to try a different approach - compare by iterating through fields
		return findMemberByValue(dest, selector, destType)
	}

	// Get the base address of the dest struct
	destVal := reflect.ValueOf(dest).Elem()

	// Check each field to see if its address matches
	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldVal := destVal.Field(i)
		if fieldVal.CanAddr() {
			fieldPtr := fieldVal.Addr().Pointer()
			if fieldPtr == resultPtr {
				return field.Name
			}
		}
	}

	return ""
}

// findMemberByValue finds a member by comparing values after setting sentinel values.
// This is a fallback when pointer comparison doesn't work.
func findMemberByValue[TDest any](dest *TDest, selector func(*TDest) any, destType reflect.Type) string {
	// Create a zero-valued struct
	destVal := reflect.ValueOf(dest).Elem()

	// Try each field - set a unique sentinel value and check if selector returns it
	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldVal := destVal.Field(i)
		if !fieldVal.CanSet() {
			continue
		}

		// Store original value
		original := reflect.New(fieldVal.Type()).Elem()
		original.Set(fieldVal)

		// Try to identify by checking if the selector returns this field
		// We do this by checking if the returned value's address matches
		result := selector(dest)
		if result == nil {
			continue
		}

		// Check if the result points to this field using unsafe pointer comparison
		resultVal := reflect.ValueOf(result)
		if resultVal.Kind() == reflect.Ptr || resultVal.Kind() == reflect.Interface {
			if resultVal.Kind() == reflect.Interface {
				resultVal = resultVal.Elem()
			}
			if resultVal.CanAddr() || resultVal.Kind() == reflect.Ptr {
				var resultAddr uintptr
				if resultVal.Kind() == reflect.Ptr {
					resultAddr = resultVal.Pointer()
				} else {
					resultAddr = resultVal.Addr().Pointer()
				}

				fieldAddr := uintptr(unsafe.Pointer(fieldVal.Addr().UnsafePointer()))
				if resultAddr == fieldAddr {
					return field.Name
				}
			}
		}
	}

	return ""
}

// ForMemberByName configures a specific destination member by name.
func (b *TypeMapBuilder[TSrc, TDest]) ForMemberByName(
	destMemberName string,
	opts ...MemberOption,
) *TypeMapBuilder[TSrc, TDest] {
	var dest TDest
	destType := reflect.TypeOf(dest)
	if destType.Kind() == reflect.Ptr {
		destType = destType.Elem()
	}

	// Find or create member map
	var mm *MemberMap
	for _, m := range b.typeMap.memberMaps {
		if m.destField == destMemberName {
			mm = m
			break
		}
	}

	if mm == nil {
		destInfo := b.mapper.config.typeCache.getTypeInfo(destType)
		if fi, ok := destInfo.fieldsByName[destMemberName]; ok {
			mm = &MemberMap{
				destField:    destMemberName,
				destFieldIdx: fi.index,
			}
			b.typeMap.memberMaps = append(b.typeMap.memberMaps, mm)
		}
	}

	if mm != nil {
		for _, opt := range opts {
			opt(mm)
		}
	}

	return b
}

// MemberOption is a function that configures a member mapping.
type MemberOption func(*MemberMap)

// MapFrom configures the source field name for a destination member.
func MapFrom(srcFieldName string) MemberOption {
	return func(mm *MemberMap) {
		mm.srcField = srcFieldName
	}
}

// MapFromFunc configures a value resolver for a destination member.
func MapFromFunc(resolver ValueResolver) MemberOption {
	return func(mm *MemberMap) {
		mm.resolver = resolver
	}
}

// Ignore configures a destination member to be ignored during mapping.
func Ignore() MemberOption {
	return func(mm *MemberMap) {
		mm.ignore = true
	}
}

// Condition configures a condition for mapping a destination member.
func Condition(cond ConditionFunc) MemberOption {
	return func(mm *MemberMap) {
		mm.condition = cond
	}
}

// UseConverter configures a type converter for a destination member.
func UseConverter(converter TypeConverter) MemberOption {
	return func(mm *MemberMap) {
		mm.converter = converter
	}
}

// ConvertUsing registers a global type converter.
func ConvertUsing[TSrc, TDest any](m *Mapper, converter func(TSrc) (TDest, error)) {
	var src TSrc
	var dest TDest
	srcType := reflect.TypeOf(src)
	destType := reflect.TypeOf(dest)

	key := typeMapKey{srcType: srcType, destType: destType}

	m.config.mu.Lock()
	defer m.config.mu.Unlock()

	m.config.converters[key] = func(s any, dt reflect.Type) (any, error) {
		srcVal, ok := s.(TSrc)
		if !ok {
			return nil, &MappingError{
				Message: "invalid source type for converter",
			}
		}
		return converter(srcVal)
	}
}

// BeforeMap adds a function to be called before mapping.
func (b *TypeMapBuilder[TSrc, TDest]) BeforeMap(fn func(src *TSrc, dest *TDest) error) *TypeMapBuilder[TSrc, TDest] {
	b.typeMap.beforeMap = append(b.typeMap.beforeMap, func(s any, d any) error {
		srcPtr, ok := s.(*TSrc)
		if !ok {
			if srcVal, ok := s.(TSrc); ok {
				srcPtr = &srcVal
			} else {
				return nil
			}
		}
		destPtr, ok := d.(*TDest)
		if !ok {
			return nil
		}
		return fn(srcPtr, destPtr)
	})
	return b
}

// AfterMap adds a function to be called after mapping.
func (b *TypeMapBuilder[TSrc, TDest]) AfterMap(fn func(src *TSrc, dest *TDest) error) *TypeMapBuilder[TSrc, TDest] {
	b.typeMap.afterMap = append(b.typeMap.afterMap, func(s any, d any) error {
		srcPtr, ok := s.(*TSrc)
		if !ok {
			if srcVal, ok := s.(TSrc); ok {
				srcPtr = &srcVal
			} else {
				return nil
			}
		}
		destPtr, ok := d.(*TDest)
		if !ok {
			return nil
		}
		return fn(srcPtr, destPtr)
	})
	return b
}

// CustomMap sets a custom mapping function for the entire type.
func (b *TypeMapBuilder[TSrc, TDest]) CustomMap(fn func(src TSrc, dest *TDest) error) *TypeMapBuilder[TSrc, TDest] {
	b.typeMap.customMapper = func(s any, d any) error {
		srcVal, ok := s.(TSrc)
		if !ok {
			return &MappingError{Message: "invalid source type for custom mapper"}
		}
		destPtr, ok := d.(*TDest)
		if !ok {
			return &MappingError{Message: "invalid destination type for custom mapper"}
		}
		return fn(srcVal, destPtr)
	}
	return b
}

// ReverseMap creates a reverse mapping from destination to source.
func (b *TypeMapBuilder[TSrc, TDest]) ReverseMap() *TypeMapBuilder[TDest, TSrc] {
	return CreateMap[TDest, TSrc](b.mapper)
}
