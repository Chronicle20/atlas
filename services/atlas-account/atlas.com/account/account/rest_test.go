package account

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

func TestTransform(t *testing.T) {
	tenantId := uuid.New()
	m, _ := NewBuilder(tenantId, "testuser").
		SetId(123).
		SetPassword("hashedpass").
		SetPin("1234").
		SetPic("5678").
		SetState(StateLoggedIn).
		SetGender(1).
		SetTOS(true).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.Id != 123 {
		t.Errorf("Id mismatch. Expected 123, got %v", rm.Id)
	}

	if rm.Name != "testuser" {
		t.Errorf("Name mismatch. Expected testuser, got %v", rm.Name)
	}

	if rm.Password != "hashedpass" {
		t.Errorf("Password mismatch. Expected hashedpass, got %v", rm.Password)
	}

	if rm.Pin != "1234" {
		t.Errorf("Pin mismatch. Expected 1234, got %v", rm.Pin)
	}

	if rm.Pic != "5678" {
		t.Errorf("Pic mismatch. Expected 5678, got %v", rm.Pic)
	}

	if rm.LoggedIn != byte(StateLoggedIn) {
		t.Errorf("LoggedIn mismatch. Expected %v, got %v", StateLoggedIn, rm.LoggedIn)
	}

	if rm.Gender != 1 {
		t.Errorf("Gender mismatch. Expected 1, got %v", rm.Gender)
	}

	if rm.TOS != true {
		t.Errorf("TOS mismatch. Expected true, got %v", rm.TOS)
	}
}

func TestExtract(t *testing.T) {
	rm := RestModel{
		Id:       456,
		Name:     "extracteduser",
		Password: "extractpass",
		Pin:      "9999",
		Pic:      "8888",
		LoggedIn: byte(StateTransition),
		Gender:   0,
		TOS:      false,
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if m.Id() != 456 {
		t.Errorf("Id mismatch. Expected 456, got %v", m.Id())
	}

	if m.Name() != "extracteduser" {
		t.Errorf("Name mismatch. Expected extracteduser, got %v", m.Name())
	}

	if m.Password() != "extractpass" {
		t.Errorf("Password mismatch. Expected extractpass, got %v", m.Password())
	}

	if m.Pin() != "9999" {
		t.Errorf("Pin mismatch. Expected 9999, got %v", m.Pin())
	}

	if m.Pic() != "8888" {
		t.Errorf("Pic mismatch. Expected 8888, got %v", m.Pic())
	}

	if m.State() != StateTransition {
		t.Errorf("State mismatch. Expected %v, got %v", StateTransition, m.State())
	}

	if m.Gender() != 0 {
		t.Errorf("Gender mismatch. Expected 0, got %v", m.Gender())
	}

	if m.TOS() != false {
		t.Errorf("TOS mismatch. Expected false, got %v", m.TOS())
	}
}

func TestRestModelGetName(t *testing.T) {
	rm := RestModel{}
	if rm.GetName() != "accounts" {
		t.Errorf("GetName mismatch. Expected 'accounts', got '%v'", rm.GetName())
	}
}

func TestRestModelGetID(t *testing.T) {
	rm := RestModel{Id: 789}
	if rm.GetID() != "789" {
		t.Errorf("GetID mismatch. Expected '789', got '%v'", rm.GetID())
	}
}

func TestRestModelSetID(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("321")
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}

	if rm.Id != 321 {
		t.Errorf("Id mismatch after SetID. Expected 321, got %v", rm.Id)
	}
}

func TestRestModelSetIDInvalid(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("notanumber")
	if err == nil {
		t.Fatal("Expected error for invalid id, got nil")
	}
}

func TestCreateRestModelGetName(t *testing.T) {
	crm := CreateRestModel{}
	if crm.GetName() != "accounts" {
		t.Errorf("GetName mismatch. Expected 'accounts', got '%v'", crm.GetName())
	}
}

// createAccountBody is the JSON:API envelope the UI POSTs to create an
// account: no resource id (the id is assigned server-side), name + password
// in attributes.
var createAccountBody = []byte(`{"data":{"type":"accounts","attributes":{"name":"newuser","password":"secret","gender":0}}}`)

// TestCreateRestModelUnmarshalNoID pins the account-create fix: the create
// route deserializes into CreateRestModel, whose SetID ignores the (empty)
// create id and whose Password field is json-tagged, so the password is
// captured. Regression for the 400 the UI got on account creation.
func TestCreateRestModelUnmarshalNoID(t *testing.T) {
	var crm CreateRestModel
	if err := jsonapi.Unmarshal(createAccountBody, &crm); err != nil {
		t.Fatalf("unmarshal create body into CreateRestModel: %v", err)
	}
	if crm.Name != "newuser" {
		t.Errorf("Name = %q, want newuser", crm.Name)
	}
	if crm.Password != "secret" {
		t.Errorf("Password = %q, want secret (RestModel drops it via json:\"-\")", crm.Password)
	}
}

// TestRestModelUnmarshalRejectsEmptyID documents why the create route cannot
// use RestModel: its SetID does strconv.ParseUint on the create id, which is
// empty, so deserialization fails with a 400 before the handler ever runs.
func TestRestModelUnmarshalRejectsEmptyID(t *testing.T) {
	var rm RestModel
	if err := jsonapi.Unmarshal(createAccountBody, &rm); err == nil {
		t.Fatalf("expected RestModel unmarshal to fail on empty create id, got nil")
	}
}
