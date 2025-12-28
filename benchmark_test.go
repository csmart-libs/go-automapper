package automapper

import (
	"testing"
)

// Benchmark types
type BenchSource struct {
	ID        int
	Name      string
	Email     string
	Age       int
	Active    bool
	Score     float64
	Tags      []string
	CreatedAt string
}

type BenchDest struct {
	ID        int
	Name      string
	Email     string
	Age       int
	Active    bool
	Score     float64
	Tags      []string
	CreatedAt string
}

// Manual mapping function for comparison
func manualMap(src BenchSource) BenchDest {
	return BenchDest{
		ID:        src.ID,
		Name:      src.Name,
		Email:     src.Email,
		Age:       src.Age,
		Active:    src.Active,
		Score:     src.Score,
		Tags:      src.Tags,
		CreatedAt: src.CreatedAt,
	}
}

var benchSource = BenchSource{
	ID:        1,
	Name:      "John Doe",
	Email:     "john@example.com",
	Age:       30,
	Active:    true,
	Score:     95.5,
	Tags:      []string{"go", "developer", "senior"},
	CreatedAt: "2024-01-01",
}

// BenchmarkManualMapping benchmarks manual struct mapping
func BenchmarkManualMapping(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = manualMap(benchSource)
	}
}

// BenchmarkAutoMapper benchmarks automapper mapping
func BenchmarkAutoMapper(b *testing.B) {
	mapper := New()
	CreateMap[BenchSource, BenchDest](mapper)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Map[BenchDest](mapper, benchSource)
	}
}

// BenchmarkAutoMapperPreConfigured benchmarks with pre-warmed cache
func BenchmarkAutoMapperPreConfigured(b *testing.B) {
	mapper := New()
	CreateMap[BenchSource, BenchDest](mapper)
	// Warm up the cache
	_, _ = Map[BenchDest](mapper, benchSource)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Map[BenchDest](mapper, benchSource)
	}
}

// Nested benchmark types
type BenchNestedSource struct {
	ID      int
	Name    string
	Address BenchAddressSource
	Items   []BenchItemSource
}

type BenchAddressSource struct {
	Street string
	City   string
	Zip    string
}

type BenchItemSource struct {
	ID    int
	Name  string
	Price float64
}

type BenchNestedDest struct {
	ID      int
	Name    string
	Address BenchAddressDest
	Items   []BenchItemDest
}

type BenchAddressDest struct {
	Street string
	City   string
	Zip    string
}

type BenchItemDest struct {
	ID    int
	Name  string
	Price float64
}

var benchNestedSource = BenchNestedSource{
	ID:   1,
	Name: "Complex Order",
	Address: BenchAddressSource{
		Street: "123 Main St",
		City:   "Boston",
		Zip:    "02101",
	},
	Items: []BenchItemSource{
		{ID: 1, Name: "Item 1", Price: 10.99},
		{ID: 2, Name: "Item 2", Price: 20.99},
		{ID: 3, Name: "Item 3", Price: 30.99},
	},
}

// Manual nested mapping
func manualNestedMap(src BenchNestedSource) BenchNestedDest {
	items := make([]BenchItemDest, len(src.Items))
	for i, item := range src.Items {
		items[i] = BenchItemDest{
			ID:    item.ID,
			Name:  item.Name,
			Price: item.Price,
		}
	}
	return BenchNestedDest{
		ID:   src.ID,
		Name: src.Name,
		Address: BenchAddressDest{
			Street: src.Address.Street,
			City:   src.Address.City,
			Zip:    src.Address.Zip,
		},
		Items: items,
	}
}

// BenchmarkManualNestedMapping benchmarks manual nested mapping
func BenchmarkManualNestedMapping(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = manualNestedMap(benchNestedSource)
	}
}

// BenchmarkAutoMapperNested benchmarks automapper with nested structs
func BenchmarkAutoMapperNested(b *testing.B) {
	mapper := New()
	CreateMap[BenchNestedSource, BenchNestedDest](mapper)
	CreateMap[BenchAddressSource, BenchAddressDest](mapper)
	CreateMap[BenchItemSource, BenchItemDest](mapper)
	// Warm up
	_, _ = Map[BenchNestedDest](mapper, benchNestedSource)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Map[BenchNestedDest](mapper, benchNestedSource)
	}
}

