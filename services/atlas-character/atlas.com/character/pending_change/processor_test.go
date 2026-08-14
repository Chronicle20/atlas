package pending_change

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"

	pendingchange2 "atlas-character/kafka/message/pending_change"
)

// outboxValuesMatching returns the serialized bodies of every outbox row
// containing substr, newest last, so a test can assert on the payload itself
// and not merely on how many rows exist.
func outboxValuesMatching(t *testing.T, db *gorm.DB, substr string) [][]byte {
	t.Helper()
	var es []outbox.Entity
	err := db.Model(&outbox.Entity{}).
		Where("encode(message_value, 'escape') LIKE ?", "%"+substr+"%").
		Order("id ASC").
		Find(&es).Error
	if err != nil {
		t.Fatalf("read outbox rows matching %q: %v", substr, err)
	}
	out := make([][]byte, 0, len(es))
	for _, e := range es {
		out = append(out, e.MessageValue)
	}
	return out
}

func TestCreateReservesTheNameAndConsumesTheCoupon(t *testing.T) {
	db := newProcessorTestDB(t)
	characterId := seedCharacter(t, db, "Alfa", world.Id(0))

	assetId := uint32(5400000)
	p := NewProcessor(testLogger(t), testContext(t), db)
	m, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Bravo", world.Id(0), &assetId)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	if m.Status() != StatusPending {
		t.Fatalf("status = %s, want PENDING", m.Status())
	}
	if m.SourceWorldId() != world.Id(0) {
		t.Fatalf("sourceWorldId = %d, want 0", m.SourceWorldId())
	}
	// FR-2.8 / design §5.5: the default lifetime is 7 days until Task 8's
	// tenant configuration replaces it.
	if d := m.ExpiresAt().Sub(m.CreatedAt()); d != DefaultExpiry {
		t.Fatalf("lifetime = %s, want %s", d, DefaultExpiry)
	}

	// The reservation is live and blocks the name for everyone (FR-3.3).
	reserved, err := p.NameReserved("bRaVo")
	if err != nil {
		t.Fatalf("NameReserved: %v", err)
	}
	if !reserved {
		t.Fatal("expected the requested name to be reserved case-insensitively")
	}
	if reserved, err = p.NameReserved("Charlie"); err != nil || reserved {
		t.Fatalf("expected an unrelated name to be free, got reserved=%v err=%v", reserved, err)
	}

	if got := countOutboxMessagesMatching(t, db, "PENDING_CHANGE_CREATED"); got != 1 {
		t.Fatalf("expected 1 created event, got %d", got)
	}
	if got := countOutboxMessagesMatching(t, db, "destroy_asset"); got != 1 {
		t.Fatalf("expected 1 consumption command, got %d", got)
	}
}

// NameReservedFor is the adapter that closes the FR-3.3 loop without character
// importing pending_change.
func TestNameReservedForBlocksCharacterNameValidity(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Delta", world.Id(0))

	assetId := uint32(5400000)
	if _, err := NewProcessor(l, ctx, db).
		CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Echo", world.Id(0), &assetId); err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	res, err := character.NewProcessor(l, ctx, db).
		WithNameReserved(NameReservedFor(db)).
		CheckNameValidity("Echo", world.Id(0), character.NameScopeTenant)
	if err != nil {
		t.Fatalf("CheckNameValidity: %v", err)
	}
	if res.Valid || res.Reason != "reserved" {
		t.Fatalf("got valid=%v reason=%s, want reserved", res.Valid, res.Reason)
	}
}

// A name that is already a real character's is rejected at request time with a
// reason from the design §6 taxonomy, and nothing is persisted or emitted.
func TestCreateRejectsATakenNameWithoutSideEffects(t *testing.T) {
	db := newProcessorTestDB(t)
	characterId := seedCharacter(t, db, "Foxtrot", world.Id(0))
	seedCharacter(t, db, "Hotel", world.Id(1))

	p := NewProcessor(testLogger(t), testContext(t), db)
	_, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Hotel", world.Id(0), nil)
	var ie IneligibleError
	if !errors.As(err, &ie) || ie.Reason != "name_taken" {
		t.Fatalf("err = %v, want IneligibleError{name_taken}", err)
	}
	if !errors.Is(err, ErrIneligible) {
		t.Fatalf("expected the rejection to wrap ErrIneligible, got %v", err)
	}
	if got, err := p.GetByCharacterId(characterId); err != nil || len(got) != 0 {
		t.Fatalf("expected no persisted record, got %d (err %v)", len(got), err)
	}
	if got := countOutboxMessagesMatching(t, db, "PENDING_CHANGE_CREATED"); got != 0 {
		t.Fatalf("expected no emission on a rejected request, got %d", got)
	}
}

