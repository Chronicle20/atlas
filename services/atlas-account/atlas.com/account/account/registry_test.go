package account

import (
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"testing"
)

func TestCoordinator(t *testing.T) {
	c := Get()
	tenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ak := AccountKey{Tenant: tenant, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}
	s2 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}
	_ = ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}

	var err error
	if c.IsLoggedIn(ak) {
		t.Error("IsLoggedIn should return false. not logged in yet")
	}
	err = c.Login(ak, s1)
	if err != nil {
		t.Error(err)
	}
	if !c.IsLoggedIn(ak) {
		t.Error("IsLoggedIn should return true")
	}
	c.Logout(ak, s1)
	if c.IsLoggedIn(ak) {
		t.Error("IsLoggedIn should return false. not logged in yet")
	}
	err = c.Login(ak, s1)
	if err != nil {
		t.Error(err)
	}
	if !c.IsLoggedIn(ak) {
		t.Error("IsLoggedIn should return true")
	}
	err = c.Transition(ak, s1)
	if err != nil {
		t.Error(err)
	}
	if !c.IsLoggedIn(ak) {
		t.Error("IsLoggedIn should return true")
	}
	err = c.Login(ak, s2)
	if err != nil {
		t.Error(err)
	}
	if !c.IsLoggedIn(ak) {
		t.Error("IsLoggedIn should return true")
	}
	c.Logout(ak, s1)
	if !c.IsLoggedIn(ak) {
		t.Error("IsLoggedIn should return true")
	}
}

func TestHappyPath(t *testing.T) {
	c := Get()
	tenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ak := AccountKey{Tenant: tenant, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}
	s2 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}

	var err error

	err = c.Login(ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = c.Transition(ak, s1)
	if err != nil {
		t.Error(err)
	}
	c.Logout(ak, s1)
	err = c.Login(ak, s2)
	if err != nil {
		t.Error(err)
	}
	if !c.IsLoggedIn(ak) {
		t.Error("IsLoggedIn should return true")
	}
}

func TestUnhappyPath(t *testing.T) {
	c := Get()
	tenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ak := AccountKey{Tenant: tenant, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}
	s2 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}

	var err error

	err = c.Login(ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = c.Transition(ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = c.Login(ak, s2)
	if err != nil {
		t.Error(err)
	}
	c.Logout(ak, s1)
	if !c.IsLoggedIn(ak) {
		t.Error("IsLoggedIn should return true")
	}
}

func TestChangeChannelHappy(t *testing.T) {
	c := Get()
	tenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ak := AccountKey{Tenant: tenant, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}
	s2 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}
	s3 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}

	var err error

	err = c.Login(ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = c.Transition(ak, s1)
	if err != nil {
		t.Error(err)
	}
	c.Logout(ak, s1)

	err = c.Login(ak, s2)
	if err != nil {
		t.Error(err)
	}
	err = c.Transition(ak, s2)
	if err != nil {
		t.Error(err)
	}
	c.Logout(ak, s2)
	err = c.Login(ak, s3)
	if err != nil {
		t.Error(err)
	}
	if !c.IsLoggedIn(ak) {
		t.Error("IsLoggedIn should return true")
	}
}

func TestChangeChannelUnhappy(t *testing.T) {
	c := Get()
	tenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ak := AccountKey{Tenant: tenant, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}
	s2 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}
	s3 := ServiceKey{SessionId: uuid.New(), Service: ServiceChannel}

	var err error

	err = c.Login(ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = c.Transition(ak, s1)
	if err != nil {
		t.Error(err)
	}
	c.Logout(ak, s1)
	err = c.Login(ak, s2)
	if err != nil {
		t.Error(err)
	}
	err = c.Transition(ak, s2)
	if err != nil {
		t.Error(err)
	}
	err = c.Login(ak, s3)
	if err != nil {
		t.Error(err)
	}
	c.Logout(ak, s2)
	if !c.IsLoggedIn(ak) {
		t.Error("IsLoggedIn should return true")
	}
}

func TestDoubleLogin(t *testing.T) {
	c := Get()
	tenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ak := AccountKey{Tenant: tenant, AccountId: 1}
	s1 := ServiceKey{SessionId: uuid.New(), Service: ServiceLogin}

	var err error

	err = c.Login(ak, s1)
	if err != nil {
		t.Error(err)
	}
	err = c.Login(ak, s1)
	if err == nil {
		t.Errorf("double login should return an error")
	}
}
