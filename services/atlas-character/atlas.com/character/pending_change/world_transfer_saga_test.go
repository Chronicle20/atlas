package pending_change

// world_transfer_saga_test.go — atlas-character's half of the WorldTransfer
// saga (task-227 Task 14, fix round 1).
//
// The defect these pin: Task 6 created the WithWorldTransferStarter seam and
// deferred the contract; Tasks 12-14 built the entire orchestrator side; and
// nothing ever connected them. NewProcessor left worldTransferStarter nil, so
// startWorldTransfer returned "no world-transfer saga dispatcher wired" for
// EVERY WORLD_TRANSFER apply. The feature was dead end to end while every test
// in the suite was green, because the only test that exercised the path
// injected its own starter.

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// TestProductionProcessorHasAWorldTransferStarter is the regression guard. It
// asserts against the PRODUCTION constructor — the exact call
// kafka/consumer/character/consumer.go:339 makes on the LOGOUT apply path,
// which is the only path a world transfer ever takes. On the pre-fix code this
// field is nil and this test fails.
func TestProductionProcessorHasAWorldTransferStarter(t *testing.T) {
	db := newProcessorTestDB(t)
	p := NewProcessor(testLogger(t), testContext(t), db)

	impl, ok := p.(*ProcessorImpl)
	if !ok {
		t.Fatalf("NewProcessor returned %T, want *ProcessorImpl", p)
	}
	if impl.worldTransferStarter == nil {
		t.Fatal("the production processor has no world-transfer saga dispatcher wired; every WORLD_TRANSFER apply will fail with \"no world-transfer saga dispatcher wired\"")
	}
}

// TestWithTransactionPreservesTheWorldTransferStarter: ApplyForCharacter runs
// inside a transaction, so a WithTransaction copy that dropped the starter
// would reinstate the exact bug one layer down.
func TestWithTransactionPreservesTheWorldTransferStarter(t *testing.T) {
	db := newProcessorTestDB(t)
	p := NewProcessor(testLogger(t), testContext(t), db).WithTransaction(db)

	if p.(*ProcessorImpl).worldTransferStarter == nil {
		t.Fatal("WithTransaction dropped the world-transfer starter")
	}
}

// TestApplyForCharacterOnAProductionProcessorReachesTheStarter proves the wiring
// behaviourally rather than by field inspection: a production-constructed
// processor applying a WORLD_TRANSFER must NOT fail with the unwired error. It
// does fail here — the severance-snapshot lookups have no service to talk to in
// a unit test — but the distinction between "no dispatcher" and "the dispatcher
// ran and could not reach atlas-guilds" is the entire finding.
func TestApplyForCharacterOnAProductionProcessorReachesTheStarter(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Sierra", world.Id(0))

	p := NewProcessor(l, ctx, db).withTransferEligibilityGates(passingGateDeps())
	if _, err := p.CreateAndEmit(uuid.New(), characterId, TypeWorldTransfer, "", world.Id(1), nil); err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	err := p.ApplyForCharacter(characterId)
	if err != nil && strings.Contains(err.Error(), "no world-transfer saga dispatcher wired") {
		t.Fatalf("the production apply path still has no dispatcher wired: %v", err)
	}
}

func worldTransferSagaFixture(t *testing.T) Model {
	t.Helper()
	return NewBuilder().
		SetId(uuid.MustParse("dddddddd-0000-0000-0000-00000000007b")).
		SetCharacterId(123).
		SetType(TypeWorldTransfer).
		SetStatus(StatusPending).
		SetSourceWorldId(world.Id(0)).
		SetDestinationWorldId(world.Id(2)).
		SetTransactionId(uuid.New()).
		SetCreatedAt(time.Now()).
		SetExpiresAt(time.Now().Add(time.Hour)).
		Build()
}

// TestWorldTransferSagaHasTheFiveStepsInTheFixedOrder pins design §3.11's step
// order. change_character_world MUST be last: it is the single-row update, and
// FR-4.8's "never in two worlds, never in none" rests entirely on every
// severance preceding it.
func TestWorldTransferSagaHasTheFiveStepsInTheFixedOrder(t *testing.T) {
	m := worldTransferSagaFixture(t)

	msgs, err := worldTransferCommandProvider(m, 5, 3, 9, []uint32{7, 8})()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 command, got %d", len(msgs))
	}

	var s sharedsaga.Saga
	if err := json.Unmarshal(msgs[0].Value, &s); err != nil {
		t.Fatalf("unmarshal saga: %v", err)
	}

	if s.SagaType != sharedsaga.WorldTransfer {
		t.Fatalf("sagaType = %s, want %s", s.SagaType, sharedsaga.WorldTransfer)
	}
	want := []struct {
		id     string
		action sharedsaga.Action
	}{
		{stepValidateWorldTransfer, sharedsaga.ValidateWorldTransfer},
		{stepLeaveGuildForTransfer, sharedsaga.LeaveGuildForTransfer},
		{stepLeavePartyForTransfer, sharedsaga.LeavePartyForTransfer},
		{stepSeverBuddiesForTransfer, sharedsaga.SeverBuddiesForTransfer},
		{stepChangeCharacterWorld, sharedsaga.ChangeCharacterWorld},
	}
	if len(s.Steps) != len(want) {
		t.Fatalf("step count = %d, want %d", len(s.Steps), len(want))
	}
	for i, w := range want {
		if s.Steps[i].StepId != w.id {
			t.Fatalf("step %d id = %s, want %s", i, s.Steps[i].StepId, w.id)
		}
		if s.Steps[i].Action != w.action {
			t.Fatalf("step %d action = %s, want %s", i, s.Steps[i].Action, w.action)
		}
		if s.Steps[i].Status != sharedsaga.Pending {
			t.Fatalf("step %d status = %s, want pending", i, s.Steps[i].Status)
		}
	}
}

