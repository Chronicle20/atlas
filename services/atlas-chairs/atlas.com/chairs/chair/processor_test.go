package chair

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func resetProcessorRegistry() {
	r := GetRegistry()
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.characterRegister = make(map[uint32]Model)
}

func TestGetById_Success(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	characterId := uint32(12345)
	chairId := uint32(1)
	chairType := "FIXED"

	// Set up registry directly
	GetRegistry().Set(characterId, Model{id: chairId, chairType: chairType})

	// Test GetById
	p := NewProcessor(l, tctx)
	m, err := p.GetById(characterId)

	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}

	if m.Id() != chairId {
		t.Errorf("Expected chair id %d, got %d", chairId, m.Id())
	}

	if m.Type() != chairType {
		t.Errorf("Expected chair type %s, got %s", chairType, m.Type())
	}
}

func TestGetById_NotFound(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	nonExistentCharacter := uint32(99999)

	p := NewProcessor(l, tctx)
	_, err := p.GetById(nonExistentCharacter)

	if err == nil {
		t.Fatal("Expected error for non-existent character, got nil")
	}
}

func TestGetById_MultipleCharacters(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	// Set up multiple chairs
	chars := []struct {
		characterId uint32
		chairId     uint32
		chairType   string
	}{
		{100, 0, "FIXED"},
		{200, 3010001, "PORTABLE"},
		{300, 2, "FIXED"},
	}

	for _, c := range chars {
		GetRegistry().Set(c.characterId, Model{id: c.chairId, chairType: c.chairType})
	}

	p := NewProcessor(l, tctx)

	// Verify each character's chair
	for _, c := range chars {
		m, err := p.GetById(c.characterId)
		if err != nil {
			t.Errorf("GetById(%d) failed: %v", c.characterId, err)
			continue
		}
		if m.Id() != c.chairId {
			t.Errorf("Character %d: expected chair id %d, got %d", c.characterId, c.chairId, m.Id())
		}
		if m.Type() != c.chairType {
			t.Errorf("Character %d: expected chair type %s, got %s", c.characterId, c.chairType, m.Type())
		}
	}
}

func TestGetById_AfterClear(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	characterId := uint32(12345)

	// Set up then clear
	GetRegistry().Set(characterId, Model{id: 1, chairType: "FIXED"})
	GetRegistry().Clear(characterId)

	p := NewProcessor(l, tctx)
	_, err := p.GetById(characterId)

	if err == nil {
		t.Fatal("Expected error after clear, got nil")
	}
}

// Note: Set and Clear methods have external dependencies (Kafka producer, external map service)
// that would require mocking for full unit test coverage.
//
// To properly test Set():
// - Mock the Kafka producer (producer.ProviderImpl)
// - Mock the data/map processor (_map2.NewProcessor)
//
// To properly test Clear():
// - Mock the Kafka producer (producer.ProviderImpl)
//
// For now, these methods are tested indirectly through:
// 1. Integration tests at the service level
// 2. Registry tests that verify the state management
//
// The processor logic that can be unit tested (GetById) is covered above.
// The validation logic in Set (chair existence, item category checks) would
// require dependency injection to mock properly.

func TestModel_Accessors(t *testing.T) {
	chairId := uint32(42)
	chairType := "PORTABLE"

	m := Model{id: chairId, chairType: chairType}

	if m.Id() != chairId {
		t.Errorf("Id() expected %d, got %d", chairId, m.Id())
	}

	if m.Type() != chairType {
		t.Errorf("Type() expected %s, got %s", chairType, m.Type())
	}
}

func TestModel_FixedChairTypes(t *testing.T) {
	testCases := []struct {
		name      string
		id        uint32
		chairType string
	}{
		{"Fixed chair 0", 0, "FIXED"},
		{"Fixed chair 1", 1, "FIXED"},
		{"Fixed chair 10", 10, "FIXED"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{id: tc.id, chairType: tc.chairType}
			if m.Id() != tc.id {
				t.Errorf("Expected id %d, got %d", tc.id, m.Id())
			}
			if m.Type() != tc.chairType {
				t.Errorf("Expected type %s, got %s", tc.chairType, m.Type())
			}
		})
	}
}

func TestModel_PortableChairTypes(t *testing.T) {
	// Portable chairs have item IDs in the 301xxxx range
	testCases := []struct {
		name      string
		id        uint32
		chairType string
	}{
		{"Portable chair 3010000", 3010000, "PORTABLE"},
		{"Portable chair 3010001", 3010001, "PORTABLE"},
		{"Portable chair 3019999", 3019999, "PORTABLE"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{id: tc.id, chairType: tc.chairType}
			if m.Id() != tc.id {
				t.Errorf("Expected id %d, got %d", tc.id, m.Id())
			}
			if m.Type() != tc.chairType {
				t.Errorf("Expected type %s, got %s", tc.chairType, m.Type())
			}
		})
	}
}
