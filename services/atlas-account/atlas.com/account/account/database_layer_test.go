package account

import (
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func setupTestDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	err = db.AutoMigrate(Entity{})
	if err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}
	return db
}

func sampleTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func TestInternalCreate(t *testing.T) {
	db := setupTestDatabase(t)
	testName := "name"
	testPassword := "password"

	st := sampleTenant()
	a, err := create(db)(st, testName, testPassword, 0)
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	if a.TenantId() != st.Id() {
		t.Fatalf("Tenant mismatch. Expected %v, got %v", st.Id(), a.TenantId())
	}

	if a.Name() != testName {
		t.Fatalf("Name mismatch. Expected %v, got %v", testName, a.Name())
	}

	if a.Password() != testPassword {
		t.Fatalf("Password mismatch. Expected %v, got %v", testName, a.Password())
	}
}

func TestInternalUpdate(t *testing.T) {
	db := setupTestDatabase(t)

	testName := "name"
	testPassword := "password"
	testPin := "1234"
	testPic := "4567"

	st := sampleTenant()
	a, err := create(db)(st, testName, testPassword, 0)
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	err = update(db)(updatePin(testPin), updatePic(testPic), updateTos(true), updateGender(1))(st, a.Id())
	if err != nil {
		t.Fatalf("Failed to update account: %v", err)
	}

	re, err := entityById(st, a.id)(db)()
	if err != nil {
		t.Fatalf("Failed to retrieve updated account: %v", err)
	}
	r, err := Make(re)
	if err != nil {
		t.Fatalf("Failed to retrieve updated account: %v", err)
	}

	if r.Name() != testName {
		t.Fatalf("Name mismatch. Expected %v, got %v", testName, r.Name())
	}

	if r.Password() != testPassword {
		t.Fatalf("Password mismatch. Expected %v, got %v", testName, r.Password())
	}

	if r.Pin() != testPin {
		t.Fatalf("Pin mismatch. Expected %v, got %v", testPin, r.Pin())
	}

	if r.Pic() != testPic {
		t.Fatalf("Pic mismatch. Expected %v, got %v", testPic, r.Pic())
	}

	if r.TOS() != true {
		t.Fatalf("TOS mismatch. Expected %v, got %v", true, r.TOS())
	}

	if r.gender != 1 {
		t.Fatalf("Gender mismatch. Expected %v, got %v", 1, r.gender)
	}
}

func TestInternalGetByName(t *testing.T) {
	db := setupTestDatabase(t)

	testName := "name"
	testPassword := "password"

	st := sampleTenant()
	_, err := create(db)(st, testName, testPassword, 0)
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}
	_, err = create(db)(st, "other1", testPassword, 0)
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}
	_, err = create(db)(st, "other2", testPassword, 0)
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}
	_, err = create(db)(st, "other3", testPassword, 0)
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	re, err := entitiesByName(st, testName)(db)()
	if err != nil {
		t.Fatalf("Failed to retrieve updated account: %v", err)
	}

	if len(re) != 1 {
		t.Fatalf("Number of records mismatch. Expected %v, got %v", 1, len(re))
	}

	r, err := Make(re[0])
	if err != nil {
		t.Fatalf("Failed to retrieve account: %v", err)
	}

	if r.Name() != testName {
		t.Fatalf("Name mismatch. Expected %v, got %v", testName, r.Name())
	}

	if r.Password() != testPassword {
		t.Fatalf("Password mismatch. Expected %v, got %v", testName, r.Password())
	}
}
