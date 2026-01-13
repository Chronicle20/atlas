package messenger

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

// setupTestContext creates a context with a unique tenant for test isolation
func setupTestContext(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "TEST", 1, 1)
	if err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), ten)
}

func TestMemberFilterMatch(t *testing.T) {
	// Create a model with a specific member
	m, err := NewBuilder().
		SetTenantId(uuid.New()).
		SetId(1).
		AddMember(100, 0).
		AddMember(200, 1).
		Build()
	if err != nil {
		t.Fatalf("failed to build model: %v", err)
	}

	// Test filter matches member 100
	filter := MemberFilter(100)
	if !filter(m) {
		t.Fatal("expected filter to match member 100")
	}

	// Test filter matches member 200
	filter = MemberFilter(200)
	if !filter(m) {
		t.Fatal("expected filter to match member 200")
	}
}

func TestMemberFilterNoMatch(t *testing.T) {
	// Create a model with specific members
	m, err := NewBuilder().
		SetTenantId(uuid.New()).
		SetId(1).
		AddMember(100, 0).
		AddMember(200, 1).
		Build()
	if err != nil {
		t.Fatalf("failed to build model: %v", err)
	}

	// Test filter does not match non-existent member
	filter := MemberFilter(999)
	if filter(m) {
		t.Fatal("expected filter to not match non-existent member 999")
	}
}

func TestMemberFilterEmptyMembers(t *testing.T) {
	// Create a model with no members
	m, err := NewBuilder().
		SetTenantId(uuid.New()).
		SetId(1).
		Build()
	if err != nil {
		t.Fatalf("failed to build model: %v", err)
	}

	// Test filter does not match when no members
	filter := MemberFilter(100)
	if filter(m) {
		t.Fatal("expected filter to not match when model has no members")
	}
}

func TestGetByIdSuccess(t *testing.T) {
	ctx := setupTestContext(t)

	// Create a messenger via registry
	ten := tenant.MustFromContext(ctx)
	r := GetRegistry()
	created := r.Create(ten, 100)

	// Test GetById returns the messenger
	result, err := GetById(ctx)(created.Id())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result.Id() != created.Id() {
		t.Fatalf("expected id=%d, got %d", created.Id(), result.Id())
	}
	if len(result.Members()) != 1 {
		t.Fatalf("expected 1 member, got %d", len(result.Members()))
	}
	if result.Members()[0].Id() != 100 {
		t.Fatalf("expected member id=100, got %d", result.Members()[0].Id())
	}
}

func TestGetByIdNotFound(t *testing.T) {
	ctx := setupTestContext(t)

	// Test GetById with non-existent ID
	_, err := GetById(ctx)(999999999)
	if err == nil {
		t.Fatal("expected error for non-existent messenger, got nil")
	}
}

func TestGetSliceEmpty(t *testing.T) {
	ctx := setupTestContext(t)

	// Test GetSlice with no messengers (new tenant)
	result, err := GetSlice(ctx)()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(result))
	}
}

func TestGetSliceWithData(t *testing.T) {
	ctx := setupTestContext(t)

	// Create messengers via registry
	ten := tenant.MustFromContext(ctx)
	r := GetRegistry()
	m1 := r.Create(ten, 100)
	m2 := r.Create(ten, 200)

	// Test GetSlice returns all messengers
	result, err := GetSlice(ctx)()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 messengers, got %d", len(result))
	}

	// Verify both messengers are returned
	ids := make(map[uint32]bool)
	for _, m := range result {
		ids[m.Id()] = true
	}
	if !ids[m1.Id()] {
		t.Fatalf("expected messenger %d in results", m1.Id())
	}
	if !ids[m2.Id()] {
		t.Fatalf("expected messenger %d in results", m2.Id())
	}
}

func TestGetSliceWithFilter(t *testing.T) {
	ctx := setupTestContext(t)

	// Create messengers via registry
	ten := tenant.MustFromContext(ctx)
	r := GetRegistry()
	m1 := r.Create(ten, 100)
	_ = r.Create(ten, 200)

	// Test GetSlice with MemberFilter returns only matching
	result, err := GetSlice(ctx)(MemberFilter(100))
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 messenger matching filter, got %d", len(result))
	}
	if result[0].Id() != m1.Id() {
		t.Fatalf("expected messenger %d, got %d", m1.Id(), result[0].Id())
	}
}

func TestGetSliceWithFilterNoMatch(t *testing.T) {
	ctx := setupTestContext(t)

	// Create messengers via registry
	ten := tenant.MustFromContext(ctx)
	r := GetRegistry()
	_ = r.Create(ten, 100)
	_ = r.Create(ten, 200)

	// Test GetSlice with filter that matches nothing
	result, err := GetSlice(ctx)(MemberFilter(999))
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 messengers matching filter, got %d", len(result))
	}
}

func TestProcessorImplGetById(t *testing.T) {
	ctx := setupTestContext(t)

	// Create a messenger via registry
	ten := tenant.MustFromContext(ctx)
	r := GetRegistry()
	created := r.Create(ten, 100)

	// Test ProcessorImpl.GetById
	proc := NewProcessor(nil, ctx)
	result, err := proc.GetById(created.Id())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result.Id() != created.Id() {
		t.Fatalf("expected id=%d, got %d", created.Id(), result.Id())
	}
}

func TestProcessorImplGetSlice(t *testing.T) {
	ctx := setupTestContext(t)

	// Create messengers via registry
	ten := tenant.MustFromContext(ctx)
	r := GetRegistry()
	_ = r.Create(ten, 100)
	_ = r.Create(ten, 200)

	// Test ProcessorImpl.GetSlice
	proc := NewProcessor(nil, ctx)
	result, err := proc.GetSlice()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 messengers, got %d", len(result))
	}
}
