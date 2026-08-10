package batch

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuilderRejectsAnInvalidBatch(t *testing.T) {
	if _, err := NewBuilder(0).Build(); err == nil {
		t.Error("want an error for a zero requestedCount")
	}
	if _, err := NewBuilder(5).SetGeneratedCount(6).Build(); err == nil {
		t.Error("want an error when generatedCount exceeds requestedCount")
	}
}

func TestBuilderBuildsAValidBatch(t *testing.T) {
	id := uuid.New()
	createdAt := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

	m, err := NewBuilder(10).
		SetId(id).
		SetDescription("august promo").
		SetGeneratedCount(10).
		SetCreatedAt(createdAt).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m.Id() != id {
		t.Errorf("Id() = %v, want %v", m.Id(), id)
	}
	if m.Description() != "august promo" {
		t.Errorf("Description() = %q, want %q", m.Description(), "august promo")
	}
	if m.RequestedCount() != 10 {
		t.Errorf("RequestedCount() = %v, want 10", m.RequestedCount())
	}
	if m.GeneratedCount() != 10 {
		t.Errorf("GeneratedCount() = %v, want 10", m.GeneratedCount())
	}
	if !m.CreatedAt().Equal(createdAt) {
		t.Errorf("CreatedAt() = %v, want %v", m.CreatedAt(), createdAt)
	}
}
