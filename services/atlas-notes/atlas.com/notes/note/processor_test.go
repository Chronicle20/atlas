package note_test

import (
	"atlas-notes/kafka/message"
	"atlas-notes/note"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	var migrators []func(db *gorm.DB) error
	migrators = append(migrators, note.Migration)

	for _, migrator := range migrators {
		if err := migrator(db); err != nil {
			t.Fatalf("Failed to migrate database: %v", err)
		}
	}
	return db
}

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func TestProcessorImpl_CRUD(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	np := note.NewProcessor(l, ctx, db)

	characterId := uint32(1)
	senderId := uint32(2)
	msg := "Hello!"
	flag := byte(0)

	mb := message.NewBuffer()
	nm, err := np.Create(mb)(characterId)(senderId)(msg)(flag)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	if nm.CharacterId() != characterId {
		t.Fatalf("Unexpected characterId")
	}
	if nm.SenderId() != senderId {
		t.Fatalf("Unexpected senderId")
	}
	if nm.Message() != msg {
		t.Fatalf("Unexpected message")
	}
	if nm.Flag() != flag {
		t.Fatalf("Unexpected flag")
	}
}

func TestProcessorImpl_Update(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	np := note.NewProcessor(l, ctx, db)

	// Create note
	characterId := uint32(1)
	senderId := uint32(2)
	mb := message.NewBuffer()

	nm, err := np.Create(mb)(characterId)(senderId)("Original")(0)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Update note
	mb2 := message.NewBuffer()
	updated, err := np.Update(mb2)(nm.Id())(characterId)(senderId)("Updated")(1)
	if err != nil {
		t.Fatalf("Failed to update note: %v", err)
	}

	if updated.Message() != "Updated" {
		t.Fatalf("Expected message 'Updated', got '%s'", updated.Message())
	}
	if updated.Flag() != 1 {
		t.Fatalf("Expected flag 1, got %d", updated.Flag())
	}
}

func TestProcessorImpl_Delete(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	np := note.NewProcessor(l, ctx, db)

	// Create note
	characterId := uint32(1)
	senderId := uint32(2)
	mb := message.NewBuffer()

	nm, err := np.Create(mb)(characterId)(senderId)("To be deleted")(0)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Delete note
	mb2 := message.NewBuffer()
	err = np.Delete(mb2)(nm.Id())
	if err != nil {
		t.Fatalf("Failed to delete note: %v", err)
	}

	// Verify note is deleted
	_, err = np.ByIdProvider(nm.Id())()
	if err == nil {
		t.Fatalf("Expected error when getting deleted note")
	}
}