// Every payload must arrive at the orchestrator as its CONCRETE type with the
// snapshot values populated. Title and BuddyIds are the load-bearing ones: they
// exist solely so the compensation can be exact, and there is no second source
// for either once the severance has run.
func TestWorldTransferSagaPayloadsCarryTheSeveranceSnapshot(t *testing.T) {
	m := worldTransferSagaFixture(t)

	msgs, err := worldTransferCommandProvider(m, 5, 3, 9, []uint32{7, 8})()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	var s sharedsaga.Saga
	if err := json.Unmarshal(msgs[0].Value, &s); err != nil {
		t.Fatalf("unmarshal saga: %v", err)
	}

	vp, ok := s.Steps[0].Payload.(sharedsaga.ValidateWorldTransferPayload)
	if !ok {
		t.Fatalf("step 0 payload = %T, want ValidateWorldTransferPayload", s.Steps[0].Payload)
	}
	if vp.CharacterId != 123 || vp.SourceWorldId != world.Id(0) || vp.DestinationWorldId != world.Id(2) || vp.PendingChangeId != m.Id() {
		t.Fatalf("validate payload = %+v", vp)
	}

	gp, ok := s.Steps[1].Payload.(sharedsaga.LeaveGuildForTransferPayload)
	if !ok {
		t.Fatalf("step 1 payload = %T, want LeaveGuildForTransferPayload", s.Steps[1].Payload)
	}
	if gp.GuildId != 5 || gp.Title != 3 {
		t.Fatalf("guild payload = %+v, want guildId 5 title 3", gp)
	}
	if gp.WorldId != world.Id(0) {
		t.Fatalf("guild payload worldId = %d, want the SOURCE world", gp.WorldId)
	}

	pp, ok := s.Steps[2].Payload.(sharedsaga.LeavePartyForTransferPayload)
	if !ok {
		t.Fatalf("step 2 payload = %T, want LeavePartyForTransferPayload", s.Steps[2].Payload)
	}
	if pp.PartyId != 9 {
		t.Fatalf("party payload = %+v, want partyId 9", pp)
	}

	bp, ok := s.Steps[3].Payload.(sharedsaga.SeverBuddiesForTransferPayload)
	if !ok {
		t.Fatalf("step 3 payload = %T, want SeverBuddiesForTransferPayload", s.Steps[3].Payload)
	}
	if len(bp.BuddyIds) != 2 || bp.BuddyIds[0] != 7 || bp.BuddyIds[1] != 8 {
		t.Fatalf("buddy payload = %+v, want [7 8]", bp)
	}

	cp, ok := s.Steps[4].Payload.(sharedsaga.ChangeCharacterWorldPayload)
	if !ok {
		t.Fatalf("step 4 payload = %T, want ChangeCharacterWorldPayload", s.Steps[4].Payload)
	}
	// SourceWorldId is what the compensation restores to; getting it from the
	// record rather than re-reading it is design §4's whole point.
	if cp.SourceWorldId != world.Id(0) || cp.DestinationWorldId != world.Id(2) {
		t.Fatalf("change payload = %+v", cp)
	}
	if cp.PendingChangeId != m.Id() {
		t.Fatalf("change payload pendingChangeId = %s, want %s", cp.PendingChangeId, m.Id())
	}
}

// A guildless / partyless / buddyless character produces zero values, which the
// orchestrator's handlers read as legitimate skip signals. This is the shape a
// failed LOOKUP must never be allowed to imitate — hence
// productionWorldTransferStarter propagating every lookup error instead.
func TestWorldTransferSagaZeroesAreLegitimateSkipSignals(t *testing.T) {
	m := worldTransferSagaFixture(t)

	msgs, err := worldTransferCommandProvider(m, 0, 0, 0, nil)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	var s sharedsaga.Saga
	if err := json.Unmarshal(msgs[0].Value, &s); err != nil {
		t.Fatalf("unmarshal saga: %v", err)
	}
	if len(s.Steps) != 5 {
		t.Fatalf("step count = %d, want 5 even when every membership is empty", len(s.Steps))
	}
	if gp := s.Steps[1].Payload.(sharedsaga.LeaveGuildForTransferPayload); gp.GuildId != 0 {
		t.Fatalf("guildId = %d, want 0", gp.GuildId)
	}
	if bp := s.Steps[3].Payload.(sharedsaga.SeverBuddiesForTransferPayload); len(bp.BuddyIds) != 0 {
		t.Fatalf("buddyIds = %v, want empty", bp.BuddyIds)
	}
}

// The consumption, refund and transfer sagas must not share a transaction id:
// atlas-saga-orchestrator keys stored sagas by transaction id, so a collision
// would make one overwrite another.
func TestWorldTransferSagaTransactionIdIsDistinctFromTheCouponSagas(t *testing.T) {
	m := worldTransferSagaFixture(t)

	transfer := sagaTransactionId(m, sagaPurposeWorldTransfer)
	destroy := sagaTransactionId(m, sagaPurposeDestroyAsset)
	award := sagaTransactionId(m, sagaPurposeAwardAsset)

	if transfer == destroy || transfer == award || transfer == m.Id() {
		t.Fatalf("transfer saga transaction id collides: transfer=%s destroy=%s award=%s record=%s", transfer, destroy, award, m.Id())
	}
	// Derived, not minted: reproducible from the record alone.
	if transfer != sagaTransactionId(m, sagaPurposeWorldTransfer) {
		t.Fatal("transfer saga transaction id is not stable across calls")
	}
}
