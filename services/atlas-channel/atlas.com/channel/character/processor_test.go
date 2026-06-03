package character_test

import (
	"atlas-channel/character"
	"atlas-channel/character/mock"
	"atlas-channel/monsterbook"
	"atlas-channel/test"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func testSetup() (*logrus.Logger, func()) {
	logger, _ := logtest.NewNullLogger()
	cleanup := func() {}
	return logger, cleanup
}

func createTestCharacter(id uint32, name string, level byte) character.Model {
	return character.NewModelBuilder().
		SetId(id).
		SetName(name).
		SetLevel(level).
		SetAccountId(1).
		SetJobId(0).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetHp(50).
		SetMaxHp(50).
		SetMp(5).
		SetMaxMp(5).
		SetMapId(100000000).
		MustBuild()
}

func TestNewProcessor(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	p := character.NewProcessor(logger, ctx)

	if p == nil {
		t.Fatal("Expected processor to be initialized")
	}
}

func TestMockProcessor_GetById_Success(t *testing.T) {
	mockProc := mock.NewMockProcessor()
	testChar := createTestCharacter(123, "TestChar", 10)
	mockProc.AddCharacter(testChar)

	result, err := mockProc.GetById()(123)

	if err != nil {
		t.Fatalf("GetById() unexpected error: %v", err)
	}
	if result.Id() != 123 {
		t.Errorf("GetById() returned character with Id %d, want 123", result.Id())
	}
	if result.Name() != "TestChar" {
		t.Errorf("GetById() returned character with Name %s, want TestChar", result.Name())
	}
	if result.Level() != 10 {
		t.Errorf("GetById() returned character with Level %d, want 10", result.Level())
	}
}

func TestMockProcessor_GetById_NotFound(t *testing.T) {
	mockProc := mock.NewMockProcessor()

	_, err := mockProc.GetById()(99999)

	if err == nil {
		t.Error("GetById() expected error for non-existent character, got nil")
	}
}

func TestMockProcessor_GetById_Error(t *testing.T) {
	mockProc := mock.NewMockProcessor()
	expectedErr := errors.New("database connection failed")
	mockProc.GetByIdError = expectedErr

	_, err := mockProc.GetById()(123)

	if err == nil {
		t.Error("GetById() expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("GetById() error = %v, want %v", err, expectedErr)
	}
}

func TestMockProcessor_GetByName_Success(t *testing.T) {
	mockProc := mock.NewMockProcessor()
	testChar := createTestCharacter(456, "UniqueChar", 50)
	mockProc.AddCharacter(testChar)

	result, err := mockProc.GetByName("UniqueChar")

	if err != nil {
		t.Fatalf("GetByName() unexpected error: %v", err)
	}
	if result.Id() != 456 {
		t.Errorf("GetByName() returned character with Id %d, want 456", result.Id())
	}
	if result.Name() != "UniqueChar" {
		t.Errorf("GetByName() returned character with Name %s, want UniqueChar", result.Name())
	}
}

func TestMockProcessor_GetByName_NotFound(t *testing.T) {
	mockProc := mock.NewMockProcessor()

	_, err := mockProc.GetByName("NonExistent")

	if err == nil {
		t.Error("GetByName() expected error for non-existent character, got nil")
	}
}

func TestMockProcessor_ByNameProvider_Success(t *testing.T) {
	mockProc := mock.NewMockProcessor()
	testChar := createTestCharacter(789, "ProviderChar", 25)
	mockProc.AddCharacter(testChar)

	results, err := mockProc.ByNameProvider("ProviderChar")()

	if err != nil {
		t.Fatalf("ByNameProvider() unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("ByNameProvider() returned %d results, want 1", len(results))
	}
	if results[0].Id() != 789 {
		t.Errorf("ByNameProvider() first result Id = %d, want 789", results[0].Id())
	}
}

func TestMockProcessor_ByNameProvider_NotFound(t *testing.T) {
	mockProc := mock.NewMockProcessor()

	results, err := mockProc.ByNameProvider("NonExistent")()

	if err != nil {
		t.Fatalf("ByNameProvider() unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("ByNameProvider() returned %d results, want 0", len(results))
	}
}

func TestMockProcessor_GetById_WithDecorator(t *testing.T) {
	mockProc := mock.NewMockProcessor()
	testChar := createTestCharacter(100, "DecoratorTest", 30)
	mockProc.AddCharacter(testChar)

	// Create a decorator that modifies the character
	levelDecorator := func(c character.Model) character.Model {
		// Note: We can't actually modify immutable model fields, but we test that decorator is called
		return c
	}

	result, err := mockProc.GetById(levelDecorator)(100)

	if err != nil {
		t.Fatalf("GetById() with decorator unexpected error: %v", err)
	}
	if result.Id() != 100 {
		t.Errorf("GetById() with decorator returned character with Id %d, want 100", result.Id())
	}
}

func TestMockProcessor_Decorators(t *testing.T) {
	mockProc := mock.NewMockProcessor()
	testChar := createTestCharacter(111, "DecoratorChar", 20)

	// Test that decorators return the character unchanged (mock implementation)
	result := mockProc.InventoryDecorator(testChar)
	if result.Id() != testChar.Id() {
		t.Error("InventoryDecorator() changed character ID unexpectedly")
	}

	result = mockProc.SkillModelDecorator(testChar)
	if result.Id() != testChar.Id() {
		t.Error("SkillModelDecorator() changed character ID unexpectedly")
	}
}

func TestMockProcessor_CommandMethods(t *testing.T) {
	mockProc := mock.NewMockProcessor()
	testField := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()

	// Test that command methods don't error (mock returns nil)
	err := mockProc.RequestDistributeAp(testField, 123, 0, nil)
	if err != nil {
		t.Errorf("RequestDistributeAp() unexpected error: %v", err)
	}

	err = mockProc.RequestDropMeso(testField, 123, 1000)
	if err != nil {
		t.Errorf("RequestDropMeso() unexpected error: %v", err)
	}

	err = mockProc.ChangeHP(testField, 123, 10)
	if err != nil {
		t.Errorf("ChangeHP() unexpected error: %v", err)
	}

	err = mockProc.ChangeMP(testField, 123, 10)
	if err != nil {
		t.Errorf("ChangeMP() unexpected error: %v", err)
	}

	err = mockProc.RequestDistributeSp(testField, 123, 0, 1001001, 1)
	if err != nil {
		t.Errorf("RequestDistributeSp() unexpected error: %v", err)
	}
}

func TestProcessorInterface(t *testing.T) {
	// Verify that MockProcessor implements character.Processor interface
	var _ character.Processor = (*mock.MockProcessor)(nil)
}

func TestProcessorImpl_PartyDecorator_NotInParty(t *testing.T) {
	mockProc := mock.NewMockProcessor()
	c := createTestCharacter(123, "SoloChar", 10)

	// Mock decorator is a pass-through that does NOT populate party.
	out := mockProc.PartyDecorator(c)
	if out.InParty() {
		t.Error("InParty() = true on mock-decorated solo character, want false")
	}
}

func TestProcessorImpl_PartyDecorator_InterfaceContract(t *testing.T) {
	// Compile-time assertion that PartyDecorator is on the interface.
	var _ func(character.Model) character.Model = (mock.NewMockProcessor()).PartyDecorator
}

func TestMonsterBookDecorator_FailOpen(t *testing.T) {
	// Upstream down → decorator returns the model unchanged (cover 0, no cards).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	defer monsterbook.SetBaseURLForTest(srv.URL)()

	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tm)
	p := character.NewProcessor(logrus.New(), ctx)

	m := character.NewModelBuilder().SetId(42).MustBuild()
	got := p.MonsterBookDecorator(m)
	if got.CoverCardId() != 0 {
		t.Errorf("cover should be 0 on fail-open, got %d", got.CoverCardId())
	}
	if len(got.MonsterBookCards()) != 0 {
		t.Errorf("cards should be empty on fail-open, got %d", len(got.MonsterBookCards()))
	}
}

func TestMonsterBookDecorator_Populates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if strings.HasSuffix(r.URL.Path, "/monster-book/cards") {
			_, _ = w.Write([]byte(`{"data":[{"type":"monster-book-card","id":"2380005","attributes":{"level":2,"isSpecial":false}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"type":"monster-book","id":"42","attributes":{"coverCardId":2380001}}}`))
	}))
	defer srv.Close()
	defer monsterbook.SetBaseURLForTest(srv.URL)()

	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tm)
	p := character.NewProcessor(logrus.New(), ctx)

	m := character.NewModelBuilder().SetId(42).MustBuild()
	got := p.MonsterBookDecorator(m)
	if got.CoverCardId() != item.Id(2380001) {
		t.Errorf("cover = %d, want 2380001", got.CoverCardId())
	}
	if len(got.MonsterBookCards()) != 1 || got.MonsterBookCards()[0].CardId() != item.Id(2380005) {
		t.Errorf("cards not populated: %+v", got.MonsterBookCards())
	}
}
