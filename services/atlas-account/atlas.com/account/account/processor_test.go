package account

import (
	"atlas-account/kafka/message"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/bcrypt"
)

// allAccounts drains AllProvider's first page (a single page big enough for
// these tests' fixture sizes) and fails the test on error, standing in for
// the deleted unfiltered GetByTenant/ByTenantProvider methods.
func allAccounts(t *testing.T, p Processor) []Model {
	t.Helper()
	paged, err := p.AllProvider(model.Page{Number: 1, Size: 50})()
	if err != nil {
		t.Fatalf("Failed to get accounts: %v", err)
	}
	return paged.Items
}

func TestCreate(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()

	testName := "name"
	testPassword := "password"

	tctx := tenant.WithContext(context.Background(), st)

	mb := message.NewBuffer()
	m, err := NewProcessor(l, tctx, db).Create(mb)(testName)(testPassword)
	if err != nil {
		t.Fatalf("Unable to create account: %v", err)
	}

	if m.Name() != testName {
		t.Fatalf("Name does not match")
	}

	if bcrypt.CompareHashAndPassword([]byte(m.Password()), []byte(testPassword)) != nil {
		t.Fatalf("Password does not match")
	}
}

func TestGetById(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	mb := message.NewBuffer()
	created, err := NewProcessor(l, tctx, db).Create(mb)("testuser")("password")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	p := NewProcessor(l, tctx, db)
	found, err := p.GetById(created.Id())
	if err != nil {
		t.Fatalf("Failed to get account by id: %v", err)
	}

	if found.Id() != created.Id() {
		t.Errorf("Id mismatch. Expected %v, got %v", created.Id(), found.Id())
	}

	if found.Name() != created.Name() {
		t.Errorf("Name mismatch. Expected %v, got %v", created.Name(), found.Name())
	}
}

func TestGetByIdNotFound(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	p := NewProcessor(l, tctx, db)
	_, err := p.GetById(99999)
	if err == nil {
		t.Fatal("Expected error for non-existent account, got nil")
	}
}

func TestGetByName(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	testName := "uniquename"
	mb := message.NewBuffer()
	created, err := NewProcessor(l, tctx, db).Create(mb)(testName)("password")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	p := NewProcessor(l, tctx, db)
	found, err := p.GetByName(testName)
	if err != nil {
		t.Fatalf("Failed to get account by name: %v", err)
	}

	if found.Id() != created.Id() {
		t.Errorf("Id mismatch. Expected %v, got %v", created.Id(), found.Id())
	}

	if found.Name() != testName {
		t.Errorf("Name mismatch. Expected %v, got %v", testName, found.Name())
	}
}

func TestGetByNameNotFound(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	p := NewProcessor(l, tctx, db)
	_, err := p.GetByName("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent account, got nil")
	}
}

func TestUpdatePin(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	mb := message.NewBuffer()
	created, err := NewProcessor(l, tctx, db).Create(mb)("testuser")("password")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	input, _ := NewBuilder(st.Id(), created.Name()).
		SetPin("1234").
		Build()

	p := NewProcessor(l, tctx, db)
	updated, err := p.Update(created.Id(), input)
	if err != nil {
		t.Fatalf("Failed to update account: %v", err)
	}

	if updated.Pin() != "1234" {
		t.Errorf("Pin mismatch. Expected 1234, got %v", updated.Pin())
	}
}

func TestUpdatePic(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	mb := message.NewBuffer()
	created, err := NewProcessor(l, tctx, db).Create(mb)("testuser")("password")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	input, _ := NewBuilder(st.Id(), created.Name()).
		SetPic("5678").
		Build()

	p := NewProcessor(l, tctx, db)
	updated, err := p.Update(created.Id(), input)
	if err != nil {
		t.Fatalf("Failed to update account: %v", err)
	}

	if updated.Pic() != "5678" {
		t.Errorf("Pic mismatch. Expected 5678, got %v", updated.Pic())
	}
}

