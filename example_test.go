// Package automapper demonstrates the usage of the go-automapper library.
//
// This file contains example code showing various features of the library.
package automapper_test

import (
	"fmt"

	"github.com/csmart-libs/go-automapper"
)

// Entity types (domain layer)
type User struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
	Age       int
	Address   Address
	Tags      []string
}

type Address struct {
	Street  string
	City    string
	State   string
	ZipCode string
}

// DTO types (presentation layer)
type UserDTO struct {
	ID          int
	FirstName   string
	LastName    string
	Email       string
	Age         int
	AddressCity string // Flattened from Address.City
	Tags        []string
}

type UserDetailDTO struct {
	ID      int
	Name    string // Combined FirstName + LastName
	Email   string
	Address AddressDTO
}

type AddressDTO struct {
	Street  string
	City    string
	State   string
	ZipCode string
}

// Example demonstrates basic usage of automapper.
func Example() {
	// Create a new mapper
	mapper := automapper.New()

	// Configure mappings
	automapper.CreateMap[User, UserDTO](mapper)

	// Create source object
	user := User{
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Age:       30,
		Address: Address{
			Street:  "123 Main St",
			City:    "Boston",
			State:   "MA",
			ZipCode: "02101",
		},
		Tags: []string{"developer", "golang"},
	}

	// Perform mapping
	dto, err := automapper.Map[UserDTO](mapper, user)
	if err != nil {
		panic(err)
	}

	fmt.Printf("User: %s %s, Email: %s\n", dto.FirstName, dto.LastName, dto.Email)
	fmt.Printf("City: %s\n", dto.AddressCity)
	fmt.Printf("Tags: %v\n", dto.Tags)

	// Output:
	// User: John Doe, Email: john@example.com
	// City: Boston
	// Tags: [developer golang]
}

// ExampleNestedMapping demonstrates nested struct mapping.
func Example_nestedMapping() {
	mapper := automapper.New()

	// Configure both parent and nested type mappings
	automapper.CreateMap[User, UserDetailDTO](mapper)
	automapper.CreateMap[Address, AddressDTO](mapper)

	user := User{
		ID:        1,
		FirstName: "Jane",
		LastName:  "Smith",
		Email:     "jane@example.com",
		Address: Address{
			City:  "New York",
			State: "NY",
		},
	}

	dto, _ := automapper.Map[UserDetailDTO](mapper, user)

	fmt.Printf("City: %s, State: %s\n", dto.Address.City, dto.Address.State)

	// Output:
	// City: New York, State: NY
}

// ExampleCustomResolver demonstrates custom value resolver.
func Example_customResolver() {
	mapper := automapper.New()

	automapper.CreateMap[User, UserDetailDTO](mapper).
		ForMemberByName("Name", automapper.MapFromFunc(func(src any, dest any) (any, error) {
			u := src.(User)
			return u.FirstName + " " + u.LastName, nil
		}))

	user := User{FirstName: "John", LastName: "Doe"}

	dto, _ := automapper.Map[UserDetailDTO](mapper, user)

	fmt.Printf("Name: %s\n", dto.Name)

	// Output:
	// Name: John Doe
}

// ExampleSliceMapping demonstrates slice mapping.
func Example_sliceMapping() {
	mapper := automapper.New()
	automapper.CreateMap[User, UserDTO](mapper)

	users := []User{
		{ID: 1, FirstName: "John", Email: "john@example.com"},
		{ID: 2, FirstName: "Jane", Email: "jane@example.com"},
	}

	dtos, _ := automapper.MapSlice[User, UserDTO](mapper, users)

	for _, dto := range dtos {
		fmt.Printf("User %d: %s\n", dto.ID, dto.FirstName)
	}

	// Output:
	// User 1: John
	// User 2: Jane
}

