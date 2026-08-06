package templates

import (
	"atlas-configurations/data"
	datamock "atlas-configurations/data/mock"
	configsocket "atlas-configurations/socket"
	"atlas-configurations/templates/characters"
	"atlas-configurations/templates/characters/preset"
	"atlas-configurations/templates/socket"
	"atlas-configurations/templates/socket/handler"
	"atlas-configurations/templates/socket/writer"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// assertSharedInput is a compile-time proof that ToValidationInput returns
// the shared validator package's configsocket.Input, not a tree-local
// look-alike. staticcheck's QF1011 flags `var _ configsocket.Input = in` as
// a redundant type on the variable declaration — true today, but that
// redundancy IS the assertion: it stops compiling the moment the adapter's
// return type drifts from configsocket.Input. A function-parameter check
// carries the same guarantee without tripping QF1011, since QF1011 only
// applies to variable declarations, not call expressions.
func assertSharedInput(configsocket.Input) {}

// testDB opens a fresh in-memory SQLite database and migrates it with
// testEntity (defined in processor_test.go), NOT the real Entity: Entity's
// `gorm:"...default:uuid_generate_v4()"` tag is a Postgres-only function that
// SQLite's AutoMigrate cannot translate ("near \"(\": syntax error"). Create
// already generates the id in Go for database portability, so the DB-side
// default is never relied upon; testEntity mirrors Entity's columns without
// the Postgres-specific default.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&testEntity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func validTemplate() RestModel {
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

	id, err := p.Create(validTemplate())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	bad := validTemplate()
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

	bad := validTemplate()
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

	id, err := p.Create(validTemplate())
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
	assertSharedInput(in)
}

// TestUpdateById_MergesSocketAndPresetIssues proves socket and preset
// validation failures no longer mask each other: a document invalid in BOTH
// dimensions must come back as a single validationFailureError carrying BOTH
// families, and AsJSONAPIErrors() must render both.
func TestUpdateById_MergesSocketAndPresetIssues(t *testing.T) {
	db := testDB(t)
	l := logrus.New()
	ctx := context.Background()

	base := validTemplate()
	id, err := NewProcessor(l, ctx, db).Create(base)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := &datamock.ProcessorMock{Items: map[uint32]data.ItemInfo{}, Skills: map[uint32]data.SkillInfo{}}
	p := NewProcessor(l, ctx, db).WithValidator(preset.NewValidator(fake))

	bad := validTemplate()
	// Socket-invalid: handler carries no validator.
	bad.Socket.Handlers[0].Validator = ""
	// Preset-invalid: R-1 (empty name) and R-3 (unknown jobId).
	bad.Characters = characters.RestModel{
		Presets: []preset.RestModel{{
			Attributes: preset.Attributes{
				Name:  "",
				JobId: 99999,
				Level: 1,
			},
		}},
	}

	err = p.UpdateById(id, bad)
	var ve *validationFailureError
	if !errors.As(err, &ve) {
		t.Fatalf("UpdateById err = %v, want *validationFailureError", err)
	}

	apiErrs := ve.AsJSONAPIErrors()

	var sawSocketPath, sawPresetName, sawPresetJobId bool
	for _, e := range apiErrs {
		if e.Status != "400" {
			t.Errorf("status = %q, want 400 for %+v", e.Status, e)
		}
		path, _ := e.Meta["path"].(string)
		switch {
		case path == "socket.handlers[0].validator":
			sawSocketPath = true
		case strings.HasPrefix(path, "presets[") && strings.HasSuffix(path, "].name"):
			sawPresetName = true
		case strings.HasPrefix(path, "presets[") && strings.HasSuffix(path, "].jobId"):
			sawPresetJobId = true
		}
	}

	if !sawSocketPath || !sawPresetName || !sawPresetJobId {
		t.Fatalf("socket and preset issues did not both reach the merged 400 body: socket=%v presetName=%v presetJobId=%v, errs=%+v",
			sawSocketPath, sawPresetName, sawPresetJobId, apiErrs)
	}
}