func TestCreateRejectsATransferToTheSameWorld(t *testing.T) {
	db := newProcessorTestDB(t)
	characterId := seedCharacter(t, db, "Kilo", world.Id(3))

	_, err := NewProcessor(testLogger(t), testContext(t), db).
		CreateAndEmit(uuid.New(), characterId, TypeWorldTransfer, "", world.Id(3), nil)
	var ie IneligibleError
	if !errors.As(err, &ie) || ie.Reason != "world_same" {
		t.Fatalf("err = %v, want IneligibleError{world_same}", err)
	}
}

// FR-2.4 / design §5.1 step 6: the applier is the only writer of the new name,
// and the character update reuses the existing PATCH path, which already emits
// NAME_CHANGED.
func TestApplyForCharacterAppliesAPendingNameChange(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Lima", world.Id(0))

	p := NewProcessor(l, ctx, db)
	m, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Mike", world.Id(0), nil)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	if err := p.ApplyForCharacter(characterId); err != nil {
		t.Fatalf("ApplyForCharacter: %v", err)
	}

	c, err := character.NewProcessor(l, ctx, db).GetById()(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if c.Name() != "Mike" {
		t.Fatalf("name = %s, want Mike", c.Name())
	}
	got, err := p.GetById(m.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Status() != StatusApplied {
		t.Fatalf("status = %s, want APPLIED", got.Status())
	}
	if got.ResolvedAt() == nil {
		t.Fatal("expected resolved_at to be stamped")
	}
	if n := countOutboxMessagesMatching(t, db, "NAME_CHANGED"); n != 1 {
		t.Fatalf("expected exactly one NAME_CHANGED emission, got %d", n)
	}
	// APPLIED releases the reservation.
	if reserved, err := p.NameReserved("Mike"); err != nil || reserved {
		t.Fatalf("expected the reservation to be released, got reserved=%v err=%v", reserved, err)
	}
}

// Design §5.2: the name was taken between reservation and apply. The record is
// REJECTED with the taxonomy reason and the coupon is refunded once.
func TestApplyForCharacterRejectsAndRefundsWhenTheNameWasTaken(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "November", world.Id(0))

	assetId := uint32(5400000)
	p := NewProcessor(l, ctx, db)
	m, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Oscar", world.Id(0), &assetId)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	// Another character takes the name in the interim.
	seedCharacter(t, db, "Oscar", world.Id(1))

	if err := p.ApplyForCharacter(characterId); err != nil {
		t.Fatalf("ApplyForCharacter: %v", err)
	}

	got, err := p.GetById(m.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Status() != StatusRejected || got.Reason() != "name_taken" {
		t.Fatalf("got %s / %s, want REJECTED / name_taken", got.Status(), got.Reason())
	}
	if n := countOutboxMessagesMatching(t, db, "award_asset"); n != 1 {
		t.Fatalf("expected the coupon to be refunded exactly once, got %d", n)
	}
	if n := countOutboxMessagesMatching(t, db, "NAME_CHANGED"); n != 0 {
		t.Fatalf("expected no name change on the rejected path, got %d", n)
	}
}

// A world transfer stays PENDING at apply time — the saga's terminal event is
// what drives Resolve — and its dispatcher is an injected seam, so an unwired
// processor fails loudly rather than silently dropping the request.
func TestApplyForCharacterDispatchesTheWorldTransferAndLeavesItPending(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Papa", world.Id(0))

	base := NewProcessor(l, ctx, db).withTransferEligibilityGates(passingGateDeps())
	m, err := base.CreateAndEmit(uuid.New(), characterId, TypeWorldTransfer, "", world.Id(1), nil)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	if err := base.ApplyForCharacter(characterId); err == nil {
		t.Fatal("expected an unwired world-transfer dispatcher to fail loudly")
	}

	var started []uuid.UUID
	wired := base.WithWorldTransferStarter(func(_ logrus.FieldLogger, _ context.Context, _ *message.Buffer, pc Model) error {
		started = append(started, pc.Id())
		return nil
	})
	if err := wired.ApplyForCharacter(characterId); err != nil {
		t.Fatalf("ApplyForCharacter: %v", err)
	}
	if len(started) != 1 || started[0] != m.Id() {
		t.Fatalf("expected the transfer saga to be started once for %s, got %v", m.Id(), started)
	}

	got, err := base.GetById(m.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Status() != StatusPending {
		t.Fatalf("status = %s, want PENDING until the saga terminates", got.Status())
	}
}

// The CONTROLLER RULING: the LOGIN catch-up routes on the character's CURRENT
// world, not the record's source world. An APPLIED world transfer has already
// moved the character, so routing on the source announces the success to the
// world the player just left.
func TestRenotifyRoutesOnTheCurrentWorldNotTheSourceWorld(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Quebec", world.Id(0))

	p := NewProcessor(l, ctx, db).withTransferEligibilityGates(passingGateDeps())
	m, err := p.CreateAndEmit(uuid.New(), characterId, TypeWorldTransfer, "", world.Id(4), nil)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	if _, moved, err := p.ResolveAndEmit(m.Id(), StatusApplied, ""); err != nil || !moved {
		t.Fatalf("ResolveAndEmit: moved=%v err=%v", moved, err)
	}

	// The saga has moved the character to the destination world by the time the
	// player logs back in.
	if err := db.Exec("UPDATE characters SET world = ? WHERE id = ?", world.Id(4), characterId).Error; err != nil {
		t.Fatalf("move character: %v", err)
	}

	if err := p.RenotifyForCharacter(characterId); err != nil {
		t.Fatalf("RenotifyForCharacter: %v", err)
	}

	vs := outboxValuesMatching(t, db, "PENDING_CHANGE_RESOLVED")
	if len(vs) != 2 {
		t.Fatalf("expected the original emission plus one re-emission, got %d", len(vs))
	}
	var original, renotified pendingchange2.StatusEvent[pendingchange2.ResolvedEventBody]
	if err := json.Unmarshal(vs[0], &original); err != nil {
		t.Fatalf("decode original: %v", err)
	}
	if err := json.Unmarshal(vs[1], &renotified); err != nil {
		t.Fatalf("decode re-emission: %v", err)
	}
	// At original emission time the character had not moved: source is correct.
	if original.WorldId != world.Id(0) {
		t.Fatalf("original worldId = %d, want the source world 0", original.WorldId)
	}
	if renotified.WorldId != world.Id(4) {
		t.Fatalf("re-emitted worldId = %d, want the character's current world 4", renotified.WorldId)
	}

	// notified_at is stamped, so a second login does not re-announce.
	if err := p.RenotifyForCharacter(characterId); err != nil {
		t.Fatalf("second RenotifyForCharacter: %v", err)
	}
	if got := len(outboxValuesMatching(t, db, "PENDING_CHANGE_RESOLVED")); got != 2 {
		t.Fatalf("expected the catch-up to drain exactly once, got %d emissions", got)
	}
}

// Design §5.5: the sweep expires and refunds, and running it twice refunds once
// — the same transition guard as the operator cancel.
func TestSweepExpiresAndRefundsExactlyOnce(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Romeo", world.Id(0))

	assetId := uint32(5400000)
	p := NewProcessor(l, ctx, db)
	m, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Sierra", world.Id(0), &assetId)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	// Nothing is due yet.
	if err := p.Sweep(time.Now()); err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if got := countOutboxMessagesMatching(t, db, "award_asset"); got != 0 {
		t.Fatalf("expected an unexpired record to survive the sweep, got %d refunds", got)
	}

	after := m.ExpiresAt().Add(time.Minute)
	for i := 0; i < 2; i++ {
		if err := p.Sweep(after); err != nil {
			t.Fatalf("Sweep %d: %v", i, err)
		}
	}

	got, err := p.GetById(m.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Status() != StatusExpired || got.Reason() != "expired" {
		t.Fatalf("got %s / %s, want EXPIRED / expired", got.Status(), got.Reason())
	}
	if n := countOutboxMessagesMatching(t, db, "award_asset"); n != 1 {
		t.Fatalf("expected exactly one refund across two sweeps, got %d", n)
	}
}
