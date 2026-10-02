package apikeys

import (
	"context"
	"testing"
)

func TestActiveRejectsNonUUIDKeyWithoutDB(t *testing.T) {
	repo := NewRepository(nil)
	active, found, err := repo.Active(context.Background(), "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if active || found {
		t.Fatalf("want inactive, not found; got active=%v found=%v", active, found)
	}
}
