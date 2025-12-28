package automapper

import (
	"fmt"
	"reflect"
)

// MappingError represents an error that occurred during mapping.
type MappingError struct {
	Message    string
	SrcType    reflect.Type
	DestType   reflect.Type
	FieldName  string
	InnerError error
}

func (e *MappingError) Error() string {
	if e.FieldName != "" {
		return fmt.Sprintf("mapping error for field '%s' (%v -> %v): %s",
			e.FieldName, e.SrcType, e.DestType, e.Message)
	}
	if e.SrcType != nil && e.DestType != nil {
		return fmt.Sprintf("mapping error (%v -> %v): %s", e.SrcType, e.DestType, e.Message)
	}
	return fmt.Sprintf("mapping error: %s", e.Message)
}

func (e *MappingError) Unwrap() error {
	return e.InnerError
}

// Map performs mapping from source to a new destination instance.
func Map[TDest any](m *Mapper, src any) (TDest, error) {
	var dest TDest
	destVal := reflect.ValueOf(&dest).Elem()

	err := m.mapValue(reflect.ValueOf(src), destVal)
	if err != nil {
		return dest, err
	}

	return dest, nil
}

// MapTo performs mapping from source to an existing destination instance.
func MapTo[TDest any](m *Mapper, src any, dest *TDest) error {
	destVal := reflect.ValueOf(dest).Elem()
	return m.mapValue(reflect.ValueOf(src), destVal)
}

// MapSlice maps a slice of source objects to a slice of destination objects.
func MapSlice[TSrc, TDest any](m *Mapper, src []TSrc) ([]TDest, error) {
	if src == nil {
		if m.config.allowNilColl {
			return nil, nil
		}
		return []TDest{}, nil
	}

	result := make([]TDest, len(src))
	for i, s := range src {
		dest, err := Map[TDest](m, s)
		if err != nil {
			return nil, &MappingError{
				Message:    fmt.Sprintf("error mapping element at index %d", i),
				InnerError: err,
			}
		}
		result[i] = dest
	}
	return result, nil
}

// mapValue is the core mapping function that handles all type mappings.
func (m *Mapper) mapValue(srcVal, destVal reflect.Value) error {
	// Handle nil source
	if !srcVal.IsValid() {
		return nil
	}

	// Dereference pointers
	srcVal = derefValue(srcVal)
	if !srcVal.IsValid() {
		return nil
	}

	srcType := srcVal.Type()
	destType := destVal.Type()
	if destType.Kind() == reflect.Ptr {
		if destVal.IsNil() {
			destVal.Set(reflect.New(destType.Elem()))
		}
		destVal = destVal.Elem()
		destType = destType.Elem()
	}

	// Check for type converter
	key := typeMapKey{srcType: srcType, destType: destType}
	m.config.mu.RLock()
	converter, hasConverter := m.config.converters[key]
	m.config.mu.RUnlock()

	if hasConverter {
		result, err := converter(srcVal.Interface(), destType)
		if err != nil {
			return err
		}
		destVal.Set(reflect.ValueOf(result))
		return nil
	}

	// Handle different kinds
	switch srcType.Kind() {
	case reflect.Struct:
		return m.mapStruct(srcVal, destVal, srcType, destType)
	case reflect.Slice, reflect.Array:
		return m.mapSlice(srcVal, destVal, srcType, destType)
	case reflect.Map:
		return m.mapMap(srcVal, destVal, srcType, destType)
	default:
		// Direct assignment for compatible types
		if srcType.AssignableTo(destType) {
			destVal.Set(srcVal)
			return nil
		}
		if srcType.ConvertibleTo(destType) {
			destVal.Set(srcVal.Convert(destType))
			return nil
		}
		return &MappingError{
			Message:  "incompatible types",
			SrcType:  srcType,
			DestType: destType,
		}
	}
}

// mapStruct maps a struct from source to destination.
func (m *Mapper) mapStruct(srcVal, destVal reflect.Value, srcType, destType reflect.Type) error {
	key := typeMapKey{srcType: srcType, destType: destType}

	m.config.mu.RLock()
	typeMap, exists := m.config.typeMaps[key]
	optMap := m.config.optimizedMaps[key]
	optLevel := m.config.optLevel
	m.config.mu.RUnlock()

	if !exists {
		// Auto-create mapping if not exists
		typeMap = m.autoCreateTypeMap(srcType, destType)
	}

	// Use optimized path if available and optimization is enabled
	if optLevel > OptimizationNone && optMap != nil && optMap.compiled {
		return m.mapStructOptimized(srcVal, destVal, optMap)
	}

	// Standard mapping path
	return m.mapStructStandard(srcVal, destVal, typeMap)
}

