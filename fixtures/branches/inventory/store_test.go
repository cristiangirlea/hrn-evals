package inventory

import (
	"errors"
	"slices"
	"testing"
)

func TestGet(t *testing.T) {
	noteRun(t)
	var s Store
	s.Add(Item{"bolt", 4})
	if it, err := s.Get(0); err != nil || it.Name != "bolt" {
		t.Errorf("Get(0) = %v, %v", it, err)
	}
	if _, err := s.Get(-1); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(-1) error = %v, want ErrNotFound", err)
	}
}

func TestTotal(t *testing.T) {
	noteRun(t)
	var s Store
	s.Add(Item{"bolt", 4})
	s.Add(Item{"nut", 6})
	if got := s.Total(); got != 10 {
		t.Errorf("Total() = %d, want 10", got)
	}
}

func TestNames(t *testing.T) {
	noteRun(t)
	var s Store
	if got := s.Names(); len(got) != 0 {
		t.Errorf("Names() of an empty store = %v, want none", got)
	}
	s.Add(Item{"bolt", 4})
	s.Add(Item{"nut", 6})
	if got := s.Names(); !slices.Equal(got, []string{"bolt", "nut"}) {
		t.Errorf("Names() = %v, want [bolt nut]", got)
	}
}
