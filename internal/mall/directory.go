package mall

import (
	"errors"
	"strings"
)

var ErrStoreNotFound = errors.New("store not found")

// Store represents a store in the mall directory.
type Store struct {
	Name        string
	Floor       string
	Zone        string
	Coordinates string
}

// directory holds the in-memory store map.
var directory = initDirectory()

func initDirectory() map[string]*Store {
	stores := map[string]*Store{
		"nike": {
			Name:        "Nike",
			Floor:       "1",
			Zone:        "Sportswear",
			Coordinates: "10.5,5.2,0",
		},
		"adidas": {
			Name:        "Adidas",
			Floor:       "1",
			Zone:        "Sportswear",
			Coordinates: "12.0,5.2,0",
		},
		"electronics": {
			Name:        "Electronics Zone",
			Floor:       "2",
			Zone:        "Electronics",
			Coordinates: "8.0,15.0,0",
		},
		"food court": {
			Name:        "Food Court",
			Floor:       "1",
			Zone:        "Dining",
			Coordinates: "5.0,20.0,0",
		},
		"food": {
			Name:        "Food Court",
			Floor:       "1",
			Zone:        "Dining",
			Coordinates: "5.0,20.0,0",
		},
		"restaurant": {
			Name:        "Food Court",
			Floor:       "1",
			Zone:        "Dining",
			Coordinates: "5.0,20.0,0",
		},
		"restaurants": {
			Name:        "Food Court",
			Floor:       "1",
			Zone:        "Dining",
			Coordinates: "5.0,20.0,0",
		},
	}
	return stores
}

// FindStore looks up a store by name. Search is case-insensitive.
// Supports partial matching: "nike" -> Nike, "food" -> Food Court.
func FindStore(name string) (*Store, error) {
	if name == "" {
		return nil, ErrStoreNotFound
	}
	key := strings.ToLower(strings.TrimSpace(name))
	if store, ok := directory[key]; ok {
		return store, nil
	}
	// Try partial match: check if any store name contains the query
	for k, store := range directory {
		if strings.Contains(k, key) || strings.Contains(key, k) {
			return store, nil
		}
		if strings.Contains(strings.ToLower(store.Name), key) {
			return store, nil
		}
	}
	return nil, ErrStoreNotFound
}
