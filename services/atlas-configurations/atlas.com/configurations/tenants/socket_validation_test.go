package tenants

import (
	"atlas-configurations/tenants/socket"
	"atlas-configurations/tenants/socket/handler"
	"atlas-configurations/tenants/socket/writer"
	"context"
	"errors"
	"testing"

	configsocket "atlas-configurations/socket"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// testDB reuses setupTestDB (processor_test.go), which migrates testEntity
// rather than the real Entity: Entity's
// `gorm:"...default:uuid_generate_v4()"` tag is a Postgres-only function that
// SQLite's AutoMigrate cannot translate. Create already generates the id in
// Go for database portability, so the DB-side default is never relied upon.
func testDB(t *testing.T) *gorm.DB {
	return setupTestDB(t)
}

func validTenant() RestModel {
	return RestModel{
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
		Socket: socket.RestModel{
			Handlers: []handler.RestModel{
				{OpCode: "0x01", Validator: "NoOpValidator", Handler: "LoginHandle", Services: []string{"login"}},
			},
			Writers: []writer.RestModel{
				{OpCode: "0x00", Writer: "AuthSuccess", Services: []string{"login"}},
			},
		},
	}
}

func TestUpdateById_RejectsConflictingUnsupportedState(t *testing.T) {
	db := testDB(t)
	p := NewProcessor(logrus.New(), context.Background(), db)

	id, err := p.Create(validTenant())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	bad := validTenant()
	bad.Socket.Unsupported.Handlers = []string{"LoginHandle"}

	err = p.UpdateById(id, bad)
	var ve *validationFailureError
	if !errors.As(err, &ve) {
		t.Fatalf("UpdateById err = %v, want *validationFailureError", err)
	}
	jsonErrs := ve.AsJSONAPIErrors()
	if len(jsonErrs) != 1 {
		t.Fatalf("got %d JSON:API errors, want 1: %+v", len(jsonErrs), jsonErrs)
	}
	if jsonErrs[0].Status != "400" {
		t.Errorf("status = %q, want 400", jsonErrs[0].Status)
	}
	if got := jsonErrs[0].Meta["path"]; got != "socket.unsupported.handlers[0]" {
		t.Errorf("meta.path = %v, want socket.unsupported.handlers[0]", got)
	}
}

func TestCreate_RejectsInvalidSocket(t *testing.T) {
	db := testDB(t)
	p := NewProcessor(logrus.New(), context.Background(), db)

	bad := validTenant()
	bad.Socket.Handlers[0].Validator = ""

	_, err := p.Create(bad)
	var ve *validationFailureError
	if !errors.As(err, &ve) {
		t.Fatalf("Create err = %v, want *validationFailureError", err)
	}
	if got := ve.AsJSONAPIErrors()[0].Meta["path"]; got != "socket.handlers[0].validator" {
		t.Errorf("meta.path = %v, want socket.handlers[0].validator", got)
	}
}

func TestCreate_AcceptsValidSocketAndNormalizes(t *testing.T) {
	db := testDB(t)
	p := NewProcessor(logrus.New(), context.Background(), db)

	id, err := p.Create(validTenant())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("Create returned the nil UUID")
	}

	got, err := p.GetById(id)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Socket.Unsupported.Handlers == nil || got.Socket.Unsupported.Writers == nil {
		t.Errorf("stored document did not normalize unsupported: %+v", got.Socket.Unsupported)
	}
}

func TestToValidationInput_FlattensBothCollections(t *testing.T) {
	rm := socket.RestModel{
		Handlers: []handler.RestModel{
			{OpCode: "0x01", Validator: "NoOpValidator", Handler: "LoginHandle", Services: []string{"login"}},
		},
		Writers: []writer.RestModel{
			{OpCode: "0x00", Writer: "AuthSuccess", Services: []string{"login"}},
		},
		Unsupported: socket.UnsupportedRestModel{Handlers: []string{"GuestLoginHandle"}},
	}
	in := socket.ToValidationInput(rm)

	if len(in.Handlers) != 1 || in.Handlers[0].Name != "LoginHandle" || in.Handlers[0].Validator != "NoOpValidator" {
		t.Errorf("handlers not flattened: %+v", in.Handlers)
	}
	if len(in.Writers) != 1 || in.Writers[0].Name != "AuthSuccess" || in.Writers[0].Validator != "" {
		t.Errorf("writers not flattened: %+v", in.Writers)
	}
	if len(in.UnsupportedHandlers) != 1 || in.UnsupportedHandlers[0] != "GuestLoginHandle" {
		t.Errorf("unsupported not carried: %+v", in.UnsupportedHandlers)
	}
	// Compile-time proof the adapter returns the shared package's type.
	var _ configsocket.Input = in
}
