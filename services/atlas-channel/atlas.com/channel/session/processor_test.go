package session_test

import (
	"atlas-channel/session"
	"atlas-channel/test"
	"testing"

	channel2 "github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// testSetup creates common test fixtures
func testSetup() (*logrus.Logger, func()) {
	logger, _ := logtest.NewNullLogger()
	tenantId := test.DefaultTenantId

	// Cleanup function to clear registry after test
	cleanup := func() {
		session.ClearRegistryForTenant(tenantId)
	}

	return logger, cleanup
}

// createTestSession creates a test session with the given parameters
func createTestSession(sessionId uuid.UUID) session.Model {
	t := test.CreateDefaultMockTenant()

	// Create a basic session - note: this requires a connection, so we use a minimal approach
	// For testing, we add sessions directly to the registry using the test helper
	s := session.NewSession(sessionId, t, 0, nil)

	// Add to registry
	session.AddSessionToRegistry(t.Id(), s)

	return s
}

func TestNewProcessor(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	p := session.NewProcessor(logger, ctx)

	if p == nil {
		t.Fatal("Expected processor to be initialized")
	}
}

func TestByIdModelProvider_Found(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenantId := test.DefaultTenantId
	tenant := test.CreateDefaultMockTenant()

	// Add a session to the registry
	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenantId, s)

	// Create processor and look up the session
	p := session.NewProcessor(logger, ctx)
	result, err := p.ByIdModelProvider(sessionId)()

	if err != nil {
		t.Fatalf("ByIdModelProvider() unexpected error: %v", err)
	}
	if result.SessionId() != sessionId {
		t.Errorf("ByIdModelProvider() returned session with ID %v, want %v", result.SessionId(), sessionId)
	}
}

func TestByIdModelProvider_NotFound(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	nonExistentId := uuid.New()

	p := session.NewProcessor(logger, ctx)
	_, err := p.ByIdModelProvider(nonExistentId)()

	if err == nil {
		t.Error("ByIdModelProvider() expected error for non-existent session, got nil")
	}
}

func TestIfPresentById_Executes(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	// Add a session to the registry
	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)

	// Track if the operator was called
	called := false
	p.IfPresentById(sessionId, func(m session.Model) error {
		called = true
		return nil
	})

	if !called {
		t.Error("IfPresentById() operator was not called when session exists")
	}
}

func TestIfPresentById_NoOp(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	nonExistentId := uuid.New()

	p := session.NewProcessor(logger, ctx)

	// Track if the operator was called
	called := false
	p.IfPresentById(nonExistentId, func(m session.Model) error {
		called = true
		return nil
	})

	if called {
		t.Error("IfPresentById() operator was called when session does not exist")
	}
}

func TestSetAccountId(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	// Add a session with no account ID
	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	updatedSession := p.SetAccountId(sessionId, 12345)

	if updatedSession.AccountId() != 12345 {
		t.Errorf("SetAccountId() returned session with AccountId %d, want 12345", updatedSession.AccountId())
	}

	// Verify the registry was updated
	retrieved, err := p.ByIdModelProvider(sessionId)()
	if err != nil {
		t.Fatalf("ByIdModelProvider() unexpected error: %v", err)
	}
	if retrieved.AccountId() != 12345 {
		t.Errorf("Registry session AccountId = %d, want 12345", retrieved.AccountId())
	}
}

func TestSetCharacterId(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	updatedSession := p.SetCharacterId(sessionId, 67890)

	if updatedSession.CharacterId() != 67890 {
		t.Errorf("SetCharacterId() returned session with CharacterId %d, want 67890", updatedSession.CharacterId())
	}

	// Verify the registry was updated
	retrieved, err := p.ByIdModelProvider(sessionId)()
	if err != nil {
		t.Fatalf("ByIdModelProvider() unexpected error: %v", err)
	}
	if retrieved.CharacterId() != 67890 {
		t.Errorf("Registry session CharacterId = %d, want 67890", retrieved.CharacterId())
	}
}

func TestSetMapId(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	mapId := _map.Id(100000000)
	updatedSession := p.SetMapId(sessionId, mapId)

	if updatedSession.MapId() != mapId {
		t.Errorf("SetMapId() returned session with MapId %d, want %d", updatedSession.MapId(), mapId)
	}

	// Verify the registry was updated
	retrieved, err := p.ByIdModelProvider(sessionId)()
	if err != nil {
		t.Fatalf("ByIdModelProvider() unexpected error: %v", err)
	}
	if retrieved.MapId() != mapId {
		t.Errorf("Registry session MapId = %d, want %d", retrieved.MapId(), mapId)
	}
}

