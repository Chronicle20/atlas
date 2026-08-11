package trade

import (
	"atlas-trades/escrow"
	"atlas-trades/ledger"
	"atlas-trades/settlement"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// --- the boot harness -----------------------------------------------------------
//
// Startup reconciliation is NOT reached through a ProcessorImpl a test can hand
// fakes to: ReconcileAtBoot runs before any request has supplied a tenant, and
// builds its own processor per row via NewProcessor. Its seams are therefore the
// ones production has — a real database, and the two REST roots
// requests.RootUrl resolves from the environment. Both are supplied here rather
// than stubbed, which is also what makes the "captured before the settlement
// pass" test reachable at all: only a real orchestrator answer drives Reconcile
// to delete a record mid-run.

// reconcileLogger is quiet by default. Reconciliation logs a Warn per stranded
// room by design, and a boot sweep in a test suite would otherwise print one per
// case.
func reconcileLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

// reconcileDb is the boot database: the four tables the two passes read and
// write between them.
func reconcileDb(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, outbox.Migration, ledger.Migration, settlement.Migration, escrow.Migration)
}

// reconcileTenant derives a tenant from the test name and a discriminator, so a
// multi-tenant case gets two genuinely different tenant ids without either
// colliding with another test's.
func reconcileTenant(t *testing.T, name string) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.NewSHA1(uuid.NameSpaceOID, []byte(t.Name()+"/"+name)), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

// seedEscrowItem writes one custody row exactly as the staging path's Accept
// would, and returns its id. The snapshot carries a template and a quantity so
// an unwind payload built from it is distinguishable from a zero value.
func seedEscrowItem(t *testing.T, db *gorm.DB, tm tenant.Model, roomId uuid.UUID, ownerId character.Id, tradeSlot byte) uuid.UUID {
	t.Helper()
	id := uuid.New()
	m := escrow.NewItemBuilder(id, roomId, ownerId).
		SetTradeSlot(tradeSlot).
		SetSource(inventory.TypeValueUse, asset.Id(9000+uint32(tradeSlot))).
		SetSnapshot(sharedsaga.AssetSnapshot{Slot: int16(tradeSlot), TemplateId: 2000000, Quantity: 5}).
		Build()
	if err := escrow.CreateItem(db, tm)(m); err != nil {
		t.Fatalf("seed escrow item: %v", err)
	}
	return id
}

// seedEscrowMeso writes one committed escrow meso row.
func seedEscrowMeso(t *testing.T, db *gorm.DB, tm tenant.Model, roomId uuid.UUID, ownerId character.Id, amount uint32) {
	t.Helper()
	if err := escrow.UpsertMeso(db, tm)(roomId, ownerId, amount); err != nil {
		t.Fatalf("seed escrow meso: %v", err)
	}
}

// seedSettlementRecord writes the durable record a submitted settlement leaves
// behind, naming roomId. Two sides, because the administrator rejects anything
// else.
func seedSettlementRecord(t *testing.T, db *gorm.DB, tm tenant.Model, roomId uuid.UUID) uuid.UUID {
	t.Helper()
	transactionId := uuid.New()
	m := settlement.NewBuilder(transactionId, roomId, 100, 3, testField(t), 100, 200).
		AddSide(0, 100, "Owner", 0, 0, 0, nil).
		AddSide(1, 200, "Guest", 0, 0, 0, nil).
		Build()
	ctx := tenant.WithContext(context.Background(), tm)
	if _, err := settlement.NewProcessor(reconcileLogger(), ctx, db).Submit(m); err != nil {
		t.Fatalf("seed settlement record: %v", err)
	}
	return transactionId
}

// serveLocations stands up atlas-maps' character-location endpoint and points
// the REST client at it. Characters not in `fields` answer 404, which is the
// un-locatable owner case — a 404 is not retried, so it is also the fast one.
func serveLocations(t *testing.T, fields map[character.Id][2]byte) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 3 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var id character.Id
		if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		f, ok := fields[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"type": "character-locations",
				"id":   fmt.Sprintf("%d", uint32(id)),
				"attributes": map[string]interface{}{
					"worldId":   f[0],
					"channelId": f[1],
					"mapId":     100000000,
					"instance":  uuid.Nil.String(),
				},
			},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")
}

