package automapper

import (
	"testing"
)

// Test types for basic mapping
type SourceBasic struct {
	Name  string
	Age   int
	Email string
}

type DestBasic struct {
	Name  string
	Age   int
	Email string
}

func TestBasicMapping(t *testing.T) {
	mapper := New()
	CreateMap[SourceBasic, DestBasic](mapper)

	src := SourceBasic{
		Name:  "John Doe",
		Age:   30,
		Email: "john@example.com",
	}

	dest, err := Map[DestBasic](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Name != src.Name {
		t.Errorf("Name mismatch: got %s, want %s", dest.Name, src.Name)
	}
	if dest.Age != src.Age {
		t.Errorf("Age mismatch: got %d, want %d", dest.Age, src.Age)
	}
	if dest.Email != src.Email {
		t.Errorf("Email mismatch: got %s, want %s", dest.Email, src.Email)
	}
}

func TestMapTo(t *testing.T) {
	mapper := New()
	CreateMap[SourceBasic, DestBasic](mapper)

	src := SourceBasic{Name: "Jane", Age: 25, Email: "jane@test.com"}
	var dest DestBasic

	err := MapTo(mapper, src, &dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Name != "Jane" {
		t.Errorf("Name mismatch: got %s, want Jane", dest.Name)
	}
}

// Test types for nested mapping
type Address struct {
	Street string
	City   string
	Zip    string
}

type SourceNested struct {
	Name    string
	Address Address
}

type AddressDTO struct {
	Street string
	City   string
	Zip    string
}

type DestNested struct {
	Name    string
	Address AddressDTO
}

func TestNestedMapping(t *testing.T) {
	mapper := New()
	CreateMap[SourceNested, DestNested](mapper)
	CreateMap[Address, AddressDTO](mapper)

	src := SourceNested{
		Name: "John",
		Address: Address{
			Street: "123 Main St",
			City:   "Boston",
			Zip:    "02101",
		},
	}

	dest, err := Map[DestNested](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Name != src.Name {
		t.Errorf("Name mismatch: got %s, want %s", dest.Name, src.Name)
	}
	if dest.Address.Street != src.Address.Street {
		t.Errorf("Street mismatch: got %s, want %s", dest.Address.Street, src.Address.Street)
	}
	if dest.Address.City != src.Address.City {
		t.Errorf("City mismatch: got %s, want %s", dest.Address.City, src.Address.City)
	}
}

// Test types for flattening
type Customer struct {
	Name string
}

type Order struct {
	Total    float64
	Customer Customer
}

type OrderDTO struct {
	Total        float64
	CustomerName string
}

func TestFlattening(t *testing.T) {
	mapper := New()
	CreateMap[Order, OrderDTO](mapper)

	src := Order{
		Total:    99.99,
		Customer: Customer{Name: "Alice"},
	}

	dest, err := Map[OrderDTO](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Total != src.Total {
		t.Errorf("Total mismatch: got %f, want %f", dest.Total, src.Total)
	}
	if dest.CustomerName != src.Customer.Name {
		t.Errorf("CustomerName mismatch: got %s, want %s", dest.CustomerName, src.Customer.Name)
	}
}

// Test slice mapping
type SourceItem struct {
	ID   int
	Name string
}

type DestItem struct {
	ID   int
	Name string
}

func TestSliceMapping(t *testing.T) {
	mapper := New()
	CreateMap[SourceItem, DestItem](mapper)

	src := []SourceItem{
		{ID: 1, Name: "Item 1"},
		{ID: 2, Name: "Item 2"},
		{ID: 3, Name: "Item 3"},
	}

	dest, err := MapSlice[SourceItem, DestItem](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dest) != len(src) {
		t.Fatalf("Length mismatch: got %d, want %d", len(dest), len(src))
	}

	for i, item := range dest {
		if item.ID != src[i].ID {
			t.Errorf("ID mismatch at %d: got %d, want %d", i, item.ID, src[i].ID)
		}
		if item.Name != src[i].Name {
			t.Errorf("Name mismatch at %d: got %s, want %s", i, item.Name, src[i].Name)
		}
	}
}

// Test slice as struct member
type SourceWithSlice struct {
	Name  string
	Items []SourceItem
}

type DestWithSlice struct {
	Name  string
	Items []DestItem
}

func TestSliceInStruct(t *testing.T) {
	mapper := New()
	CreateMap[SourceWithSlice, DestWithSlice](mapper)
	CreateMap[SourceItem, DestItem](mapper)

	src := SourceWithSlice{
		Name: "Container",
		Items: []SourceItem{
			{ID: 1, Name: "A"},
			{ID: 2, Name: "B"},
		},
	}

	dest, err := Map[DestWithSlice](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Name != src.Name {
		t.Errorf("Name mismatch: got %s, want %s", dest.Name, src.Name)
	}
	if len(dest.Items) != 2 {
		t.Fatalf("Items length mismatch: got %d, want 2", len(dest.Items))
	}
}

// Test custom value resolver
func TestValueResolver(t *testing.T) {
	mapper := New()
	CreateMap[SourceBasic, DestBasic](mapper).
		ForMemberByName("Email", MapFromFunc(func(src any, dest any) (any, error) {
			s := src.(SourceBasic)
			return "resolved_" + s.Email, nil
		}))

	src := SourceBasic{Name: "Test", Age: 20, Email: "test@test.com"}

	dest, err := Map[DestBasic](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Email != "resolved_test@test.com" {
		t.Errorf("Email mismatch: got %s, want resolved_test@test.com", dest.Email)
	}
}

// Test ignore field
func TestIgnoreField(t *testing.T) {
	mapper := New()
	CreateMap[SourceBasic, DestBasic](mapper).
		ForMemberByName("Email", Ignore())

	src := SourceBasic{Name: "Test", Age: 20, Email: "test@test.com"}

	dest, err := Map[DestBasic](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Email != "" {
		t.Errorf("Email should be empty, got %s", dest.Email)
	}
	if dest.Name != "Test" {
		t.Errorf("Name mismatch: got %s, want Test", dest.Name)
	}
}

// Test conditional mapping
func TestConditionalMapping(t *testing.T) {
	mapper := New()
	CreateMap[SourceBasic, DestBasic](mapper).
		ForMemberByName("Age", Condition(func(src any) bool {
			s := src.(SourceBasic)
			return s.Age >= 18
		}))

	// Age >= 18, should map
	src1 := SourceBasic{Name: "Adult", Age: 25, Email: "adult@test.com"}
	dest1, err := Map[DestBasic](mapper, src1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest1.Age != 25 {
		t.Errorf("Age should be 25, got %d", dest1.Age)
	}

	// Age < 18, should not map
	src2 := SourceBasic{Name: "Minor", Age: 15, Email: "minor@test.com"}
	dest2, err := Map[DestBasic](mapper, src2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest2.Age != 0 {
		t.Errorf("Age should be 0 (not mapped), got %d", dest2.Age)
	}
}

// Test type converter
func TestTypeConverter(t *testing.T) {
	type SourceWithString struct {
		Value string
	}
	type DestWithInt struct {
		Value int
	}

	mapper := New()
	ConvertUsing(mapper, func(s string) (int, error) {
		if s == "one" {
			return 1, nil
		}
		return 0, nil
	})
	CreateMap[SourceWithString, DestWithInt](mapper)

	src := SourceWithString{Value: "one"}
	dest, err := Map[DestWithInt](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Value != 1 {
		t.Errorf("Value should be 1, got %d", dest.Value)
	}
}

// Test nil slice handling
func TestNilSlice(t *testing.T) {
	mapper := New()
	CreateMap[SourceItem, DestItem](mapper)

	var src []SourceItem = nil
	dest, err := MapSlice[SourceItem, DestItem](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest == nil {
		t.Error("dest should not be nil (empty slice expected)")
	}
	if len(dest) != 0 {
		t.Errorf("dest should be empty, got %d items", len(dest))
	}
}

// Test nil slice with AllowNullCollections
func TestNilSliceAllowed(t *testing.T) {
	mapper := NewWithConfig(WithAllowNullCollections())
	CreateMap[SourceItem, DestItem](mapper)

	var src []SourceItem = nil
	dest, err := MapSlice[SourceItem, DestItem](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest != nil {
		t.Error("dest should be nil when AllowNullCollections is set")
	}
}

// Test pointer fields
type SourceWithPointer struct {
	Name    string
	Address *Address
}

type DestWithPointer struct {
	Name    string
	Address *AddressDTO
}

func TestPointerFields(t *testing.T) {
	mapper := New()
	CreateMap[SourceWithPointer, DestWithPointer](mapper)
	CreateMap[Address, AddressDTO](mapper)

	src := SourceWithPointer{
		Name: "John",
		Address: &Address{
			Street: "123 Main St",
			City:   "Boston",
			Zip:    "02101",
		},
	}

	dest, err := Map[DestWithPointer](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Name != "John" {
		t.Errorf("Name mismatch: got %s, want John", dest.Name)
	}
	if dest.Address == nil {
		t.Fatal("Address should not be nil")
	}
	if dest.Address.City != "Boston" {
		t.Errorf("City mismatch: got %s, want Boston", dest.Address.City)
	}
}

// Test nil pointer field
func TestNilPointerField(t *testing.T) {
	mapper := New()
	CreateMap[SourceWithPointer, DestWithPointer](mapper)
	CreateMap[Address, AddressDTO](mapper)

	src := SourceWithPointer{
		Name:    "John",
		Address: nil,
	}

	dest, err := Map[DestWithPointer](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Name != "John" {
		t.Errorf("Name mismatch: got %s, want John", dest.Name)
	}
	// nil pointer should remain nil or be handled gracefully
}

// Test BeforeMap and AfterMap hooks
func TestBeforeAfterMap(t *testing.T) {
	mapper := New()
	beforeCalled := false
	afterCalled := false

	CreateMap[SourceBasic, DestBasic](mapper).
		BeforeMap(func(src *SourceBasic, dest *DestBasic) error {
			beforeCalled = true
			return nil
		}).
		AfterMap(func(src *SourceBasic, dest *DestBasic) error {
			afterCalled = true
			dest.Email = "modified_" + dest.Email
			return nil
		})

	src := SourceBasic{Name: "Test", Age: 25, Email: "test@test.com"}

	dest, err := Map[DestBasic](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !beforeCalled {
		t.Error("BeforeMap should have been called")
	}
	if !afterCalled {
		t.Error("AfterMap should have been called")
	}
	if dest.Email != "modified_test@test.com" {
		t.Errorf("AfterMap modification failed: got %s", dest.Email)
	}
}

// Test CustomMap
func TestCustomMap(t *testing.T) {
	mapper := New()
	CreateMap[SourceBasic, DestBasic](mapper).
		CustomMap(func(src SourceBasic, dest *DestBasic) error {
			dest.Name = "Custom_" + src.Name
			dest.Age = src.Age * 2
			dest.Email = src.Email
			return nil
		})

	src := SourceBasic{Name: "Test", Age: 20, Email: "test@test.com"}

	dest, err := Map[DestBasic](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Name != "Custom_Test" {
		t.Errorf("Name mismatch: got %s, want Custom_Test", dest.Name)
	}
	if dest.Age != 40 {
		t.Errorf("Age mismatch: got %d, want 40", dest.Age)
	}
}

// Test map field
type SourceWithMap struct {
	Name   string
	Labels map[string]string
}

type DestWithMap struct {
	Name   string
	Labels map[string]string
}

func TestMapField(t *testing.T) {
	mapper := New()
	CreateMap[SourceWithMap, DestWithMap](mapper)

	src := SourceWithMap{
		Name: "Test",
		Labels: map[string]string{
			"env":  "prod",
			"tier": "backend",
		},
	}

	dest, err := Map[DestWithMap](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Name != "Test" {
		t.Errorf("Name mismatch: got %s", dest.Name)
	}
	if len(dest.Labels) != 2 {
		t.Fatalf("Labels length mismatch: got %d", len(dest.Labels))
	}
	if dest.Labels["env"] != "prod" {
		t.Errorf("Labels[env] mismatch: got %s", dest.Labels["env"])
	}
}

// Test ForMember with field selector (pointer return)
func TestForMemberWithPointerSelector(t *testing.T) {
	mapper := New()

	// Using ForMember with a selector that returns a pointer to the field
	CreateMap[SourceBasic, DestBasic](mapper).
		ForMember(func(d *DestBasic) any { return &d.Email }, MapFromFunc(func(src any, dest any) (any, error) {
			s := src.(SourceBasic)
			return "forMember_" + s.Email, nil
		}))

	src := SourceBasic{Name: "Test", Age: 20, Email: "test@test.com"}

	dest, err := Map[DestBasic](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Email != "forMember_test@test.com" {
		t.Errorf("Email mismatch: got %s, want forMember_test@test.com", dest.Email)
	}
}

// Test ForMember with Ignore option
func TestForMemberIgnore(t *testing.T) {
	mapper := New()

	CreateMap[SourceBasic, DestBasic](mapper).
		ForMember(func(d *DestBasic) any { return &d.Age }, Ignore())

	src := SourceBasic{Name: "Test", Age: 25, Email: "test@test.com"}

	dest, err := Map[DestBasic](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Age != 0 {
		t.Errorf("Age should be 0 (ignored): got %d", dest.Age)
	}
	if dest.Name != "Test" {
		t.Errorf("Name mismatch: got %s", dest.Name)
	}
}

// Test ForMember with MapFrom option
func TestForMemberMapFrom(t *testing.T) {
	type SourceAlt struct {
		FullName string
		Years    int
		Contact  string
	}

	type DestAlt struct {
		Name  string
		Age   int
		Email string
	}

	mapper := New()

	CreateMap[SourceAlt, DestAlt](mapper).
		ForMember(func(d *DestAlt) any { return &d.Name }, MapFrom("FullName")).
		ForMember(func(d *DestAlt) any { return &d.Age }, MapFrom("Years")).
		ForMember(func(d *DestAlt) any { return &d.Email }, MapFrom("Contact"))

	src := SourceAlt{FullName: "John Doe", Years: 30, Contact: "john@example.com"}

	dest, err := Map[DestAlt](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Name != "John Doe" {
		t.Errorf("Name mismatch: got %s, want John Doe", dest.Name)
	}
	if dest.Age != 30 {
		t.Errorf("Age mismatch: got %d, want 30", dest.Age)
	}
	if dest.Email != "john@example.com" {
		t.Errorf("Email mismatch: got %s, want john@example.com", dest.Email)
	}
}

// =============================================================================
// Optimization Mode Tests
// =============================================================================

// Test types for optimization tests
type OptSource struct {
	ID     int
	Name   string
	Age    int
	Active bool
	Score  float64
}

type OptDest struct {
	ID     int
	Name   string
	Age    int
	Active bool
	Score  float64
}

// TestPooledMapping tests mapping with pooling enabled
func TestPooledMapping(t *testing.T) {
	mapper := NewWithConfig(WithPooling())
	CreateMap[OptSource, OptDest](mapper)

	src := OptSource{ID: 1, Name: "Test", Age: 25, Active: true, Score: 88.5}
	dest, err := Map[OptDest](mapper, src)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.ID != 1 {
		t.Errorf("ID mismatch: got %d, want 1", dest.ID)
	}
	if dest.Name != "Test" {
		t.Errorf("Name mismatch: got %s, want Test", dest.Name)
	}
	if dest.Age != 25 {
		t.Errorf("Age mismatch: got %d, want 25", dest.Age)
	}
	if dest.Active != true {
		t.Errorf("Active mismatch: got %v, want true", dest.Active)
	}
	if dest.Score != 88.5 {
		t.Errorf("Score mismatch: got %f, want 88.5", dest.Score)
	}
}

// TestUnsafeMapping tests mapping with unsafe optimizations
func TestUnsafeMapping(t *testing.T) {
	mapper := NewWithConfig(WithUnsafeOptimizations())
	CreateMap[OptSource, OptDest](mapper)

	src := OptSource{ID: 42, Name: "Unsafe", Age: 30, Active: false, Score: 99.9}
	dest, err := Map[OptDest](mapper, src)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.ID != 42 {
		t.Errorf("ID mismatch: got %d, want 42", dest.ID)
	}
	if dest.Name != "Unsafe" {
		t.Errorf("Name mismatch: got %s, want Unsafe", dest.Name)
	}
	if dest.Age != 30 {
		t.Errorf("Age mismatch: got %d, want 30", dest.Age)
	}
	if dest.Active != false {
		t.Errorf("Active mismatch: got %v, want false", dest.Active)
	}
	if dest.Score != 99.9 {
		t.Errorf("Score mismatch: got %f, want 99.9", dest.Score)
	}
}

// TestSpecializedMapping tests mapping with specialized mappers
func TestSpecializedMapping(t *testing.T) {
	mapper := NewWithConfig(WithSpecializedMappers())
	CreateMap[OptSource, OptDest](mapper)

	src := OptSource{ID: 100, Name: "Specialized", Age: 35, Active: true, Score: 77.7}
	dest, err := Map[OptDest](mapper, src)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.ID != 100 {
		t.Errorf("ID mismatch: got %d, want 100", dest.ID)
	}
	if dest.Name != "Specialized" {
		t.Errorf("Name mismatch: got %s, want Specialized", dest.Name)
	}
	if dest.Age != 35 {
		t.Errorf("Age mismatch: got %d, want 35", dest.Age)
	}
	if dest.Active != true {
		t.Errorf("Active mismatch: got %v, want true", dest.Active)
	}
	if dest.Score != 77.7 {
		t.Errorf("Score mismatch: got %f, want 77.7", dest.Score)
	}
}

// TestOptimizedNestedMapping tests nested struct mapping with optimizations
func TestOptimizedNestedMapping(t *testing.T) {
	mapper := NewWithConfig(WithSpecializedMappers())
	CreateMap[SourceNested, DestNested](mapper)
	CreateMap[Address, AddressDTO](mapper)

	src := SourceNested{
		Name: "John",
		Address: Address{
			Street: "123 Main St",
			City:   "Boston",
			Zip:    "02101",
		},
	}

	dest, err := Map[DestNested](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Name != src.Name {
		t.Errorf("Name mismatch: got %s, want %s", dest.Name, src.Name)
	}
	if dest.Address.City != "Boston" {
		t.Errorf("City mismatch: got %s, want Boston", dest.Address.City)
	}
}

// TestOptimizedWithHooks tests that hooks work correctly with optimization
func TestOptimizedWithHooks(t *testing.T) {
	mapper := NewWithConfig(WithSpecializedMappers())
	hookCalled := false

	CreateMap[OptSource, OptDest](mapper).
		AfterMap(func(src *OptSource, dest *OptDest) error {
			hookCalled = true
			dest.Name = "Modified"
			return nil
		})

	src := OptSource{ID: 1, Name: "Original", Age: 25, Active: true, Score: 50.0}
	dest, err := Map[OptDest](mapper, src)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hookCalled {
		t.Error("AfterMap hook should have been called")
	}
	if dest.Name != "Modified" {
		t.Errorf("Name should be modified by hook: got %s", dest.Name)
	}
}

// TestOptimizedSliceMapping tests slice mapping with optimizations
func TestOptimizedSliceMapping(t *testing.T) {
	mapper := NewWithConfig(WithPooling())
	CreateMap[SourceItem, DestItem](mapper)

	src := []SourceItem{
		{ID: 1, Name: "Item 1"},
		{ID: 2, Name: "Item 2"},
		{ID: 3, Name: "Item 3"},
	}

	dest, err := MapSlice[SourceItem, DestItem](mapper, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dest) != 3 {
		t.Fatalf("Length mismatch: got %d, want 3", len(dest))
	}
	if dest[1].Name != "Item 2" {
		t.Errorf("Name mismatch: got %s, want Item 2", dest[1].Name)
	}
}

// TestOptimizationLevelConfiguration tests configuration options
func TestOptimizationLevelConfiguration(t *testing.T) {
	t.Run("WithOptimizationLevel Pooled", func(t *testing.T) {
		mapper := NewWithConfig(WithOptimizationLevel(OptimizationPooled))
		if mapper.config.optLevel != OptimizationPooled {
			t.Errorf("optLevel mismatch: got %v, want %v", mapper.config.optLevel, OptimizationPooled)
		}
	})

	t.Run("WithOptimizationLevel Unsafe", func(t *testing.T) {
		mapper := NewWithConfig(WithOptimizationLevel(OptimizationUnsafe))
		if mapper.config.optLevel != OptimizationUnsafe {
			t.Errorf("optLevel mismatch: got %v, want %v", mapper.config.optLevel, OptimizationUnsafe)
		}
		if !mapper.config.useUnsafe {
			t.Error("useUnsafe should be true")
		}
	})

	t.Run("WithOptimizationLevel Specialized", func(t *testing.T) {
		mapper := NewWithConfig(WithOptimizationLevel(OptimizationSpecialized))
		if mapper.config.optLevel != OptimizationSpecialized {
			t.Errorf("optLevel mismatch: got %v, want %v", mapper.config.optLevel, OptimizationSpecialized)
		}
		if !mapper.config.useUnsafe {
			t.Error("useUnsafe should be true")
		}
	})
}