func TestProcessorImpl_DeleteAll(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	np := note.NewProcessor(l, ctx, db)

	// Create notes for character 1
	characterId := uint32(1)
	senderId := uint32(2)
	mb := message.NewBuffer()

	_, err := np.Create(mb)(characterId)(senderId)("Note 1")(0)
	if err != nil {
		t.Fatalf("Failed to create note 1: %v", err)
	}

	_, err = np.Create(mb)(characterId)(senderId)("Note 2")(0)
	if err != nil {
		t.Fatalf("Failed to create note 2: %v", err)
	}

	// Create note for different character
	otherCharacterId := uint32(99)
	_, err = np.Create(mb)(otherCharacterId)(senderId)("Other note")(0)
	if err != nil {
		t.Fatalf("Failed to create other note: %v", err)
	}

	// Delete all notes for character 1
	mb2 := message.NewBuffer()
	err = np.DeleteAll(mb2)(characterId)
	if err != nil {
		t.Fatalf("Failed to delete all notes: %v", err)
	}

	// Verify notes for character 1 are deleted
	notes, err := np.ByCharacterProvider(characterId)()
	if err != nil {
		t.Fatalf("Failed to get notes: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("Expected 0 notes for character 1, got %d", len(notes))
	}

	// Verify other character's note is still there
	otherNotes, err := np.ByCharacterProvider(otherCharacterId)()
	if err != nil {
		t.Fatalf("Failed to get other notes: %v", err)
	}
	if len(otherNotes) != 1 {
		t.Fatalf("Expected 1 note for other character, got %d", len(otherNotes))
	}
}

func TestProcessorImpl_Discard(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	np := note.NewProcessor(l, ctx, db)

	// Create test notes
	characterId := uint32(1)
	senderId := uint32(2)
	mb := message.NewBuffer()

	n1, err := np.Create(mb)(characterId)(senderId)("Note 1")(0)
	if err != nil {
		t.Fatalf("Failed to create note 1: %v", err)
	}

	n2, err := np.Create(mb)(characterId)(senderId)("Note 2")(0)
	if err != nil {
		t.Fatalf("Failed to create note 2: %v", err)
	}

	n3, err := np.Create(mb)(characterId)(senderId)("Note 3")(0)
	if err != nil {
		t.Fatalf("Failed to create note 3: %v", err)
	}

	// Discard notes 1 and 2
	mb2 := message.NewBuffer()
	err = np.Discard(mb2)(world.Id(0))(channel.Id(0))(characterId)([]uint32{n1.Id(), n2.Id()})
	if err != nil {
		t.Fatalf("Failed to discard notes: %v", err)
	}

	// Verify notes 1 and 2 are deleted
	notes, err := np.ByCharacterProvider(characterId)()
	if err != nil {
		t.Fatalf("Failed to get notes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("Expected 1 note remaining, got %d", len(notes))
	}
	if notes[0].Id() != n3.Id() {
		t.Fatalf("Expected note 3 to remain, got note %d", notes[0].Id())
	}
}

func TestProcessorImpl_Discard_SkipsOtherCharacterNotes(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	np := note.NewProcessor(l, ctx, db)

	// Create notes for different characters
	characterId := uint32(1)
	otherCharacterId := uint32(99)
	senderId := uint32(2)
	mb := message.NewBuffer()

	n1, err := np.Create(mb)(characterId)(senderId)("My note")(0)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	n2, err := np.Create(mb)(otherCharacterId)(senderId)("Other's note")(0)
	if err != nil {
		t.Fatalf("Failed to create other note: %v", err)
	}

	// Try to discard both notes as character 1 (should skip other's note)
	mb2 := message.NewBuffer()
	err = np.Discard(mb2)(world.Id(0))(channel.Id(0))(characterId)([]uint32{n1.Id(), n2.Id()})
	if err != nil {
		t.Fatalf("Failed to discard notes: %v", err)
	}

	// Verify character 1's note is deleted
	notes, err := np.ByCharacterProvider(characterId)()
	if err != nil {
		t.Fatalf("Failed to get notes: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("Expected 0 notes for character 1, got %d", len(notes))
	}

	// Verify other character's note is still there
	otherNotes, err := np.ByCharacterProvider(otherCharacterId)()
	if err != nil {
		t.Fatalf("Failed to get other notes: %v", err)
	}
	if len(otherNotes) != 1 {
		t.Fatalf("Expected 1 note for other character, got %d", len(otherNotes))
	}
}

func TestProcessorImpl_ByIdProvider(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	np := note.NewProcessor(l, ctx, db)

	// Create note
	characterId := uint32(1)
	senderId := uint32(2)
	msg := "Test note"
	mb := message.NewBuffer()

	created, err := np.Create(mb)(characterId)(senderId)(msg)(0)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Get by ID
	found, err := np.ByIdProvider(created.Id())()
	if err != nil {
		t.Fatalf("Failed to get note by ID: %v", err)
	}

	if found.Id() != created.Id() {
		t.Fatalf("Expected ID %d, got %d", created.Id(), found.Id())
	}
	if found.Message() != msg {
		t.Fatalf("Expected message '%s', got '%s'", msg, found.Message())
	}
}

func TestProcessorImpl_ByCharacterProvider(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	np := note.NewProcessor(l, ctx, db)

	// Create notes for character 1
	characterId := uint32(1)
	senderId := uint32(2)
	mb := message.NewBuffer()

	_, err := np.Create(mb)(characterId)(senderId)("Note 1")(0)
	if err != nil {
		t.Fatalf("Failed to create note 1: %v", err)
	}

	_, err = np.Create(mb)(characterId)(senderId)("Note 2")(0)
	if err != nil {
		t.Fatalf("Failed to create note 2: %v", err)
	}

	// Create note for different character
	otherCharacterId := uint32(99)
	_, err = np.Create(mb)(otherCharacterId)(senderId)("Other note")(0)
	if err != nil {
		t.Fatalf("Failed to create other note: %v", err)
	}

	// Get notes for character 1
	notes, err := np.ByCharacterProvider(characterId)()
	if err != nil {
		t.Fatalf("Failed to get notes: %v", err)
	}

	if len(notes) != 2 {
		t.Fatalf("Expected 2 notes, got %d", len(notes))
	}
}

func TestProcessorImpl_InTenantProvider(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	np := note.NewProcessor(l, ctx, db)

	// Create notes for different characters
	mb := message.NewBuffer()

	_, err := np.Create(mb)(1)(2)("Note 1")(0)
	if err != nil {
		t.Fatalf("Failed to create note 1: %v", err)
	}

	_, err = np.Create(mb)(3)(4)("Note 2")(0)
	if err != nil {
		t.Fatalf("Failed to create note 2: %v", err)
	}

	_, err = np.Create(mb)(5)(6)("Note 3")(0)
	if err != nil {
		t.Fatalf("Failed to create note 3: %v", err)
	}

	// Get all notes in tenant
	notes, err := np.InTenantProvider()()
	if err != nil {
		t.Fatalf("Failed to get all notes: %v", err)
	}

	if len(notes) != 3 {
		t.Fatalf("Expected 3 notes, got %d", len(notes))
	}
}