// serveSagaOutcome stands up atlas-saga-orchestrator's GET /sagas/{id} and
// points the REST client at it. `completed` decides the step status, and
// therefore whether Reconcile resolves the record or leaves it alone.
func serveSagaOutcome(t *testing.T, completed bool) {
	t.Helper()
	status := "pending"
	if completed {
		status = "completed"
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		id := parts[len(parts)-1]
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"type": "sagas",
				"id":   id,
				"attributes": map[string]interface{}{
					"sagaType":    "trade_transaction",
					"initiatedBy": "atlas-trades",
					"steps": []map[string]interface{}{
						{"stepId": "release_from_trade", "status": status, "action": "release_from_trade"},
					},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("SAGAS_SERVICE_URL", srv.URL+"/")
}

// unwindOf decodes every trade_unwind composite the sweep published, paired with
// the TENANT_ID header its outbox row carries. The header is read rather than
// inferred: "each row handled under its own tenant" is a statement about the
// envelope every downstream service routes on, and a payload alone cannot show
// it.
type tenantedUnwind struct {
	tenantId string
	payload  sharedsaga.TradeUnwindPayload
}

func unwindsWithTenant(t *testing.T, db *gorm.DB) []tenantedUnwind {
	t.Helper()
	var rows []outbox.Entity
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	var out []tenantedUnwind
	for _, r := range rows {
		var s sharedsaga.Saga
		if err := json.Unmarshal(r.MessageValue, &s); err != nil {
			// Not every outbox row is a saga command; the status events share
			// the table.
			continue
		}
		if len(s.Steps) != 1 || s.Steps[0].Action != sharedsaga.TradeUnwind {
			continue
		}
		payload, ok := s.Steps[0].Payload.(sharedsaga.TradeUnwindPayload)
		if !ok {
			t.Fatalf("unwind payload type: got %T", s.Steps[0].Payload)
		}
		// Header VALUES are base64 inside the stored jsonb (the tenant version
		// headers carry NUL bytes and invalid UTF-8, which jsonb rejects); keys
		// are plain. See libs/atlas-outbox/headers.go.
		var headers map[string]string
		if err := json.Unmarshal(r.Headers, &headers); err != nil {
			t.Fatalf("decode outbox headers: %v", err)
		}
		id, err := base64.StdEncoding.DecodeString(headers["TENANT_ID"])
		if err != nil {
			t.Fatalf("decode TENANT_ID header: %v", err)
		}
		out = append(out, tenantedUnwind{tenantId: string(id), payload: payload})
	}
	return out
}

func unwindPayloads(t *testing.T, db *gorm.DB) []sharedsaga.TradeUnwindPayload {
	t.Helper()
	var out []sharedsaga.TradeUnwindPayload
	for _, u := range unwindsWithTenant(t, db) {
		out = append(out, u.payload)
	}
	return out
}

// mesoRow re-reads one escrow meso row so a test can assert on what the sweep
// left behind.
func mesoRow(t *testing.T, db *gorm.DB, tm tenant.Model, roomId uuid.UUID, ownerId character.Id) (escrow.MesoModel, bool) {
	t.Helper()
	rows, err := escrow.MesosByRoom(db, tm.Id())(roomId)
	if err != nil {
		t.Fatalf("read escrow mesos: %v", err)
	}
	for _, r := range rows {
		if r.OwnerId() == ownerId {
			return r, true
		}
	}
	return escrow.MesoModel{}, false
}

// --- the escrow sweep -----------------------------------------------------------

// TestReconcileEscrowReturnsAStrandedRowToItsOwner pins the base case of design
// §5A.9. Rooms are process-local, so an escrow row that survives a restart is an
// asset nobody can reach — the player's item, held by a trade that no longer
// exists.
func TestReconcileEscrowReturnsAStrandedRowToItsOwner(t *testing.T) {
	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	roomId := uuid.New()
	escrowId := seedEscrowItem(t, db, tm, roomId, 100, 1)

	if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}

	payloads := unwindPayloads(t, db)
	if len(payloads) != 1 {
		t.Fatalf("unwind sagas: got %d, want 1", len(payloads))
	}
	if len(payloads[0].Items) != 1 {
		t.Fatalf("unwind items: got %d, want 1", len(payloads[0].Items))
	}
	if payloads[0].Items[0].OwnerId != 100 {
		t.Errorf("owner: got %d, want 100", payloads[0].Items[0].OwnerId)
	}
	if payloads[0].Items[0].Item.EscrowId != escrowId {
		t.Errorf("escrow id: got %s, want %s", payloads[0].Items[0].Item.EscrowId, escrowId)
	}
	// The snapshot is what re-materialises the asset; an unwind that carried the
	// row's identity but not its stats would hand back a degraded item.
	if payloads[0].Items[0].Item.Snapshot.TemplateId != 2000000 {
		t.Errorf("snapshot template: got %d, want 2000000", payloads[0].Items[0].Item.Snapshot.TemplateId)
	}
}

// TestReconcileEscrowSkipsARowAlreadyClaimedForReturn pins that the boot sweep
// respects the return claim.
//
// A row can be latched before a restart and still be sitting in the table
// afterwards: the claiming unwind commits with the row, and its
// release_from_trade lands whenever the orchestrator gets to it. The sweep sees
// only "a row with no room", which is indistinguishable from genuinely stranded
// — so without the claim it submits a second trade_unwind, and the owner is
// handed the item twice. The claim is a column precisely so it survives the
// restart this sweep exists for.
//
// The second row is the control: latching one row must not silence the sweep for
// the rest of the room.
func TestReconcileEscrowSkipsARowAlreadyClaimedForReturn(t *testing.T) {
	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	roomId := uuid.New()
	claimedId := seedEscrowItem(t, db, tm, roomId, 100, 1)
	strandedId := seedEscrowItem(t, db, tm, roomId, 100, 2)

	won, err := escrow.ClaimItemForReturn(db, tm.Id())(claimedId)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if !won {
		t.Fatal("the pre-restart claim on an unclaimed row must win")
	}

	if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}

	payloads := unwindPayloads(t, db)
	if len(payloads) != 1 {
		t.Fatalf("unwind sagas: got %d, want 1", len(payloads))
	}
	if len(payloads[0].Items) != 1 {
		t.Fatalf("unwind items: got %d, want only the unclaimed row", len(payloads[0].Items))
	}
	if payloads[0].Items[0].Item.EscrowId != strandedId {
		t.Errorf("unwound row: got %s, want the unclaimed %s — the claimed row's return is already in flight", payloads[0].Items[0].Item.EscrowId, strandedId)
	}
}