func TestUpdateTOS(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	mb := message.NewBuffer()
	created, err := NewProcessor(l, tctx, db).Create(mb)("testuser")("password")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	input, _ := NewBuilder(st.Id(), created.Name()).
		SetTOS(true).
		Build()

	p := NewProcessor(l, tctx, db)
	updated, err := p.Update(created.Id(), input)
	if err != nil {
		t.Fatalf("Failed to update account: %v", err)
	}

	if updated.TOS() != true {
		t.Errorf("TOS mismatch. Expected true, got %v", updated.TOS())
	}
}

func TestUpdateGender(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	mb := message.NewBuffer()
	created, err := NewProcessor(l, tctx, db).Create(mb)("testuser")("password")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	input, _ := NewBuilder(st.Id(), created.Name()).
		SetGender(1).
		Build()

	p := NewProcessor(l, tctx, db)
	updated, err := p.Update(created.Id(), input)
	if err != nil {
		t.Fatalf("Failed to update account: %v", err)
	}

	if updated.Gender() != 1 {
		t.Errorf("Gender mismatch. Expected 1, got %v", updated.Gender())
	}
}

func TestUpdateNoChanges(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	mb := message.NewBuffer()
	created, err := NewProcessor(l, tctx, db).Create(mb)("testuser")("password")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	input, _ := NewBuilder(st.Id(), created.Name()).Build()

	p := NewProcessor(l, tctx, db)
	updated, err := p.Update(created.Id(), input)
	if err != nil {
		t.Fatalf("Failed to update account: %v", err)
	}

	if updated.Id() != created.Id() {
		t.Errorf("Account should be unchanged")
	}
}

func TestUpdateNotFound(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	input, _ := NewBuilder(st.Id(), "test").
		SetPin("1234").
		Build()

	p := NewProcessor(l, tctx, db)
	_, err := p.Update(99999, input)
	if err == nil {
		t.Fatal("Expected error for non-existent account, got nil")
	}
}

func TestAllProvider(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	mb := message.NewBuffer()
	p := NewProcessor(l, tctx, db)
	_, _ = p.Create(mb)("user1")("password")
	_, _ = p.Create(mb)("user2")("password")
	_, _ = p.Create(mb)("user3")("password")

	accounts := allAccounts(t, p)

	if len(accounts) != 3 {
		t.Errorf("Expected 3 accounts, got %v", len(accounts))
	}
}

func TestAllProviderEmpty(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	p := NewProcessor(l, tctx, db)
	accounts := allAccounts(t, p)

	if len(accounts) != 0 {
		t.Errorf("Expected 0 accounts, got %v", len(accounts))
	}
}

// TestLoggedInTenantProviderDrainsBeyondOnePage proves LoggedInTenantProvider
// (used by teardown to log out every account in the tenant) does not
// silently truncate at the first AllProvider page: it seeds more accounts
// than tenantDrainPageSize and logs in only the very last one created, then
// asserts drainAll still surfaces it.
func TestLoggedInTenantProviderDrainsBeyondOnePage(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	p := NewProcessor(l, tctx, db)

	// Seed via the entity-layer create() helper (no bcrypt hashing) rather
	// than p.Create, which would be needlessly slow at this volume.
	const total = tenantDrainPageSize + 5
	var lastCreated Model
	for i := 0; i < total; i++ {
		m, err := create(db.WithContext(tctx), st.Id(), fmt.Sprintf("user%d", i), "password", 0)
		if err != nil {
			t.Fatalf("Failed to create account %d: %v", i, err)
		}
		lastCreated = m
	}

	ak := AccountKey{Tenant: st, AccountId: lastCreated.Id()}
	if err := GetRegistry().Login(tctx, ak, ServiceKey{Service: ServiceLogin}); err != nil {
		t.Fatalf("failed to simulate login: %v", err)
	}
	defer GetRegistry().Terminate(tctx, ak)

	loggedIn, err := p.LoggedInTenantProvider()
	if err != nil {
		t.Fatalf("LoggedInTenantProvider failed: %v", err)
	}

	if len(loggedIn) != 1 {
		t.Fatalf("Expected 1 logged-in account beyond page 1, got %d", len(loggedIn))
	}
	if loggedIn[0].Id() != lastCreated.Id() {
		t.Fatalf("Expected logged-in account [%d], got [%d]", lastCreated.Id(), loggedIn[0].Id())
	}
}

