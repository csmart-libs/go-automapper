package automapper

import (
	"reflect"
	"unsafe"
)

// OptimizationLevel represents the level of optimization to apply.
type OptimizationLevel int

const (
	// OptimizationNone uses standard reflection-based mapping (default).
	OptimizationNone OptimizationLevel = iota
	// OptimizationPooled is a placeholder for future pooling support (currently same as None).
	OptimizationPooled
	// OptimizationUnsafe uses unsafe pointer operations for primitive types.
	OptimizationUnsafe
	// OptimizationSpecialized uses pre-compiled specialized mappers.
	OptimizationSpecialized
)

// SpecializedMapper is a pre-compiled optimized mapping function.
type SpecializedMapper func(src, dest reflect.Value) error

// MemberMapOptimized extends MemberMap with optimization metadata.
type MemberMapOptimized struct {
	*MemberMap
	srcKind      reflect.Kind
	destKind     reflect.Kind
	isPrimitive  bool
	srcOffset    uintptr
	destOffset   uintptr
	fieldSize    uintptr
	directAssign bool
}

// isPrimitiveKind checks if a kind is a primitive type suitable for unsafe optimization.
func isPrimitiveKind(k reflect.Kind) bool {
	switch k {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	}
	return false
}

// TypeMapOptimized extends TypeMap with optimization metadata.
type TypeMapOptimized struct {
	*TypeMap
	optimizedMembers []*MemberMapOptimized
	specializedFn    SpecializedMapper
	allPrimitive     bool
	hasCustomLogic   bool
	compiled         bool
}

// compileOptimizedTypeMap creates an optimized version of TypeMap.
func compileOptimizedTypeMap(tm *TypeMap, level OptimizationLevel) *TypeMapOptimized {
	opt := &TypeMapOptimized{
		TypeMap:          tm,
		optimizedMembers: make([]*MemberMapOptimized, len(tm.memberMaps)),
		allPrimitive:     true,
		hasCustomLogic:   tm.customMapper != nil || len(tm.beforeMap) > 0 || len(tm.afterMap) > 0,
	}

	for i, mm := range tm.memberMaps {
		optMm := &MemberMapOptimized{
			MemberMap: mm,
		}

		// Get source and dest field types
		if len(mm.srcFieldIdx) == 1 && len(mm.destFieldIdx) == 1 {
			srcField := tm.srcType.Field(mm.srcFieldIdx[0])
			destField := tm.destType.Field(mm.destFieldIdx[0])

			optMm.srcKind = srcField.Type.Kind()
			optMm.destKind = destField.Type.Kind()
			optMm.isPrimitive = isPrimitiveKind(optMm.srcKind) && isPrimitiveKind(optMm.destKind)
			optMm.srcOffset = srcField.Offset
			optMm.destOffset = destField.Offset
			optMm.fieldSize = srcField.Type.Size()
			optMm.directAssign = srcField.Type == destField.Type && optMm.isPrimitive

			if !optMm.isPrimitive {
				opt.allPrimitive = false
			}
		} else {
			opt.allPrimitive = false
		}

		// Check for custom logic
		if mm.resolver != nil || mm.converter != nil || mm.condition != nil {
			opt.hasCustomLogic = true
			optMm.isPrimitive = false
		}

		opt.optimizedMembers[i] = optMm
	}

	// Compile specialized mapper for all-primitive structs
	if level >= OptimizationSpecialized && opt.allPrimitive && !opt.hasCustomLogic {
		opt.specializedFn = compileSpecializedMapper(opt)
	}

	opt.compiled = true
	return opt
}

// compileSpecializedMapper creates a specialized mapping function for primitive-only structs.
func compileSpecializedMapper(opt *TypeMapOptimized) SpecializedMapper {
	members := opt.optimizedMembers

	return func(src, dest reflect.Value) error {
		for _, mm := range members {
			if mm.ignore {
				continue
			}
			// Direct field copy using pre-computed indices
			destField := dest.Field(mm.destFieldIdx[0])
			srcField := src.Field(mm.srcFieldIdx[0])
			destField.Set(srcField)
		}
		return nil
	}
}