// TestReconcileEscrowSubmitsNothingWhenEveryRowIsClaimed pins the empty-payload
// guard on the sweep's path: a room whose rows were all taken by somebody else
// must submit no saga at all rather than an unwind with no legs.
func TestReconcileEscrowSubmitsNothingWhenEveryRowIsClaimed(t *testing.T) {
	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	roomId := uuid.New()
	for slot := byte(1); slot <= 2; slot++ {
		id := seedEscrowItem(t, db, tm, roomId, 100, slot)
		won, err := escrow.ClaimItemForReturn(db, tm.Id())(id)
		if err != nil {
			t.Fatalf("claim: %v", err)
		}
		if !won {
			t.Fatalf("claim on unclaimed row %s must win", id)
		}
	}

	if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}
	if payloads := unwindPayloads(t, db); len(payloads) != 0 {
		t.Errorf("unwind sagas: got %d, want 0", len(payloads))
	}
}

// TestReconcileEscrowSubmitsOneUnwindPerRoom pins the grouping. A room that held
// several rows produces ONE saga: the orchestrator expands a composite, so one
// saga per row would multiply the round trips by the size of the trade window
// for no gain.
func TestReconcileEscrowSubmitsOneUnwindPerRoom(t *testing.T) {
	serveLocations(t, map[character.Id][2]byte{200: {1, 1}})

	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	roomId := uuid.New()
	for slot := byte(1); slot <= 3; slot++ {
		seedEscrowItem(t, db, tm, roomId, 100, slot)
	}
	seedEscrowMeso(t, db, tm, roomId, 200, 5000)

	if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}

	payloads := unwindPayloads(t, db)
	if len(payloads) != 1 {
		t.Fatalf("unwind sagas: got %d, want 1 for the whole room", len(payloads))
	}
	if len(payloads[0].Items) != 3 {
		t.Errorf("unwind items: got %d, want 3", len(payloads[0].Items))
	}
	if len(payloads[0].Mesos) != 1 {
		t.Errorf("unwind meso legs: got %d, want 1", len(payloads[0].Mesos))
	}
}

