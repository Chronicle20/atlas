package account

import (
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func TestCreate(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := setupTestDatabase(t)
	span := mocktracer.New().StartSpan("test")
	st := sampleTenant()

	testName := "name"
	testPassword := "password"

	m, err := Create(l, db, span, st)(testName, testPassword)
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
