package account

import (
	"atlas-account/kafka/message"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/bcrypt"
)

func TestCreate(t *testing.T) {
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

func TestGetByTenant(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	mb := message.NewBuffer()
	p := NewProcessor(l, tctx, db)
	_, _ = p.Create(mb)("user1")("password")
	_, _ = p.Create(mb)("user2")("password")
	_, _ = p.Create(mb)("user3")("password")

	accounts, err := p.GetByTenant()
	if err != nil {
		t.Fatalf("Failed to get accounts by tenant: %v", err)
	}

	if len(accounts) != 3 {
		t.Errorf("Expected 3 accounts, got %v", len(accounts))
	}
}

func TestGetByTenantEmpty(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()
	tctx := tenant.WithContext(context.Background(), st)

	p := NewProcessor(l, tctx, db)
	accounts, err := p.GetByTenant()
	if err != nil {
		t.Fatalf("Failed to get accounts by tenant: %v", err)
	}

	if len(accounts) != 0 {
		t.Errorf("Expected 0 accounts, got %v", len(accounts))
	}
}