// TestReconcileAtBootSkipsRoomsAnUnresolvedSettlementStillOwns is the
// anti-double-delivery guard of design §5A.9.
//
// A settlement still in flight legitimately owns its escrow rows — its own
// release or unwind will consume them. Sweeping them as stranded would hand the
// giver back an item the settlement then also delivers to the receiver, minting
// it.
func TestReconcileAtBootSkipsRoomsAnUnresolvedSettlementStillOwns(t *testing.T) {
	// The orchestrator reports the saga still running, so Reconcile leaves the
	// record in place and the room stays owned for the whole sweep.
	serveSagaOutcome(t, false)

	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	settledRoom := uuid.New()
	strandedRoom := uuid.New()
	seedSettlementRecord(t, db, tm, settledRoom)
	seedEscrowItem(t, db, tm, settledRoom, 100, 1)
	seedEscrowItem(t, db, tm, strandedRoom, 300, 1)

	if err := ReconcileAtBoot(reconcileLogger(), context.Background(), db); err != nil {
		t.Fatalf("reconcile at boot: %v", err)
	}

	payloads := unwindPayloads(t, db)
	if len(payloads) != 1 {
		t.Fatalf("unwind sagas: got %d, want 1 — only the room no settlement owns", len(payloads))
	}
	// The control leg proves the sweep ran at all, so the skip above is not a
	// sweep that simply did nothing.
	if len(payloads[0].Items) != 1 || payloads[0].Items[0].OwnerId != 300 {
		t.Fatalf("the unwind is not the stranded room's: %+v", payloads[0].Items)
	}
	// The owned room's row is still in escrow, waiting for its settlement.
	rows, err := escrow.ItemsByRoom(db, tm.Id())(settledRoom)
	if err != nil {
		t.Fatalf("read escrow items: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("the owned room's escrow rows: got %d, want 1 left untouched", len(rows))
	}
}

// TestReconcileAtBootCapturesTheExclusionSetBeforeTheSettlementPass is the
// ordering that the skip above depends on.
//
// Reconcile DELETES each record as it resolves it. Reading the unresolved set
// after that pass would return nothing, and the sweep would treat rows belonging
// to a just-resolved settlement — whose release saga is still in flight, moments
// behind us in the same process — as stranded. Those are precisely the rows a
// double delivery comes from.
//
// The case is built so the two readings disagree: the orchestrator reports the
// saga complete, so the record exists when ReconcileAtBoot starts and is gone
// before ReconcileEscrow reads anything.
func TestReconcileAtBootCapturesTheExclusionSetBeforeTheSettlementPass(t *testing.T) {
	serveSagaOutcome(t, true)

	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	resolvedRoom := uuid.New()
	strandedRoom := uuid.New()
	transactionId := seedSettlementRecord(t, db, tm, resolvedRoom)
	seedEscrowItem(t, db, tm, resolvedRoom, 100, 1)
	seedEscrowItem(t, db, tm, strandedRoom, 300, 1)

	if err := ReconcileAtBoot(reconcileLogger(), context.Background(), db); err != nil {
		t.Fatalf("reconcile at boot: %v", err)
	}

	// Non-vacuity: the record really was resolved DURING this run, so a sweep
	// that re-read the unresolved set would have found the room unowned.
	ctx := tenant.WithContext(context.Background(), tm)
	if _, err := settlement.NewProcessor(reconcileLogger(), ctx, db).GetByTransactionId(transactionId); err == nil {
		t.Fatal("the settlement record survived the reconciliation pass; the case does not exercise the ordering")
	}

	payloads := unwindPayloads(t, db)
	if len(payloads) != 1 {
		t.Fatalf("unwind sagas: got %d, want 1 — only the room no settlement owned", len(payloads))
	}
	if len(payloads[0].Items) != 1 || payloads[0].Items[0].OwnerId != 300 {
		t.Errorf("the just-resolved settlement's escrow was swept as stranded: %+v", payloads[0].Items)
	}
}

// TestReconcileEscrowHandlesEachTenantUnderItsOwnTenant pins the cross-tenant
// shape of the sweep. AllItems is deliberately un-scoped, so the tenant a row is
// acted upon under can only come from the row itself.
func TestReconcileEscrowHandlesEachTenantUnderItsOwnTenant(t *testing.T) {
	db := reconcileDb(t)
	first := reconcileTenant(t, "first")
	second := reconcileTenant(t, "second")
	seedEscrowItem(t, db, first, uuid.New(), 100, 1)
	seedEscrowItem(t, db, second, uuid.New(), 300, 1)

	if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}

	unwinds := unwindsWithTenant(t, db)
	if len(unwinds) != 2 {
		t.Fatalf("unwind sagas: got %d, want one per tenant", len(unwinds))
	}
	byOwner := make(map[character.Id]string)
	for _, u := range unwinds {
		if len(u.payload.Items) != 1 {
			t.Fatalf("unwind items: got %d, want 1", len(u.payload.Items))
		}
		byOwner[u.payload.Items[0].OwnerId] = u.tenantId
	}
	if got := byOwner[100]; got != first.Id().String() {
		t.Errorf("owner 100's unwind ran under tenant %s, want %s", got, first.Id())
	}
	if got := byOwner[300]; got != second.Id().String() {
		t.Errorf("owner 300's unwind ran under tenant %s, want %s", got, second.Id())
	}
}

