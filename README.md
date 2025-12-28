# Go AutoMapper

A high-performance object-to-object mapping library for Go - AutoMapper.

[![Go Reference](https://pkg.go.dev/badge/github.com/csmart-libs/go-automapper.svg)](https://pkg.go.dev/github.com/csmart-libs/go-automapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/csmart-libs/go-automapper)](https://goreportcard.com/report/github.com/csmart-libs/go-automapper)

## Features

- ✅ **Automatic mapping** between structs based on field name matching
- ✅ **Nested struct mapping** with automatic recursion
- ✅ **Flattening support** (e.g., `Address.City` → `AddressCity`)
- ✅ **Slice and array mapping**
- ✅ **Map field mapping**
- ✅ **Custom value resolvers** for field-level transformations
- ✅ **Type converters** for type-to-type transformations
- ✅ **Conditional mapping** based on source values
- ✅ **Before/After map hooks**
- ✅ **Ignore fields** configuration
- ✅ **High performance** through reflection caching
- ✅ **Type-safe API** using Go generics

## Installation

```bash
go get github.com/csmart-libs/go-automapper
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/csmart-libs/go-automapper"
)

// Source type
type User struct {
    ID        int
    FirstName string
    LastName  string
    Email     string
}

// Destination type
type UserDTO struct {
    ID        int
    FirstName string
    LastName  string
    Email     string
}

func main() {
    // Create mapper
    mapper := automapper.New()

    // Configure mapping
    automapper.CreateMap[User, UserDTO](mapper)

    // Map object
    user := User{ID: 1, FirstName: "John", LastName: "Doe", Email: "john@example.com"}
    dto, err := automapper.Map[UserDTO](mapper, user)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Mapped: %+v\n", dto)
}
```

## Usage Examples

### Basic Mapping

```go
mapper := automapper.New()
automapper.CreateMap[Source, Dest](mapper)

src := Source{Name: "John", Age: 30}
dest, err := automapper.Map[Dest](mapper, src)
```

### Map to Existing Object

```go
var dest Dest
err := automapper.MapTo(mapper, src, &dest)
```

### Slice Mapping

```go
users := []User{{ID: 1, Name: "John"}, {ID: 2, Name: "Jane"}}
dtos, err := automapper.MapSlice[User, UserDTO](mapper, users)
```

### Nested Struct Mapping

```go
type Order struct {
    ID      int
    Customer Customer
}

type Customer struct {
    Name  string
    Email string
}

type OrderDTO struct {
    ID       int
    Customer CustomerDTO
}

type CustomerDTO struct {
    Name  string
    Email string
}

mapper := automapper.New()
automapper.CreateMap[Order, OrderDTO](mapper)
automapper.CreateMap[Customer, CustomerDTO](mapper)

order := Order{ID: 1, Customer: Customer{Name: "John", Email: "john@example.com"}}
dto, _ := automapper.Map[OrderDTO](mapper, order)
```

### Flattening

Automatically maps nested properties to flattened destination fields:

```go
type Source struct {
    Customer Customer
}

type Customer struct {
    Name string
}

type Dest struct {
    CustomerName string  // Automatically mapped from Customer.Name
}

mapper := automapper.New()
automapper.CreateMap[Source, Dest](mapper)
```


### Custom Value Resolver

```go
automapper.CreateMap[User, UserDTO](mapper).
    ForMemberByName("FullName", automapper.MapFromFunc(func(src any, dest any) (any, error) {
        u := src.(User)
        return u.FirstName + " " + u.LastName, nil
    }))
```

### Map From Different Field

```go
automapper.CreateMap[Source, Dest](mapper).
    ForMemberByName("DestField", automapper.MapFrom("SrcField"))
```

### Ignore Field

```go
automapper.CreateMap[Source, Dest](mapper).
    ForMemberByName("Password", automapper.Ignore())
```

### Conditional Mapping

```go
automapper.CreateMap[Source, Dest](mapper).
    ForMemberByName("Age", automapper.Condition(func(src any) bool {
        return src.(Source).Age > 0
    }))
```

### Before/After Map Hooks

```go
automapper.CreateMap[User, UserDTO](mapper).
    BeforeMap(func(src *User, dest *UserDTO) error {
        // Called before mapping
        return nil
    }).
    AfterMap(func(src *User, dest *UserDTO) error {
        // Called after mapping
        dest.Email = strings.ToLower(dest.Email)
        return nil
    })
```

### Custom Mapper

```go
automapper.CreateMap[User, UserDTO](mapper).
    CustomMap(func(src User, dest *UserDTO) error {
        dest.ID = src.ID
        dest.FullName = src.FirstName + " " + src.LastName
        return nil
    })
```

### Type Converter

```go
automapper.ConvertUsing[time.Time, string](mapper, func(t time.Time) (string, error) {
    return t.Format("2006-01-02"), nil
})
```

## Configuration Options

```go
// Allow nil slices/maps in output (default: empty slice/map)
mapper := automapper.NewWithConfig(automapper.WithAllowNullCollections())
```

## Performance

The library uses reflection caching to minimize overhead. Benchmark results on Intel i7-12700K:

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Manual Mapping | 1.3 | 0 | 0 |
| AutoMapper Basic | 417 | 224 | 2 |
| AutoMapper Nested | 1181 | 312 | 4 |
| Manual Slice (100 items) | 456 | 3456 | 1 |
| AutoMapper Slice (100 items) | 19898 | 9856 | 201 |

While there is overhead compared to manual mapping due to reflection, the library provides significant productivity benefits for complex mapping scenarios.

## API Reference

### Core Functions

- `New()` - Creates a new mapper with default configuration
- `NewWithConfig(opts ...ConfigOption)` - Creates a mapper with custom options
- `CreateMap[TSrc, TDest](m *Mapper)` - Configures a type mapping
- `Map[TDest](m *Mapper, src any)` - Maps source to new destination
- `MapTo[TDest](m *Mapper, src any, dest *TDest)` - Maps source to existing destination
- `MapSlice[TSrc, TDest](m *Mapper, src []TSrc)` - Maps a slice

### Member Options

- `MapFrom(srcFieldName string)` - Map from a different source field
- `MapFromFunc(resolver ValueResolver)` - Use custom resolver
- `Ignore()` - Skip this field during mapping
- `Condition(cond ConditionFunc)` - Conditional mapping
- `UseConverter(converter TypeConverter)` - Use type converter

### Builder Methods

- `ForMemberByName(name string, opts ...MemberOption)` - Configure specific field
- `BeforeMap(fn)` - Add pre-mapping hook
- `AfterMap(fn)` - Add post-mapping hook
- `CustomMap(fn)` - Use custom mapping function
- `ReverseMap()` - Create reverse mapping

## License

MIT License - see LICENSE file for details.