// mapStructStandard performs standard reflection-based struct mapping.
func (m *Mapper) mapStructStandard(srcVal, destVal reflect.Value, typeMap *TypeMap) error {
	// Execute before map functions
	for _, beforeFn := range typeMap.beforeMap {
		if err := beforeFn(srcVal.Interface(), destVal.Addr().Interface()); err != nil {
			return err
		}
	}

	// Use custom mapper if defined
	if typeMap.customMapper != nil {
		return typeMap.customMapper(srcVal.Interface(), destVal.Addr().Interface())
	}

	// Map each member
	for _, mm := range typeMap.memberMaps {
		if err := m.mapMember(srcVal, destVal, mm); err != nil {
			return err
		}
	}

	// Execute after map functions
	for _, afterFn := range typeMap.afterMap {
		if err := afterFn(srcVal.Interface(), destVal.Addr().Interface()); err != nil {
			return err
		}
	}

	return nil
}

// mapMember maps a single member from source to destination.
func (m *Mapper) mapMember(srcVal, destVal reflect.Value, mm *MemberMap) error {
	// Check if ignored
	if mm.ignore {
		return nil
	}

	// Check condition
	if mm.condition != nil && !mm.condition(srcVal.Interface()) {
		return nil
	}

	// Get destination field
	destField := destVal.FieldByIndex(mm.destFieldIdx)
	if !destField.CanSet() {
		return nil
	}

	var srcValue reflect.Value

	// Use value resolver if defined
	if mm.resolver != nil {
		result, err := mm.resolver(srcVal.Interface(), destVal.Interface())
		if err != nil {
			return &MappingError{
				Message:    "resolver error",
				FieldName:  mm.destField,
				InnerError: err,
			}
		}
		srcValue = reflect.ValueOf(result)
	} else if len(mm.srcFieldIdx) > 0 {
		// Get source field value using pre-computed index
		srcValue = getNestedField(srcVal, mm.srcFieldIdx)
	} else if mm.srcField != "" {
		// Fallback: look up source field by name (for MapFrom without pre-computed index)
		srcValue = srcVal.FieldByName(mm.srcField)
	} else {
		return nil
	}

	if !srcValue.IsValid() {
		return nil
	}

	// Apply converter if defined
	if mm.converter != nil {
		result, err := mm.converter(srcValue.Interface(), destField.Type())
		if err != nil {
			return &MappingError{
				Message:    "converter error",
				FieldName:  mm.destField,
				InnerError: err,
			}
		}
		srcValue = reflect.ValueOf(result)
	}

	// Perform the assignment
	return m.assignValue(srcValue, destField)
}

// assignValue assigns a source value to a destination field.
func (m *Mapper) assignValue(srcVal reflect.Value, destVal reflect.Value) error {
	srcVal = derefValue(srcVal)
	if !srcVal.IsValid() {
		return nil
	}

	srcType := srcVal.Type()
	destType := destVal.Type()

	// Handle pointer destination
	if destType.Kind() == reflect.Ptr {
		if !srcVal.IsValid() || (srcVal.Kind() == reflect.Ptr && srcVal.IsNil()) {
			return nil
		}
		if destVal.IsNil() {
			destVal.Set(reflect.New(destType.Elem()))
		}
		return m.assignValue(srcVal, destVal.Elem())
	}

	// Check for registered type converter
	key := typeMapKey{srcType: srcType, destType: destType}
	m.config.mu.RLock()
	converter, hasConverter := m.config.converters[key]
	m.config.mu.RUnlock()

	if hasConverter {
		result, err := converter(srcVal.Interface(), destType)
		if err != nil {
			return err
		}
		destVal.Set(reflect.ValueOf(result))
		return nil
	}

	// Direct assignment
	if srcType.AssignableTo(destType) {
		destVal.Set(srcVal)
		return nil
	}

	// Type conversion
	if srcType.ConvertibleTo(destType) {
		destVal.Set(srcVal.Convert(destType))
		return nil
	}

	// Nested mapping for structs
	if srcType.Kind() == reflect.Struct && destType.Kind() == reflect.Struct {
		return m.mapValue(srcVal, destVal)
	}

	// Slice mapping
	if srcType.Kind() == reflect.Slice && destType.Kind() == reflect.Slice {
		return m.mapSlice(srcVal, destVal, srcType, destType)
	}

	return &MappingError{
		Message:  "cannot assign value",
		SrcType:  srcType,
		DestType: destType,
	}
}