// TestUnwindStrandedSkipsAMesoOwnerItCannotLocate pins unwindStranded's field
// resolution. The room that would have supplied the world and channel is gone,
// so each meso leg reads its owner's CURRENT field; an owner who cannot be found
// is skipped rather than defaulted to world 0, because a refund announced onto
// the wrong channel is worse than one that waits for the next boot.
//
// The other legs must still go out: one unreachable character cannot hold the
// rest of the room's escrow hostage.
func TestUnwindStrandedSkipsAMesoOwnerItCannotLocate(t *testing.T) {
	// 200 is locatable, 400 is not.
	serveLocations(t, map[character.Id][2]byte{200: {1, 7}})

	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	roomId := uuid.New()
	seedEscrowItem(t, db, tm, roomId, 100, 1)
	seedEscrowMeso(t, db, tm, roomId, 200, 5000)
	seedEscrowMeso(t, db, tm, roomId, 400, 7000)

	if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}

	payloads := unwindPayloads(t, db)
	if len(payloads) != 1 {
		t.Fatalf("unwind sagas: got %d, want 1", len(payloads))
	}
	if len(payloads[0].Items) != 1 {
		t.Errorf("the item leg was dropped along with the unlocatable meso owner: %+v", payloads[0].Items)
	}
	if len(payloads[0].Mesos) != 1 {
		t.Fatalf("meso legs: got %d, want 1 — only the locatable owner's", len(payloads[0].Mesos))
	}
	leg := payloads[0].Mesos[0]
	if leg.CharacterId != 200 || leg.Amount != 5000 {
		t.Errorf("meso leg: got character %d for %d, want 200 for 5000", leg.CharacterId, leg.Amount)
	}
	// The world and channel come from the owner's CURRENT field, not from a
	// zero default.
	if leg.WorldId != 1 || leg.ChannelId != 7 {
		t.Errorf("meso leg field: got world %d channel %d, want 1/7", leg.WorldId, leg.ChannelId)
	}
	// The skipped owner's row keeps its amount, so the next boot retries it.
	row, ok := mesoRow(t, db, tm, roomId, 400)
	if !ok {
		t.Fatal("the unlocatable owner's escrow row was removed; nothing is left to refund from")
	}
	if row.Amount() != 7000 {
		t.Errorf("skipped row amount: got %d, want 7000 left for the next boot", row.Amount())
	}
}

