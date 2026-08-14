package pending_change

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestTransitionIsOneWayAndReportsWhetherItMoved(t *testing.T) {
	db := newTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	tid := uuid.New()

	m, err := create(db, tid, NewBuilder().
		SetId(uuid.New()).
		SetCharacterId(11).
		SetType(TypeNameChange).
		SetStatus(StatusPending).
		SetRequestedName("Bravo").
		SetSourceWorldId(world.Id(0)).
		SetTransactionId(uuid.New()).
		SetCreatedAt(time.Now()).
		SetExpiresAt(time.Now().Add(time.Hour)).
		Build())
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	now := time.Now()
	got, moved, err := transition(db, m.Id(), StatusCancelled, "operator_cancelled", now)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if !moved {
		t.Fatal("expected the first transition to move the row")
	}
	if got.Status() != StatusCancelled || got.Reason() != "operator_cancelled" {
		t.Fatalf("unexpected post-state: %s / %s", got.Status(), got.Reason())
	}
	if got.ResolvedAt() == nil {
		t.Fatal("expected resolved_at to be stamped")
	}

	// A redelivered cancel finds a terminal row: nothing moves, nothing is
	// re-stamped, and the caller must not emit a refund.
	_, moved, err = transition(db, m.Id(), StatusCancelled, "operator_cancelled", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("second transition returned an error: %v", err)
	}
	if moved {
		t.Fatal("expected the second transition to be a no-op")
	}
}

func TestCreateMapsUniqueViolationsToSentinels(t *testing.T) {
	db := newTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	tid := uuid.New()
	base := func(charId uint32, name string) Model {
		return NewBuilder().
			SetId(uuid.New()).SetCharacterId(charId).SetType(TypeNameChange).
			SetStatus(StatusPending).SetRequestedName(name).
			SetSourceWorldId(world.Id(0)).SetTransactionId(uuid.New()).
			SetCreatedAt(time.Now()).SetExpiresAt(time.Now().Add(time.Hour)).Build()
	}
	if _, err := create(db, tid, base(21, "Charlie")); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Same character, same type.
	if _, err := create(db, tid, base(21, "Delta")); !errors.Is(err, ErrAlreadyPending) {
		t.Fatalf("expected ErrAlreadyPending, got %v", err)
	}
	// Different character, same name — case-insensitively.
	if _, err := create(db, tid, base(22, "cHaRlIe")); !errors.Is(err, ErrNameReserved) {
		t.Fatalf("expected ErrNameReserved, got %v", err)
	}
}
