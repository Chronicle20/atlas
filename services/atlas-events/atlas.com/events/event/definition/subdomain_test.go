package definition

import (
	"atlas-events/event/registry"
	"encoding/json"
	"errors"
	"testing"
)

// B2/FR-D6: BulkCreate is the only production creation path (the seed
// loader's entry point) and must reject a configuration its type's
// registered handler refuses, exactly like Processor.Create does. Before the
// fix this test failed — BulkCreate wrote the row unconditionally without
// ever calling registry.Get(...).ValidateConfiguration.
func TestBulkCreateRejectsConfigurationTheHandlerRefuses(t *testing.T) {
	registryReset(t)
	registry.Register(rejectingHandler{t: "PICKY", err: errors.New("monsterCount must be > 0")})

	db := newTestDB(t)
	m, _ := NewBuilder("PICKY", "n").SetConfiguration(json.RawMessage(`{"monsterCount":0}`)).Build()

	if err := (DefinitionSubdomain{}).BulkCreate(db.WithContext(testCtx(t)), []Model{m}); err == nil {
		t.Fatalf("expected BulkCreate to reject a configuration the handler refuses")
	}

	var count int64
	db.Model(&Entity{}).Count(&count)
	if count != 0 {
		t.Fatalf("invalid definition was persisted anyway")
	}
}

// A type with no registered handler must be rejected at BulkCreate time too.
func TestBulkCreateRejectsUnknownType(t *testing.T) {
	registryReset(t)
	db := newTestDB(t)
	m, _ := NewBuilder("NO_HANDLER", "n").SetConfiguration(json.RawMessage(`{}`)).Build()

	if err := (DefinitionSubdomain{}).BulkCreate(db.WithContext(testCtx(t)), []Model{m}); err == nil {
		t.Fatalf("expected an error for a type with no handler")
	}
}