// TestUnwindZeroesARefundedMesoRowAndKeepsItsPendingStake pins
// clearRefundedMesos.
//
// The row is ZEROED rather than deleted, and both halves matter. Zeroed, because
// the unwind saga refunds meso through a bare award_mesos that deletes nothing —
// a surviving non-zero row would be refunded again by the next sweep. Not
// deleted, because a stake still in flight resolves against this row by its
// pending_stake_id, and removing it would strand a debit the player has already
// been charged with no record left to refund from.
func TestUnwindZeroesARefundedMesoRowAndKeepsItsPendingStake(t *testing.T) {
	serveLocations(t, map[character.Id][2]byte{200: {1, 1}})

	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	roomId := uuid.New()
	stakeId := uuid.New()
	// A committed total of 5000 with a retype down to 900 still in flight — the
	// state a restart can genuinely catch a staging player in. The armed stake
	// therefore moves -4100.
	if err := escrow.ArmMesoStake(db, tm)(roomId, 200, stakeId, 900, -4_100); err != nil {
		t.Fatalf("arm stake: %v", err)
	}
	seedEscrowMeso(t, db, tm, roomId, 200, 5000)

	if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}

	payloads := unwindPayloads(t, db)
	if len(payloads) != 1 || len(payloads[0].Mesos) != 1 || payloads[0].Mesos[0].Amount != 5000 {
		t.Fatalf("expected one unwind refunding 5000, got %+v", payloads)
	}
	row, ok := mesoRow(t, db, tm, roomId, 200)
	if !ok {
		t.Fatal("the refunded row was deleted; an in-flight stake has nothing left to resolve against")
	}
	if row.Amount() != 0 {
		t.Errorf("refunded row amount: got %d, want 0", row.Amount())
	}
	if row.PendingStakeId() != stakeId {
		t.Errorf("pending stake id: got %s, want the armed %s", row.PendingStakeId(), stakeId)
	}
	if row.PendingAmount() != 900 {
		t.Errorf("pending amount: got %d, want 900", row.PendingAmount())
	}
	// The armed DELTA has to survive the zeroing too. It is the only surviving
	// record of what the in-flight saga actually moved: Amount is now 0, so a
	// refund derived from it would hand back the whole stake on top of the 5000
	// this unwind just returned.
	if row.PendingDelta() != -4_100 {
		t.Errorf("pending delta: got %d, want the armed -4100", row.PendingDelta())
	}
}

// TestASecondSweepOverARefundedRoomSubmitsNothing is the double-refund the
// zeroing prevents, stated as behaviour: a boot that sweeps a room whose meso
// was already returned must submit nothing at all.
func TestASecondSweepOverARefundedRoomSubmitsNothing(t *testing.T) {
	serveLocations(t, map[character.Id][2]byte{200: {1, 1}})

	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	roomId := uuid.New()
	seedEscrowMeso(t, db, tm, roomId, 200, 5000)

	for i := 0; i < 2; i++ {
		if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
			t.Fatalf("reconcile escrow pass %d: %v", i+1, err)
		}
	}

	payloads := unwindPayloads(t, db)
	if len(payloads) != 1 {
		t.Fatalf("unwind sagas after two sweeps: got %d, want 1 — the second sweep refunded the meso again", len(payloads))
	}
}

// TestReconcileEscrowIgnoresAnAlreadyZeroedMesoRow pins the same guard one level
// down, without depending on a prior sweep having produced the zero: a row at
// zero is not a refund waiting to happen, and a saga carrying only zero-amount
// legs would be pure noise.
func TestReconcileEscrowIgnoresAnAlreadyZeroedMesoRow(t *testing.T) {
	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	roomId := uuid.New()
	seedEscrowMeso(t, db, tm, roomId, 200, 0)

	if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}
	if payloads := unwindPayloads(t, db); len(payloads) != 0 {
		t.Errorf("unwind sagas: got %d, want 0", len(payloads))
	}
}

// TestReconcileEscrowIsANoOpWithNothingInEscrow pins the ordinary boot: the vast
// majority of restarts have no escrow at all and must publish nothing.
func TestReconcileEscrowIsANoOpWithNothingInEscrow(t *testing.T) {
	db := reconcileDb(t)
	if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}
	if payloads := unwindPayloads(t, db); len(payloads) != 0 {
		t.Errorf("unwind sagas: got %d, want 0", len(payloads))
	}
}

// sortedOwners is a stable read of an unwind's item owners, so a test can assert
// on the set without depending on map iteration order.
func sortedOwners(p sharedsaga.TradeUnwindPayload) []character.Id {
	out := make([]character.Id, 0, len(p.Items))
	for _, i := range p.Items {
		out = append(out, i.OwnerId)
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}

// TestReconcileEscrowReturnsBothSidesOfADeadRoom pins that grouping is by ROOM,
// not by owner: a dead room holds both participants' escrow, and each item goes
// back to the person it came from inside the one saga.
func TestReconcileEscrowReturnsBothSidesOfADeadRoom(t *testing.T) {
	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	roomId := uuid.New()
	seedEscrowItem(t, db, tm, roomId, 100, 1)
	seedEscrowItem(t, db, tm, roomId, 200, 2)

	if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}

	payloads := unwindPayloads(t, db)
	if len(payloads) != 1 {
		t.Fatalf("unwind sagas: got %d, want 1", len(payloads))
	}
	owners := sortedOwners(payloads[0])
	if len(owners) != 2 || owners[0] != 100 || owners[1] != 200 {
		t.Errorf("item owners: got %v, want [100 200]", owners)
	}
}