// mapSlice maps a slice from source to destination.
func (m *Mapper) mapSlice(srcVal, destVal reflect.Value, _, destType reflect.Type) error {
	if srcVal.IsNil() {
		if m.config.allowNilColl {
			destVal.Set(reflect.Zero(destType))
		} else {
			destVal.Set(reflect.MakeSlice(destType, 0, 0))
		}
		return nil
	}

	srcLen := srcVal.Len()
	destSlice := reflect.MakeSlice(destType, srcLen, srcLen)
	destElemType := destType.Elem()

	for i := 0; i < srcLen; i++ {
		srcElem := srcVal.Index(i)
		destElem := destSlice.Index(i)

		if destElemType.Kind() == reflect.Ptr {
			destElem.Set(reflect.New(destElemType.Elem()))
			if err := m.mapValue(srcElem, destElem.Elem()); err != nil {
				return &MappingError{
					Message:    fmt.Sprintf("error mapping slice element at index %d", i),
					InnerError: err,
				}
			}
		} else {
			if err := m.mapValue(srcElem, destElem); err != nil {
				return &MappingError{
					Message:    fmt.Sprintf("error mapping slice element at index %d", i),
					InnerError: err,
				}
			}
		}
	}

	destVal.Set(destSlice)
	return nil
}

// mapMap maps a map from source to destination.
func (m *Mapper) mapMap(srcVal, destVal reflect.Value, _, destType reflect.Type) error {
	if srcVal.IsNil() {
		if m.config.allowNilColl {
			destVal.Set(reflect.Zero(destType))
		} else {
			destVal.Set(reflect.MakeMap(destType))
		}
		return nil
	}

	destMap := reflect.MakeMapWithSize(destType, srcVal.Len())
	destKeyType := destType.Key()
	destValType := destType.Elem()

	iter := srcVal.MapRange()
	for iter.Next() {
		srcKey := iter.Key()
		srcMapVal := iter.Value()

		// Convert key
		destKey := reflect.New(destKeyType).Elem()
		if srcKey.Type().AssignableTo(destKeyType) {
			destKey.Set(srcKey)
		} else if srcKey.Type().ConvertibleTo(destKeyType) {
			destKey.Set(srcKey.Convert(destKeyType))
		} else {
			return &MappingError{
				Message:  "cannot convert map key",
				SrcType:  srcKey.Type(),
				DestType: destKeyType,
			}
		}

		// Convert value
		destMapVal := reflect.New(destValType).Elem()
		if err := m.assignValue(srcMapVal, destMapVal); err != nil {
			return err
		}

		destMap.SetMapIndex(destKey, destMapVal)
	}

	destVal.Set(destMap)
	return nil
}

// autoCreateTypeMap creates a type map automatically for unmapped types.
func (m *Mapper) autoCreateTypeMap(srcType, destType reflect.Type) *TypeMap {
	key := typeMapKey{srcType: srcType, destType: destType}

	m.config.mu.Lock()
	defer m.config.mu.Unlock()

	// Double-check after acquiring lock
	if tm, exists := m.config.typeMaps[key]; exists {
		return tm
	}

	tm := &TypeMap{
		srcType:      srcType,
		destType:     destType,
		memberMaps:   make([]*MemberMap, 0),
		ignoreFields: make(map[string]bool),
	}

	tm.autoConfigureMembers(m.config.typeCache)
	m.config.typeMaps[key] = tm

	// Compile optimized version if optimization is enabled
	if m.config.optLevel > OptimizationNone {
		optMap := compileOptimizedTypeMap(tm, m.config.optLevel)
		m.config.optimizedMaps[key] = optMap
	}

	return tm
}

// derefValue dereferences a pointer value.
func derefValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// getNestedField gets a field value using nested indices.
func getNestedField(v reflect.Value, indices []int) reflect.Value {
	v = derefValue(v)
	if !v.IsValid() {
		return reflect.Value{}
	}

	for _, idx := range indices {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return reflect.Value{}
		}
		if idx >= v.NumField() {
			return reflect.Value{}
		}
		v = v.Field(idx)
	}

	return v
}