// BenchmarkSliceMapping benchmarks slice mapping
func BenchmarkSliceMapping(b *testing.B) {
	mapper := New()
	CreateMap[BenchItemSource, BenchItemDest](mapper)

	items := make([]BenchItemSource, 100)
	for i := 0; i < 100; i++ {
		items[i] = BenchItemSource{ID: i, Name: "Item", Price: float64(i)}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = MapSlice[BenchItemSource, BenchItemDest](mapper, items)
	}
}

// BenchmarkManualSliceMapping benchmarks manual slice mapping
func BenchmarkManualSliceMapping(b *testing.B) {
	items := make([]BenchItemSource, 100)
	for i := 0; i < 100; i++ {
		items[i] = BenchItemSource{ID: i, Name: "Item", Price: float64(i)}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := make([]BenchItemDest, len(items))
		for j, item := range items {
			result[j] = BenchItemDest{ID: item.ID, Name: item.Name, Price: item.Price}
		}
		_ = result
	}
}

// =============================================================================
// Optimization Level Benchmarks
// =============================================================================

// Primitive-only types for specialized mapper benchmarks
type BenchPrimitiveSource struct {
	ID     int
	Name   string
	Age    int
	Active bool
	Score  float64
}

type BenchPrimitiveDest struct {
	ID     int
	Name   string
	Age    int
	Active bool
	Score  float64
}

var benchPrimitiveSource = BenchPrimitiveSource{
	ID:     1,
	Name:   "John Doe",
	Age:    30,
	Active: true,
	Score:  95.5,
}

// BenchmarkAutoMapperPooled benchmarks with pooling enabled
func BenchmarkAutoMapperPooled(b *testing.B) {
	mapper := NewWithConfig(WithPooling())
	CreateMap[BenchSource, BenchDest](mapper)
	// Warm up
	_, _ = Map[BenchDest](mapper, benchSource)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Map[BenchDest](mapper, benchSource)
	}
}

// BenchmarkAutoMapperUnsafe benchmarks with unsafe optimizations enabled
func BenchmarkAutoMapperUnsafe(b *testing.B) {
	mapper := NewWithConfig(WithUnsafeOptimizations())
	CreateMap[BenchPrimitiveSource, BenchPrimitiveDest](mapper)
	// Warm up
	_, _ = Map[BenchPrimitiveDest](mapper, benchPrimitiveSource)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Map[BenchPrimitiveDest](mapper, benchPrimitiveSource)
	}
}

// BenchmarkAutoMapperSpecialized benchmarks with specialized mappers
func BenchmarkAutoMapperSpecialized(b *testing.B) {
	mapper := NewWithConfig(WithSpecializedMappers())
	CreateMap[BenchPrimitiveSource, BenchPrimitiveDest](mapper)
	// Warm up
	_, _ = Map[BenchPrimitiveDest](mapper, benchPrimitiveSource)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Map[BenchPrimitiveDest](mapper, benchPrimitiveSource)
	}
}

// BenchmarkPrimitiveManual benchmarks manual primitive mapping for comparison
func BenchmarkPrimitiveManual(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = BenchPrimitiveDest{
			ID:     benchPrimitiveSource.ID,
			Name:   benchPrimitiveSource.Name,
			Age:    benchPrimitiveSource.Age,
			Active: benchPrimitiveSource.Active,
			Score:  benchPrimitiveSource.Score,
		}
	}
}

// BenchmarkPrimitiveStandard benchmarks standard mapping for primitives
func BenchmarkPrimitiveStandard(b *testing.B) {
	mapper := New()
	CreateMap[BenchPrimitiveSource, BenchPrimitiveDest](mapper)
	// Warm up
	_, _ = Map[BenchPrimitiveDest](mapper, benchPrimitiveSource)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Map[BenchPrimitiveDest](mapper, benchPrimitiveSource)
	}
}
