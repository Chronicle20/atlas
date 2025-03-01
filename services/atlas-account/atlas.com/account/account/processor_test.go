package account

import (
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func TestCreate(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	st := sampleTenant()

	testName := "name"
	testPassword := "password"

	tctx := tenant.WithContext(context.Background(), st)

	m, err := Create(l, db, tctx)(testName, testPassword)
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