// unsafeCopyField copies a field value using unsafe pointers.
// This is only safe for primitive types with the same type.
func unsafeCopyField(srcPtr, destPtr unsafe.Pointer, srcOffset, destOffset, size uintptr) {
	src := unsafe.Add(srcPtr, srcOffset)
	dest := unsafe.Add(destPtr, destOffset)

	// Copy bytes directly
	switch size {
	case 1:
		*(*uint8)(dest) = *(*uint8)(src)
	case 2:
		*(*uint16)(dest) = *(*uint16)(src)
	case 4:
		*(*uint32)(dest) = *(*uint32)(src)
	case 8:
		*(*uint64)(dest) = *(*uint64)(src)
	case 16:
		// For strings (which are 16 bytes: pointer + length)
		*(*[16]byte)(dest) = *(*[16]byte)(src)
	default:
		// Fallback for other sizes - copy byte by byte
		srcBytes := unsafe.Slice((*byte)(src), size)
		destBytes := unsafe.Slice((*byte)(dest), size)
		copy(destBytes, srcBytes)
	}
}

// mapMemberUnsafe maps a member using unsafe pointer operations for primitives.
func (m *Mapper) mapMemberUnsafe(srcVal, destVal reflect.Value, mm *MemberMapOptimized) error {
	if mm.ignore {
		return nil
	}

	// Fast path for direct primitive assignment (only if both values are addressable)
	if mm.directAssign && mm.isPrimitive && len(mm.srcFieldIdx) == 1 &&
		srcVal.CanAddr() && destVal.CanAddr() {
		srcPtr := unsafe.Pointer(srcVal.UnsafeAddr())
		destPtr := unsafe.Pointer(destVal.UnsafeAddr())
		unsafeCopyField(srcPtr, destPtr, mm.srcOffset, mm.destOffset, mm.fieldSize)
		return nil
	}

	// Fallback to standard mapping
	return m.mapMember(srcVal, destVal, mm.MemberMap)
}

// mapStructOptimized maps a struct using optimizations based on level.
func (m *Mapper) mapStructOptimized(srcVal, destVal reflect.Value, typeMap *TypeMapOptimized) error {
	// Always check the original TypeMap for hooks (they may be added after compilation)
	tm := typeMap.TypeMap

	// Execute before map functions (requires interface boxing)
	if len(tm.beforeMap) > 0 {
		srcIface := srcVal.Interface()
		destIface := destVal.Addr().Interface()
		for _, beforeFn := range tm.beforeMap {
			if err := beforeFn(srcIface, destIface); err != nil {
				return err
			}
		}
	}

	// Use custom mapper if defined
	if tm.customMapper != nil {
		return tm.customMapper(srcVal.Interface(), destVal.Addr().Interface())
	}

	// Use specialized mapper if available and no custom logic was added later
	hasHooks := len(tm.beforeMap) > 0 || len(tm.afterMap) > 0 || tm.customMapper != nil
	if typeMap.specializedFn != nil && !hasHooks {
		if err := typeMap.specializedFn(srcVal, destVal); err != nil {
			return err
		}
	} else if m.config.useUnsafe {
		// Map each member with unsafe optimizations
		for _, mm := range typeMap.optimizedMembers {
			if err := m.mapMemberUnsafe(srcVal, destVal, mm); err != nil {
				return err
			}
		}
	} else {
		// Standard member mapping
		for _, mm := range tm.memberMaps {
			if err := m.mapMember(srcVal, destVal, mm); err != nil {
				return err
			}
		}
	}

	// Execute after map functions
	if len(tm.afterMap) > 0 {
		srcIface := srcVal.Interface()
		destIface := destVal.Addr().Interface()
		for _, afterFn := range tm.afterMap {
			if err := afterFn(srcIface, destIface); err != nil {
				return err
			}
		}
	}

	return nil
}
