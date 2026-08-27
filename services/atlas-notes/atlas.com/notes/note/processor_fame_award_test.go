package note

import (
	"atlas-notes/kafka/message"
	"atlas-notes/saga"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// fakeSagaProcessor records every saga submitted via Create so tests can assert WHEN (relative to
// the discard transaction's outcome) fame-award sagas are fired, without hitting a real Kafka
// producer. It is declared in this (internal, package note) test file because the sagaP field on
// ProcessorImpl is unexported and can only be injected from within the package.
type fakeSagaProcessor struct {
	calls []saga.Saga
}

func (f *fakeSagaProcessor) Create(s saga.Saga) error {
	f.calls = append(f.calls, s)
	return nil
}

func fameAwardTestDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	l, _ := test.NewNullLogger()
	database.RegisterTenantCallbacks(l, db)
	if err := Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("Failed to migrate outbox table: %v", err)
	}
	return db
}

func fameAwardTestTenant() tenant.Model {
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return te
}

// newFameAwardTestProcessor builds a ProcessorImpl directly (bypassing NewProcessor) so a
// fakeSagaProcessor can be injected in place of the real, Kafka-producing saga.Processor.
func newFameAwardTestProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, sagaP saga.Processor) *ProcessorImpl {
	return &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		db:    db,
		t:     tenant.MustFromContext(ctx),
		sagaP: sagaP,
	}
}

// TestDiscardAndEmit_FameAwardNotFiredWhenDiscardFails proves the task-114 review fix: the
// fame-award saga command built while processing an earlier note in Discard's loop must NOT be
// fired if a later note in the same call fails and DiscardAndEmit returns an error.
//
// Before the fix, awardFameToSender fired the saga command synchronously, inline, the moment each
// note was processed - including for note 1, before the loop ever reached the failing second note
// id. That is exactly the non-atomic side effect this regression test targets: a command must never
// be sent for a note whose processing did not run to a successful DiscardAndEmit completion.
func TestDiscardAndEmit_FameAwardNotFiredWhenDiscardFails(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := fameAwardTestTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := fameAwardTestDatabase(t)
	fakeSaga := &fakeSagaProcessor{}
	p := newFameAwardTestProcessor(l, ctx, db, fakeSaga)

	characterId := uint32(1)
	senderId := uint32(2)

	mb := message.NewBuffer()
	n1, err := p.Create(mb)(uuid.Nil)(characterId)(senderId)("Note 1")(0)(false)
	if err != nil {
		t.Fatalf("Failed to create note 1: %v", err)
	}

	ch := channel.NewModel(0, 0)

	// The loop processes note 1 first (building its fame-award saga), then hits a note id that does
	// not exist, which makes Discard - and therefore DiscardAndEmit - return an error.
	nonExistentNoteId := n1.Id() + 1000
	err = p.DiscardAndEmit(ch, characterId, []uint32{n1.Id(), nonExistentNoteId})
	if err == nil {
		t.Fatalf("Expected DiscardAndEmit to fail due to the non-existent note id, got nil error")
	}

	// Critically: the fame-award saga built for note 1 must never have been fired, since the overall
	// DiscardAndEmit call did not succeed.
	if len(fakeSaga.calls) != 0 {
		t.Fatalf("Expected 0 fame-award saga commands to be fired when DiscardAndEmit fails, got %d", len(fakeSaga.calls))
	}
}

// TestDiscardAndEmit_FameAwardFiresAfterSuccess proves the happy path still fires exactly one
// fame-award saga command per successfully discarded, non-self, non-system note, and only once
// DiscardAndEmit's transaction has run to completion.
func TestDiscardAndEmit_FameAwardFiresAfterSuccess(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := fameAwardTestTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := fameAwardTestDatabase(t)
	fakeSaga := &fakeSagaProcessor{}
	p := newFameAwardTestProcessor(l, ctx, db, fakeSaga)

	characterId := uint32(1)
	senderId := uint32(2)

	mb := message.NewBuffer()
	n1, err := p.Create(mb)(uuid.Nil)(characterId)(senderId)("Note 1")(0)(false)
	if err != nil {
		t.Fatalf("Failed to create note 1: %v", err)
	}
	n2, err := p.Create(mb)(uuid.Nil)(characterId)(senderId)("Note 2")(0)(false)
	if err != nil {
		t.Fatalf("Failed to create note 2: %v", err)
	}

	ch := channel.NewModel(0, 0)
	err = p.DiscardAndEmit(ch, characterId, []uint32{n1.Id(), n2.Id()})
	if err != nil {
		t.Fatalf("Failed to discard notes: %v", err)
	}

	if len(fakeSaga.calls) != 2 {
		t.Fatalf("Expected 2 fame-award saga commands to be fired after a successful discard, got %d", len(fakeSaga.calls))
	}

	notes, err := p.ByCharacterProvider(characterId, model.Page{Number: 1, Size: 250})()
	if err != nil {
		t.Fatalf("Failed to get notes: %v", err)
	}
	if notes.Total != 0 {
		t.Fatalf("Expected 0 notes remaining, got %d", notes.Total)
	}
}

// TestDiscardAndEmit_GiftNoteSuppressesFameInMixedBatch proves the task-240 follow-up fix: a
// gift-originated note produces NO fame-award saga on discard, while an ordinary note discarded in
// the SAME batch still does. A mixed batch is the case that catches a misplaced `continue` — a bug
// that skips the wrong note, or skips both, would not be caught by a single-note test.
//
// The gift note and the ordinary note use DISTINCT sender ids so the surviving payload's
// CharacterId can only match the ordinary note's sender if the skip is applied to the correct
// note; an inverted skip (e.g. `if !m.GiftNote()`) or a note-identity mixup would surface the gift
// note's sender instead and fail this assertion.
func TestDiscardAndEmit_GiftNoteSuppressesFameInMixedBatch(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := fameAwardTestTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := fameAwardTestDatabase(t)
	fakeSaga := &fakeSagaProcessor{}
	p := newFameAwardTestProcessor(l, ctx, db, fakeSaga)

	characterId := uint32(1)
	giftSenderId := uint32(2)
	ordinarySenderId := uint32(3)

	mb := message.NewBuffer()
	giftNote, err := p.Create(mb)(uuid.Nil)(characterId)(giftSenderId)("thanks for the gift!")(0)(true)
	if err != nil {
		t.Fatalf("Failed to create gift note: %v", err)
	}
	ordinaryNote, err := p.Create(mb)(uuid.Nil)(characterId)(ordinarySenderId)("hi")(0)(false)
	if err != nil {
		t.Fatalf("Failed to create ordinary note: %v", err)
	}

	ch := channel.NewModel(0, 0)
	err = p.DiscardAndEmit(ch, characterId, []uint32{giftNote.Id(), ordinaryNote.Id()})
	if err != nil {
		t.Fatalf("Failed to discard notes: %v", err)
	}

	if len(fakeSaga.calls) != 1 {
		t.Fatalf("Expected exactly 1 fame-award saga command (ordinary note only), got %d", len(fakeSaga.calls))
	}
	p1, ok := fakeSaga.calls[0].Steps[0].Payload.(saga.AwardFamePayload)
	if !ok {
		t.Fatalf("step payload type: %T", fakeSaga.calls[0].Steps[0].Payload)
	}
	if p1.CharacterId != ordinarySenderId {
		t.Errorf("fame award characterId: got %d, want %d (sender of the ordinary note, not %d the gift note's sender)", p1.CharacterId, ordinarySenderId, giftSenderId)
	}
}
