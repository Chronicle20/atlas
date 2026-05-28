package account

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func TestCoordinator(t *testing.T) {
	setupTestRegistry(t)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	ak := AccountKey{Tenant: ten, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}
	s2 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}
	_ = ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}

	var err error
	if GetRegistry().IsLoggedIn(ctx, ak) {
		t.Error("IsLoggedIn should return false. not logged in yet")
	}
	err = GetRegistry().Login(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	if !GetRegistry().IsLoggedIn(ctx, ak) {
		t.Error("IsLoggedIn should return true")
	}
	GetRegistry().Logout(ctx, ak, s1)
	if GetRegistry().IsLoggedIn(ctx, ak) {
		t.Error("IsLoggedIn should return false. not logged in yet")
	}
	err = GetRegistry().Login(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	if !GetRegistry().IsLoggedIn(ctx, ak) {
		t.Error("IsLoggedIn should return true")
	}
	err = GetRegistry().Transition(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	if !GetRegistry().IsLoggedIn(ctx, ak) {
		t.Error("IsLoggedIn should return true")
	}
	err = GetRegistry().Login(ctx, ak, s2)
	if err != nil {
		t.Error(err)
	}
	if !GetRegistry().IsLoggedIn(ctx, ak) {
		t.Error("IsLoggedIn should return true")
	}
	GetRegistry().Logout(ctx, ak, s1)
	if !GetRegistry().IsLoggedIn(ctx, ak) {
		t.Error("IsLoggedIn should return true")
	}
}

func TestHappyPath(t *testing.T) {
	setupTestRegistry(t)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	ak := AccountKey{Tenant: ten, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}
	s2 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}

	var err error

	err = GetRegistry().Login(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = GetRegistry().Transition(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	GetRegistry().Logout(ctx, ak, s1)
	err = GetRegistry().Login(ctx, ak, s2)
	if err != nil {
		t.Error(err)
	}
	if !GetRegistry().IsLoggedIn(ctx, ak) {
		t.Error("IsLoggedIn should return true")
	}
}

func TestUnhappyPath(t *testing.T) {
	setupTestRegistry(t)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	ak := AccountKey{Tenant: ten, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}
	s2 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}

	var err error

	err = GetRegistry().Login(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = GetRegistry().Transition(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = GetRegistry().Login(ctx, ak, s2)
	if err != nil {
		t.Error(err)
	}
	GetRegistry().Logout(ctx, ak, s1)
	if !GetRegistry().IsLoggedIn(ctx, ak) {
		t.Error("IsLoggedIn should return true")
	}
}

func TestChangeChannelHappy(t *testing.T) {
	setupTestRegistry(t)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	ak := AccountKey{Tenant: ten, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}
	s2 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}
	s3 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}

	var err error

	err = GetRegistry().Login(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = GetRegistry().Transition(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	GetRegistry().Logout(ctx, ak, s1)

	err = GetRegistry().Login(ctx, ak, s2)
	if err != nil {
		t.Error(err)
	}
	err = GetRegistry().Transition(ctx, ak, s2)
	if err != nil {
		t.Error(err)
	}
	GetRegistry().Logout(ctx, ak, s2)
	err = GetRegistry().Login(ctx, ak, s3)
	if err != nil {
		t.Error(err)
	}
	if !GetRegistry().IsLoggedIn(ctx, ak) {
		t.Error("IsLoggedIn should return true")
	}
}

func TestChangeChannelUnhappy(t *testing.T) {
	setupTestRegistry(t)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	ak := AccountKey{Tenant: ten, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}
	s2 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}
	s3 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}

	var err error

	err = GetRegistry().Login(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = GetRegistry().Transition(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	GetRegistry().Logout(ctx, ak, s1)
	err = GetRegistry().Login(ctx, ak, s2)
	if err != nil {
		t.Error(err)
	}
	err = GetRegistry().Transition(ctx, ak, s2)
	if err != nil {
		t.Error(err)
	}
	err = GetRegistry().Login(ctx, ak, s3)
	if err != nil {
		t.Error(err)
	}
	GetRegistry().Logout(ctx, ak, s2)
	if !GetRegistry().IsLoggedIn(ctx, ak) {
		t.Error("IsLoggedIn should return true")
	}
}

func TestDoubleLogin(t *testing.T) {
	setupTestRegistry(t)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	ak := AccountKey{Tenant: ten, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}

	var err error

	err = GetRegistry().Login(ctx, ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = GetRegistry().Login(ctx, ak, s1)
	if err == nil {
		t.Errorf("double login should return an error")
	}
}

// TestGetExpiredInTransition_ExpiredReturned verifies that an account in
// StateTransition with an UpdatedAt old enough to exceed the timeout is
// returned by GetExpiredInTransition.
func TestGetExpiredInTransition_ExpiredReturned(t *testing.T) {
	setupTestRegistry(t)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	ak := AccountKey{Tenant: ten, AccountId: 42}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}

	if err := GetRegistry().Login(ctx, ak, s1); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if err := GetRegistry().Transition(ctx, ak, s1); err != nil {
		t.Fatalf("Transition: %v", err)
	}

	// A timeout of 0 means any non-zero elapsed time qualifies as expired.
	expired := GetRegistry().GetExpiredInTransition(context.Background(), 0)
	if len(expired) != 1 {
		t.Fatalf("GetExpiredInTransition() = %d entries, want 1", len(expired))
	}
	if expired[0].AccountId != ak.AccountId {
		t.Errorf("AccountId = %d, want %d", expired[0].AccountId, ak.AccountId)
	}
}

// TestGetExpiredInTransition_FreshNotReturned verifies that an account in
// StateTransition that has not yet exceeded the timeout is NOT returned.
func TestGetExpiredInTransition_FreshNotReturned(t *testing.T) {
	setupTestRegistry(t)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	ak := AccountKey{Tenant: ten, AccountId: 43}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}

	if err := GetRegistry().Login(ctx, ak, s1); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if err := GetRegistry().Transition(ctx, ak, s1); err != nil {
		t.Fatalf("Transition: %v", err)
	}

	// A very large timeout means the transition is not yet expired.
	expired := GetRegistry().GetExpiredInTransition(context.Background(), 24*time.Hour)
	if len(expired) != 0 {
		t.Errorf("GetExpiredInTransition() = %d entries, want 0", len(expired))
	}
}

// TestGetExpiredInTransition_MultiTenant verifies that expired accounts across
// two separate tenants are both returned.
func TestGetExpiredInTransition_MultiTenant(t *testing.T) {
	setupTestRegistry(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx1 := tenant.WithContext(context.Background(), ten1)
	ctx2 := tenant.WithContext(context.Background(), ten2)

	ak1 := AccountKey{Tenant: ten1, AccountId: 100}
	ak2 := AccountKey{Tenant: ten2, AccountId: 200}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}
	s2 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}

	if err := GetRegistry().Login(ctx1, ak1, s1); err != nil {
		t.Fatalf("Login ten1: %v", err)
	}
	if err := GetRegistry().Transition(ctx1, ak1, s1); err != nil {
		t.Fatalf("Transition ten1: %v", err)
	}
	if err := GetRegistry().Login(ctx2, ak2, s2); err != nil {
		t.Fatalf("Login ten2: %v", err)
	}
	if err := GetRegistry().Transition(ctx2, ak2, s2); err != nil {
		t.Fatalf("Transition ten2: %v", err)
	}

	expired := GetRegistry().GetExpiredInTransition(context.Background(), 0)
	if len(expired) != 2 {
		t.Fatalf("GetExpiredInTransition() = %d entries, want 2", len(expired))
	}
	accountIds := map[uint32]bool{expired[0].AccountId: true, expired[1].AccountId: true}
	if !accountIds[ak1.AccountId] || !accountIds[ak2.AccountId] {
		t.Errorf("unexpected account IDs in result: %v", accountIds)
	}
}
