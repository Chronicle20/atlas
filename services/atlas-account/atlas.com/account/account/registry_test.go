package account

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
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