func TestDelete(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	// Create an account
	mb := message.NewBuffer()
	created, err := NewProcessor(l, tctx, db).Create(mb)("testuser")("password")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Verify account exists
	p := NewProcessor(l, tctx, db)
	_, err = p.GetById(created.Id())
	if err != nil {
		t.Fatalf("Account should exist before deletion: %v", err)
	}

	// Delete the account
	mb = message.NewBuffer()
	err = p.Delete(mb)(created.Id())
	if err != nil {
		t.Fatalf("Failed to delete account: %v", err)
	}

	// Verify account no longer exists
	_, err = p.GetById(created.Id())
	if err == nil {
		t.Fatal("Account should not exist after deletion")
	}
}

func TestDeleteNotFound(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	p := NewProcessor(l, tctx, db)
	mb := message.NewBuffer()
	err := p.Delete(mb)(99999)

	if err == nil {
		t.Fatal("Expected error when deleting non-existent account")
	}

	if !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("Expected ErrAccountNotFound, got: %v", err)
	}
}

func TestDeleteLoggedIn(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	// Create an account
	mb := message.NewBuffer()
	created, err := NewProcessor(l, tctx, db).Create(mb)("testuser")("password")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Simulate login by adding to registry
	ak := AccountKey{Tenant: st, AccountId: created.Id()}
	sk := ServiceKey{Service: ServiceLogin}
	_ = GetRegistry().Login(tctx, ak, sk)

	// Attempt to delete should fail
	p := NewProcessor(l, tctx, db)
	mb = message.NewBuffer()
	err = p.Delete(mb)(created.Id())

	if err == nil {
		t.Fatal("Expected error when deleting logged-in account")
	}

	if !errors.Is(err, ErrAccountLoggedIn) {
		t.Errorf("Expected ErrAccountLoggedIn, got: %v", err)
	}

	// Clean up: logout the account
	GetRegistry().Terminate(tctx, ak)
}

func TestDeleteMultipleAccounts(t *testing.T) {
	setupTestRegistry(t)
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	// Create multiple accounts
	mb := message.NewBuffer()
	p := NewProcessor(l, tctx, db)
	created1, _ := p.Create(mb)("user1")("password")
	created2, _ := p.Create(mb)("user2")("password")
	created3, _ := p.Create(mb)("user3")("password")

	// Verify all exist
	accounts := allAccounts(t, p)
	if len(accounts) != 3 {
		t.Fatalf("Expected 3 accounts, got %v", len(accounts))
	}

	// Delete middle account
	mb = message.NewBuffer()
	err := p.Delete(mb)(created2.Id())
	if err != nil {
		t.Fatalf("Failed to delete account: %v", err)
	}

	// Verify only 2 remain
	accounts = allAccounts(t, p)
	if len(accounts) != 2 {
		t.Fatalf("Expected 2 accounts after deletion, got %v", len(accounts))
	}

	// Verify correct accounts remain
	_, err = p.GetById(created1.Id())
	if err != nil {
		t.Errorf("Account 1 should still exist")
	}
	_, err = p.GetById(created2.Id())
	if err == nil {
		t.Errorf("Account 2 should be deleted")
	}
	_, err = p.GetById(created3.Id())
	if err != nil {
		t.Errorf("Account 3 should still exist")
	}
}