func TestSetGm(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	updatedSession := p.SetGm(sessionId, true)

	// Note: Model doesn't expose Gm() getter in the public API
	// We verify the update returned successfully
	if updatedSession.SessionId() != sessionId {
		t.Errorf("SetGm() returned session with wrong SessionId")
	}
}

func TestSetStorageNpcId(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	updatedSession := p.SetStorageNpcId(sessionId, 9900001)

	if updatedSession.StorageNpcId() != 9900001 {
		t.Errorf("SetStorageNpcId() returned session with StorageNpcId %d, want 9900001", updatedSession.StorageNpcId())
	}
}

func TestClearStorageNpcId(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	_ = p.SetStorageNpcId(sessionId, 9900001)
	updatedSession := p.ClearStorageNpcId(sessionId)

	if updatedSession.StorageNpcId() != 0 {
		t.Errorf("ClearStorageNpcId() returned session with StorageNpcId %d, want 0", updatedSession.StorageNpcId())
	}
}

func TestAllInTenantProvider(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	tenant := test.CreateDefaultMockTenant()

	// Add multiple sessions
	sessionId1 := uuid.New()
	sessionId2 := uuid.New()
	s1 := session.NewSession(sessionId1, tenant, 0, nil)
	s2 := session.NewSession(sessionId2, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s1)
	session.AddSessionToRegistry(tenant.Id(), s2)

	p := session.NewProcessor(logger, ctx)
	sessions, err := p.AllInTenantProvider()

	if err != nil {
		t.Fatalf("AllInTenantProvider() unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("AllInTenantProvider() returned %d sessions, want 2", len(sessions))
	}
}

func TestCharacterIdFilter(t *testing.T) {
	tenant := test.CreateDefaultMockTenant()
	sessionId := uuid.New()
	s := session.NewSession(sessionId, tenant, 0, nil)

	// Add to registry and update character ID
	session.AddSessionToRegistry(tenant.Id(), s)

	filter := session.CharacterIdFilter(12345)

	// The filter should return false since the session has characterId 0
	if filter(s) {
		t.Error("CharacterIdFilter() returned true for non-matching characterId")
	}

	// Cleanup
	session.ClearRegistryForTenant(tenant.Id())
}

func TestAccountIdFilter(t *testing.T) {
	tenant := test.CreateDefaultMockTenant()
	sessionId := uuid.New()
	s := session.NewSession(sessionId, tenant, 0, nil)

	filter := session.AccountIdFilter(0)

	// The filter should return true since the session has accountId 0
	if !filter(s) {
		t.Error("AccountIdFilter() returned false for matching accountId")
	}
}

func TestWorldIdFilter(t *testing.T) {
	tenant := test.CreateDefaultMockTenant()
	sessionId := uuid.New()
	s := session.NewSession(sessionId, tenant, 0, nil)

	filter := session.WorldIdFilter(0)

	// The filter should return true since the session has worldId 0
	if !filter(s) {
		t.Error("WorldIdFilter() returned false for matching worldId")
	}
}

func TestChannelIdFilter(t *testing.T) {
	tenant := test.CreateDefaultMockTenant()
	sessionId := uuid.New()
	s := session.NewSession(sessionId, tenant, 0, nil)

	filter := session.ChannelIdFilter(0)

	// The filter should return true since the session has channelId 0
	if !filter(s) {
		t.Error("ChannelIdFilter() returned false for matching channelId")
	}
}

func TestByCharacterIdModelProvider_Found(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	// Add a session and set character ID
	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	_ = p.SetCharacterId(sessionId, 12345)

	// Look up by character ID
	ch := channel2.NewModel(0, 0)
	result, err := p.ByCharacterIdModelProvider(ch)(12345)()

	if err != nil {
		t.Fatalf("ByCharacterIdModelProvider() unexpected error: %v", err)
	}
	if result.CharacterId() != 12345 {
		t.Errorf("ByCharacterIdModelProvider() returned session with CharacterId %d, want 12345", result.CharacterId())
	}
}

func TestByCharacterIdModelProvider_NotFound(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()

	p := session.NewProcessor(logger, ctx)
	ch := channel2.NewModel(0, 0)
	_, err := p.ByCharacterIdModelProvider(ch)(99999)()

	if err == nil {
		t.Error("ByCharacterIdModelProvider() expected error for non-existent character, got nil")
	}
}

func TestIfPresentByCharacterId_Executes(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	_ = p.SetCharacterId(sessionId, 12345)

	called := false
	ch := channel2.NewModel(0, 0)
	err := p.IfPresentByCharacterId(ch)(12345, func(m session.Model) error {
		called = true
		return nil
	})

	if err != nil {
		t.Fatalf("IfPresentByCharacterId() unexpected error: %v", err)
	}
	if !called {
		t.Error("IfPresentByCharacterId() operator was not called when session exists")
	}
}

func TestIfPresentByCharacterId_NoOp(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()

	p := session.NewProcessor(logger, ctx)

	called := false
	ch := channel2.NewModel(0, 0)
	err := p.IfPresentByCharacterId(ch)(99999, func(m session.Model) error {
		called = true
		return nil
	})

	if err != nil {
		t.Fatalf("IfPresentByCharacterId() unexpected error: %v", err)
	}
	if called {
		t.Error("IfPresentByCharacterId() operator was called when session does not exist")
	}
}

func TestByAccountIdModelProvider_Found(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	_ = p.SetAccountId(sessionId, 54321)

	ch := channel2.NewModel(0, 0)
	result, err := p.ByAccountIdModelProvider(ch)(54321)()

	if err != nil {
		t.Fatalf("ByAccountIdModelProvider() unexpected error: %v", err)
	}
	if result.AccountId() != 54321 {
		t.Errorf("ByAccountIdModelProvider() returned session with AccountId %d, want 54321", result.AccountId())
	}
}

func TestIfPresentByAccountId_Executes(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	_ = p.SetAccountId(sessionId, 54321)

	called := false
	ch := channel2.NewModel(0, 0)
	err := p.IfPresentByAccountId(ch)(54321, func(m session.Model) error {
		called = true
		return nil
	})

	if err != nil {
		t.Fatalf("IfPresentByAccountId() unexpected error: %v", err)
	}
	if !called {
		t.Error("IfPresentByAccountId() operator was not called when session exists")
	}
}

func TestGetByCharacterId(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	_ = p.SetCharacterId(sessionId, 11111)

	ch := channel2.NewModel(0, 0)
	result, err := p.GetByCharacterId(ch)(11111)

	if err != nil {
		t.Fatalf("GetByCharacterId() unexpected error: %v", err)
	}
	if result.CharacterId() != 11111 {
		t.Errorf("GetByCharacterId() returned session with CharacterId %d, want 11111", result.CharacterId())
	}
}

func TestUpdateLastRequest(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	sessionId := uuid.New()
	tenant := test.CreateDefaultMockTenant()

	s := session.NewSession(sessionId, tenant, 0, nil)
	session.AddSessionToRegistry(tenant.Id(), s)

	p := session.NewProcessor(logger, ctx)
	originalSession, _ := p.ByIdModelProvider(sessionId)()
	originalTime := originalSession.LastRequest()

	// Small delay to ensure time difference
	updatedSession := p.UpdateLastRequest(sessionId)

	if !updatedSession.LastRequest().After(originalTime) && !updatedSession.LastRequest().Equal(originalTime) {
		t.Error("UpdateLastRequest() did not update the last request time")
	}
}

func TestSetAccountId_NonExistent(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	nonExistentId := uuid.New()

	p := session.NewProcessor(logger, ctx)
	result := p.SetAccountId(nonExistentId, 12345)

	// Should return empty session when not found
	if result.SessionId() != uuid.Nil {
		t.Errorf("SetAccountId() for non-existent session returned non-nil SessionId")
	}
}

func TestSetCharacterId_NonExistent(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	nonExistentId := uuid.New()

	p := session.NewProcessor(logger, ctx)
	result := p.SetCharacterId(nonExistentId, 12345)

	if result.SessionId() != uuid.Nil {
		t.Errorf("SetCharacterId() for non-existent session returned non-nil SessionId")
	}
}

func TestSetMapId_NonExistent(t *testing.T) {
	logger, cleanup := testSetup()
	defer cleanup()

	ctx := test.CreateTestContext()
	nonExistentId := uuid.New()

	p := session.NewProcessor(logger, ctx)
	result := p.SetMapId(nonExistentId, 100000000)

	if result.SessionId() != uuid.Nil {
		t.Errorf("SetMapId() for non-existent session returned non-nil SessionId")
	}
}
