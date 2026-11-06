// Package inventory keeps a small ordered list of items.
package inventory

import "errors"

// Item is one stocked item.
type Item struct {
	Name  string
	Count int
}

// Store holds items in the order they were added.
type Store struct {
	items []Item
}

// ErrNotFound is returned for an index outside the store.
var ErrNotFound = errors.New("inventory: no such item")

// Add appends an item.
func (s *Store) Add(it Item) {
	s.items = append(s.items, it)
}

// Get returns the item at index i.
func (s *Store) Get(i int) (Item, error) {
	if i < 0 || i > len(s.items) {
		return Item{}, ErrNotFound
	}
	return s.items[i], nil
}

// Total returns the sum of all counts.
func (s *Store) Total() int {
	n := 0
	for _, it := range s.items {
		n += it.Count
	}
	return n
}

// Names returns the item names in the order they were added.
func (s *Store) Names() []string {
	names := make([]string, 0, len(s.items))
	for _, it := range s.items {
		names = append(names, it.Name)
	}
	return names
}
