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
	got, moved, err := transition(db, tid, m.Id(), StatusCancelled, "operator_cancelled", now)
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
	_, moved, err = transition(db, tid, m.Id(), StatusCancelled, "operator_cancelled", now.Add(time.Minute))
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

// TestReadsAreTenantScoped seeds the SAME character_id and the SAME pending
// requested_name_lower under two different tenants. Every read must return
// only its own tenant's row, and a transition issued under tenant A must not
// move tenant B's record — this is the property the partial unique indexes
// (which are already correctly tenant-scoped) do NOT protect, because they
// only guard writes. Dropping any `tenant_id = ?` predicate makes this test
// fail.
func TestReadsAreTenantScoped(t *testing.T) {
	db := newTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	tidA := uuid.New()
	tidB := uuid.New()
	const characterId = uint32(99)
	const nameLower = "echo"

	mk := func(id uuid.UUID) Model {
		return NewBuilder().
			SetId(id).SetCharacterId(characterId).SetType(TypeNameChange).
			SetStatus(StatusPending).SetRequestedName("Echo").
			SetSourceWorldId(world.Id(0)).SetTransactionId(uuid.New()).
			SetCreatedAt(time.Now()).SetExpiresAt(time.Now().Add(time.Hour)).Build()
	}

	mA, err := create(db, tidA, mk(uuid.New()))
	if err != nil {
		t.Fatalf("create tenant A: %v", err)
	}
	mB, err := create(db, tidB, mk(uuid.New()))
	if err != nil {
		t.Fatalf("create tenant B: %v", err)
	}

	// getById must not leak tenant B's row through tenant A's id, or vice versa.
	if _, err := getById(db, tidA, mB.Id()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound reading tenant B's id under tenant A, got %v", err)
	}
	if _, err := getById(db, tidB, mA.Id()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound reading tenant A's id under tenant B, got %v", err)
	}

	// getByCharacterId / getPendingByCharacterId must only see the caller's own row.
	gotA, err := getByCharacterId(db, tidA, characterId)
	if err != nil {
		t.Fatalf("getByCharacterId tenant A: %v", err)
	}
	if len(gotA) != 1 || gotA[0].Id() != mA.Id() {
		t.Fatalf("expected tenant A to see only its own row, got %+v", gotA)
	}
	gotB, err := getPendingByCharacterId(db, tidB, characterId)
	if err != nil {
		t.Fatalf("getPendingByCharacterId tenant B: %v", err)
	}
	if len(gotB) != 1 || gotB[0].Id() != mB.Id() {
		t.Fatalf("expected tenant B to see only its own row, got %+v", gotB)
	}

	// getPendingByNameLower: the FR-3.3 reservation lookup must not let tenant A's
	// query resolve against tenant B's reservation.
	nameA, err := getPendingByNameLower(db, tidA, nameLower)
	if err != nil {
		t.Fatalf("getPendingByNameLower tenant A: %v", err)
	}
	if nameA.Id() != mA.Id() {
		t.Fatalf("expected tenant A's own reservation, got %v", nameA.Id())
	}

	// transition issued under tenant A must not move tenant B's record. Because
	// the post-update read is also tenant-scoped, the row is invisible to tenant
	// A altogether, so this surfaces as ErrNotFound rather than moved == false —
	// either way, nothing about tenant B's record may be touched.
	_, moved, err := transition(db, tidA, mB.Id(), StatusCancelled, "operator_cancelled", time.Now())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound transitioning tenant B's id under tenant A, got %v", err)
	}
	if moved {
		t.Fatal("expected transition under tenant A to be a no-op against tenant B's record")
	}
	stillPending, err := getById(db, tidB, mB.Id())
	if err != nil {
		t.Fatalf("getById tenant B after cross-tenant transition attempt: %v", err)
	}
	if stillPending.Status() != StatusPending {
		t.Fatalf("expected tenant B's record to remain PENDING, got %s", stillPending.Status())
	}

	// The correct-tenant transition still works.
	_, moved, err = transition(db, tidB, mB.Id(), StatusCancelled, "operator_cancelled", time.Now())
	if err != nil {
		t.Fatalf("transition under tenant B: %v", err)
	}
	if !moved {
		t.Fatal("expected transition under tenant B to move its own record")
	}

	// getExpired / getResolvedUnnotified must also stay tenant-scoped.
	if moved, err := markNotified(db, tidA, mB.Id(), time.Now()); err != nil {
		t.Fatalf("markNotified under tenant A: %v", err)
	} else if moved {
		t.Fatal("expected tenant A's markNotified call against tenant B's id to be a no-op")
	}
	resolvedB, err := getResolvedUnnotified(db, tidB, characterId)
	if err != nil {
		t.Fatalf("getResolvedUnnotified tenant B: %v", err)
	}
	if len(resolvedB) != 1 || resolvedB[0].NotifiedAt() != nil {
		t.Fatalf("expected tenant A's markNotified call to have no effect on tenant B's record, got %+v", resolvedB)
	}
}
