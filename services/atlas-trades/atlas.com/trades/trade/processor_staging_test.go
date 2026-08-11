package trade

import (
	"atlas-trades/configuration"
	inventorydata "atlas-trades/data/inventory"
	"atlas-trades/escrow"
	"atlas-trades/kafka/message"
	trademsg "atlas-trades/kafka/message/trade"
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// --- fakes for the staging seams ---------------------------------------------

// assetKey addresses one row of the fake inventory service.
type assetKey struct {
	characterId   character.Id
	inventoryType inventory.Type
	sourceSlot    slot.Position
}

type fakeInventory struct {
	assets map[assetKey]inventorydata.Asset
	err    error
	// capacity is the slot count every compartment reports. Zero means the
	// default 24 — the settlement free-slot check is the only reader that cares.
	capacity uint32
	// onGetCompartment fires on each compartment read. The settlement
	// pre-checks do their reads between snapshotting the room and ending it,
	// so this is the seam a test uses to land a concurrent transition in
	// exactly that window.
	onGetCompartment func()
}

func (f *fakeInventory) AssetInSlot(characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position) (inventorydata.Asset, error) {
	if f.err != nil {
		return inventorydata.Asset{}, f.err
	}
	a, ok := f.assets[assetKey{characterId, inventoryType, sourceSlot}]
	if !ok {
		return inventorydata.Asset{}, inventorydata.ErrAssetNotFound
	}
	return a, nil
}

// GetCompartment projects the fake's flat asset map into the compartment the
// caller asked for. Each asset's slot comes from the map KEY, not from the
// stored Asset, so a test can place the same asset wherever it needs it.
func (f *fakeInventory) GetCompartment(characterId character.Id, inventoryType inventory.Type) (inventorydata.Model, error) {
	if f.onGetCompartment != nil {
		f.onGetCompartment()
	}
	if f.err != nil {
		return inventorydata.Model{}, f.err
	}
	assets := make([]inventorydata.Asset, 0)
	for k, a := range f.assets {
		if k.characterId != characterId || k.inventoryType != inventoryType {
			continue
		}
		assets = append(assets, inventorydata.NewAsset(a.Id(), k.sourceSlot, a.TemplateId(), a.Quantity(), a.Flag()))
	}
	capacity := f.capacity
	if capacity == 0 {
		capacity = 24
	}
	return inventorydata.NewModel(uuid.New(), inventoryType, capacity, assets), nil
}

type fakeItemData struct {
	blocked map[item.Id]bool
	err     error
	// slotMax is the per-template stack ceiling the settlement free-slot check
	// counts merges with. An absent template reports defaultSlotMax.
	slotMax map[item.Id]uint32
	// slotMaxErr fails the slotMax lookup only, leaving tradeBlock readable —
	// the two are separate atlas-data reads on the settlement path.
	slotMaxErr error
}

// defaultSlotMax is what the fake reports for a template it has no entry for.
// It is deliberately larger than any quantity the tests stage, so a merge is
// possible whenever the templates match and a test that wants a NEW slot
// consumed uses distinct templates rather than relying on a full stack.
const defaultSlotMax = uint32(200)

func (f *fakeItemData) TradeBlock(_ inventory.Type, templateId item.Id) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.blocked[templateId], nil
}

func (f *fakeItemData) SlotMax(_ inventory.Type, templateId item.Id) (uint32, error) {
	if f.slotMaxErr != nil {
		return 0, f.slotMaxErr
	}
	if m, ok := f.slotMax[templateId]; ok {
		return m, nil
	}
	return defaultSlotMax, nil
}

// --- the fake custody store ---------------------------------------------------

// escrowStat* / escrowOwnerName / escrowCashId / escrowPetId are the snapshot
// every accepted escrow row carries in these tests. They are deliberately
// non-zero and mutually distinct so a settlement payload that dropped the
// snapshot — or read it from the room rather than from the custody row — shows
// up as a concrete wrong number rather than as a plausible zero.
//
// The cash serial, the expiry and the pet id are here because they are what the
// original escrow column set silently dropped: cash items and pets are stageable
// (checkRestrictions blocks equipped items, the untradeable flags and the WZ
// tradeBlock, nothing else), so an item could come back out of a trade stripped
// of its identity.
const (
	escrowStatWeaponAttack = uint16(17)
	escrowStatSlots        = uint16(7)
	escrowOwnerName        = "Chronicle"
	escrowCashId           = int64(4815162342)
	escrowPetId            = uint32(909)
)

// escrowExpiration is UTC-fixed so a round trip through Postgres compares equal.
var escrowExpiration = time.Date(2031, 4, 5, 6, 7, 8, 0, time.UTC)

// escrowSnapshotFor builds the snapshot a completed transfer_to_trade would have
// captured for one staged item.
func escrowSnapshotFor(i StagedItem) sharedsaga.AssetSnapshot {
	return sharedsaga.AssetSnapshot{
		Slot:         int16(i.SourceSlot()),
		TemplateId:   uint32(i.TemplateId()),
		Quantity:     uint32(i.Quantity()),
		Expiration:   escrowExpiration,
		CashId:       escrowCashId,
		Owner:        escrowOwnerName,
		WeaponAttack: escrowStatWeaponAttack,
		Slots:        escrowStatSlots,
		PetId:        escrowPetId,
	}
}

// mesoOwner keys one participant's escrowed meso within one room.
type mesoOwner struct {
	roomId  uuid.UUID
	ownerId character.Id
}

// fakeEscrow stands in for atlas-trades' own custody store.
//
// It is a fake rather than the real escrowReader because the WRITES come from
// the custody consumer in production — a different service boundary entirely,
// driven by the orchestrator rather than by anything these tests call. `accept`,
// `release` and `setMeso` stand in for it, so a test states exactly what is in
// custody at the moment it matters instead of having to drive a whole saga round
// trip to get an item into a row.
type fakeEscrow struct {
	mutex sync.Mutex
	// items is insertion-ordered, matching ItemsByRoom's created-at ordering.
	items []escrow.ItemModel
	mesos map[mesoOwner]int64
	// claimed holds the rows some return path has already taken. It reproduces
	// the real returning_at column rather than the whole row lifecycle, because
	// that column is the only thing standing between the teardown path and the
	// orphaned-stage path both returning one row.
	claimed map[uuid.UUID]struct{}
	err     error
}

func newFakeEscrow() *fakeEscrow {
	return &fakeEscrow{mesos: make(map[mesoOwner]int64), claimed: make(map[uuid.UUID]struct{})}
}

// withTx returns the same fake. It holds no database handle to rebind, and
// returning itself is what keeps a test's `accept`/`setMeso` calls visible to
// the reads the processor issues from inside its transaction.
func (f *fakeEscrow) withTx(*gorm.DB) escrowStore { return f }

func (f *fakeEscrow) ItemById(escrowId uuid.UUID) (escrow.ItemModel, bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.err != nil {
		return escrow.ItemModel{}, false, f.err
	}
	for _, m := range f.items {
		if m.Id() == escrowId {
			return m, true, nil
		}
	}
	return escrow.ItemModel{}, false, nil
}

func (f *fakeEscrow) ItemsByRoom(roomId uuid.UUID) ([]escrow.ItemModel, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	var out []escrow.ItemModel
	for _, m := range f.items {
		if m.RoomId() == roomId {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *fakeEscrow) MesosByRoom(roomId uuid.UUID) ([]escrow.MesoModel, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	keys := make([]mesoOwner, 0, len(f.mesos))
	for k := range f.mesos {
		if k.roomId == roomId {
			keys = append(keys, k)
		}
	}
	// owner_id ASC, matching the real provider's ordering.
	sort.Slice(keys, func(i, j int) bool { return keys[i].ownerId < keys[j].ownerId })
	out := make([]escrow.MesoModel, 0, len(keys))
	for _, k := range keys {
		m, err := escrow.MakeMeso(escrow.MesoEntity{
			Id:      uuid.New(),
			RoomId:  k.roomId,
			OwnerId: k.ownerId,
			Amount:  f.mesos[k],
		})
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (f *fakeEscrow) MesoByOwner(roomId uuid.UUID, ownerId character.Id) (int64, bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.err != nil {
		return 0, false, f.err
	}
	amount, ok := f.mesos[mesoOwner{roomId: roomId, ownerId: ownerId}]
	return amount, ok, nil
}

// ClaimForReturn is the fake's compare-and-set. Like the real UPDATE it is one
// indivisible step under the store's own lock, so a caller cannot observe
// "unclaimed" and then lose the row to somebody else before it acts.
func (f *fakeEscrow) ClaimForReturn(escrowId uuid.UUID, _ uuid.UUID) (bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.err != nil {
		return false, f.err
	}
	if _, taken := f.claimed[escrowId]; taken {
		return false, nil
	}
	f.claimed[escrowId] = struct{}{}
	return true, nil
}

// accept writes the custody row a completed transfer_to_trade would have
// written, from the staged item that named it.
func (f *fakeEscrow) accept(roomId uuid.UUID, ownerId character.Id, i StagedItem) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.items = append(f.items, escrow.NewItemBuilder(i.EscrowId(), roomId, ownerId).
		SetTradeSlot(i.TradeSlot()).
		SetSource(i.InventoryType(), i.AssetId()).
		SetSnapshot(escrowSnapshotFor(i)).
		Build())
}

// release drops one custody row, as a release_from_trade would.
func (f *fakeEscrow) release(escrowId uuid.UUID) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	for i := range f.items {
		if f.items[i].Id() != escrowId {
			continue
		}
		f.items = append(f.items[:i], f.items[i+1:]...)
		return
	}
}

// setMeso records a participant's COMMITTED escrowed meso total in the FAKE
// only. It is the fake half of commitEscrowedMeso and has no other caller by
// design: the delta arithmetic reads the real store, so seeding just this one
// describes a state that cannot occur and quietly stops testing the netting.
func (f *fakeEscrow) setMeso(roomId uuid.UUID, ownerId character.Id, amount int64) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.mesos[mesoOwner{roomId: roomId, ownerId: ownerId}] = amount
}

// --- harness -----------------------------------------------------------------

// stagingAsset is the plain, tradeable use-item every staging test stages
// unless it says otherwise: 100 of template 2000000 in USE slot 1.
const (
	stagingTemplateId = item.Id(2000000)
	stagingSourceSlot = slot.Position(1)
	stagingAssetId    = asset.Id(7001)
)

// newStagingProcessor builds an OPEN room between 100 and 200 with both sides
// holding the same stageable stack, then swaps in the staging seams.
func newStagingProcessor(t *testing.T, cfg configuration.Model, characters ...testCharacter) (*ProcessorImpl, *emitted) {
	t.Helper()

	p, e := newLifecycleProcessor(t, cfg, characters...)

	assets := make(map[assetKey]inventorydata.Asset)
	for _, c := range characters {
		assets[assetKey{c.Id, inventory.TypeValueUse, stagingSourceSlot}] = inventorydata.NewAsset(stagingAssetId, stagingSourceSlot, stagingTemplateId, 100, 0)
		assets[assetKey{c.Id, inventory.TypeValueUse, 2}] = inventorydata.NewAsset(stagingAssetId+1, 2, stagingTemplateId+1, 100, 0)
		assets[assetKey{c.Id, inventory.TypeValueUse, 3}] = inventorydata.NewAsset(stagingAssetId+2, 3, stagingTemplateId+2, 100, 0)
	}
	p.invp = &fakeInventory{assets: assets}
	p.idp = &fakeItemData{blocked: make(map[item.Id]bool)}

	if err := p.CreateRoom(uuid.New(), testField(t), 100, miniroom.Trade); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.Invite(uuid.New(), testField(t), 100, 200); err != nil {
		t.Fatalf("invite: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle(), room.RoomType()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	return p, e
}

// testOpenRoom is the default staged-room harness: shipped config, the three
// default characters, both trading sides seated.
func testOpenRoom(t *testing.T) (*ProcessorImpl, *emitted) {
	t.Helper()
	return newStagingProcessor(t, configuration.DefaultConfig(), defaultCharacters()...)
}

// testOpenRoomWithConfig is testOpenRoom under a caller-chosen tenant config.
func testOpenRoomWithConfig(t *testing.T, cfg configuration.Model) (*ProcessorImpl, *emitted) {
	t.Helper()
	return newStagingProcessor(t, cfg, defaultCharacters()...)
}

// testOpenRoomWithMeso is testOpenRoom with the named character holding meso.
func testOpenRoomWithMeso(t *testing.T, characterId character.Id, meso uint32) (*ProcessorImpl, *emitted) {
	t.Helper()
	rows := defaultCharacters()
	for i := range rows {
		if rows[i].Id == characterId {
			rows[i].Meso = meso
		}
	}
	return newStagingProcessor(t, configuration.DefaultConfig(), rows...)
}

// escrowOf returns the processor's fake custody store.
func escrowOf(t *testing.T, p *ProcessorImpl) *fakeEscrow {
	t.Helper()
	f, ok := p.esc.(*fakeEscrow)
	if !ok {
		t.Fatalf("escrow store: got %T, want the fake", p.esc)
	}
	return f
}

// stagedItemsOf returns the items the character currently has staged.
func stagedItemsOf(t *testing.T, p *ProcessorImpl, characterId character.Id) []StagedItem {
	t.Helper()
	room, ok := p.RoomForCharacter(characterId)
	if !ok {
		t.Fatalf("character %d holds no room", characterId)
	}
	pt, ok := room.ParticipantFor(characterId)
	if !ok {
		t.Fatalf("character %d is not seated in their room", characterId)
	}
	return pt.Items()
}

// pendingItemIn returns the character's staged item occupying the given trade
// slot.
func pendingItemIn(t *testing.T, p *ProcessorImpl, characterId character.Id, tradeSlot byte) StagedItem {
	t.Helper()
	for _, i := range stagedItemsOf(t, p, characterId) {
		if i.TradeSlot() == tradeSlot {
			return i
		}
	}
	t.Fatalf("character %d has nothing staged in trade slot %d", characterId, tradeSlot)
	return StagedItem{}
}

// stageOne drives one PUT_ITEM through the WHOLE custody round trip: the
// command, the escrow row the orchestrator's accept_to_trade would have written,
// and the terminal status that clears the pending flag. It returns the escrow
// row id.
//
// Every harness that needs a CONFIRMED staged item goes through this, because a
// stage is no longer complete when PutItem returns — under escrow-at-staging the
// item is pending, unannounced and un-settleable until its saga terminates.
func stageOne(t *testing.T, p *ProcessorImpl, characterId character.Id, sourceSlot slot.Position, quantity uint16, targetSlot byte) uuid.UUID {
	t.Helper()
	if err := p.PutItem(uuid.New(), characterId, byte(inventory.TypeValueUse), sourceSlot, quantity, targetSlot); err != nil {
		t.Fatalf("put item for %d: %v", characterId, err)
	}
	room, ok := p.RoomForCharacter(characterId)
	if !ok {
		t.Fatalf("character %d holds no room after staging", characterId)
	}
	i := pendingItemIn(t, p, characterId, targetSlot)
	escrowOf(t, p).accept(room.Id(), characterId, i)
	claimed, err := p.StageSucceeded(uuid.New(), i.EscrowId())
	if err != nil {
		t.Fatalf("stage succeeded for %d: %v", characterId, err)
	}
	if !claimed {
		t.Fatalf("stage [%s] was not claimed by StageSucceeded", i.EscrowId())
	}
	return i.EscrowId()
}

// mesoStagedBy returns the character's currently staged meso.
func mesoStagedBy(t *testing.T, p *ProcessorImpl, characterId character.Id) uint32 {
	t.Helper()
	room, ok := p.RoomForCharacter(characterId)
	if !ok {
		t.Fatalf("character %d holds no room", characterId)
	}
	pt, _ := room.ParticipantFor(characterId)
	return pt.MesoStaged()
}

// --- PUT_ITEM ----------------------------------------------------------------

// TestPutItemSubmitsATransferToTradeKeyedByItsEscrowRow pins design §5A.4's
// whole staging contract in one place: the stage is a real custody transfer, the
// saga's transaction id IS the escrow row id (so a terminal status needs no
// lookup table to find the dialog slot it belongs to), and NOTHING is announced
// yet — the item is pending until the row actually exists.
func TestPutItemSubmitsATransferToTradeKeyedByItsEscrowRow(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 3); err != nil {
		t.Fatalf("put item: %v", err)
	}

	items := stagedItemsOf(t, p, 100)
	if len(items) != 1 {
		t.Fatalf("staged items: got %d, want 1", len(items))
	}
	if !items[0].Pending() {
		t.Error("the staged item is not pending; nothing has written its escrow row yet")
	}
	if items[0].TradeSlot() != 3 || items[0].Quantity() != 5 || items[0].TemplateId() != stagingTemplateId {
		t.Errorf("staged item: got %+v, want trade slot 3 quantity 5 of template %d", items[0], stagingTemplateId)
	}
	if items[0].EscrowId() == uuid.Nil {
		t.Fatal("staged item carries no escrow id; nothing could ever return it")
	}

	sagas := stageSagas(t, e)
	if len(sagas) != 1 {
		t.Fatalf("transfer_to_trade sagas: got %d, want 1", len(sagas))
	}
	if sagas[0].SagaType != sharedsaga.TradeStaging {
		t.Errorf("sagaType: got %s, want %s", sagas[0].SagaType, sharedsaga.TradeStaging)
	}
	if sagas[0].TransactionId != items[0].EscrowId() {
		t.Errorf("saga transactionId: got %s, want the escrow row id %s", sagas[0].TransactionId, items[0].EscrowId())
	}
	payload := stagePayloadOf(t, sagas[0])
	if payload.TransactionId != items[0].EscrowId() || payload.EscrowId != items[0].EscrowId() {
		t.Errorf("payload ids: got transaction %s escrow %s, want both to be %s", payload.TransactionId, payload.EscrowId, items[0].EscrowId())
	}
	room, _ := p.RoomForCharacter(100)
	if payload.RoomId != room.Id() {
		t.Errorf("payload roomId: got %s, want %s", payload.RoomId, room.Id())
	}
	if payload.CharacterId != 100 || payload.TradeSlot != 3 {
		t.Errorf("payload identity: got character %d trade slot %d, want 100 / 3", payload.CharacterId, payload.TradeSlot)
	}
	if payload.SourceInventoryType != byte(inventory.TypeValueUse) || payload.SourceSlot != int16(stagingSourceSlot) {
		t.Errorf("payload source: got compartment %d slot %d, want %d / %d", payload.SourceInventoryType, payload.SourceSlot, byte(inventory.TypeValueUse), stagingSourceSlot)
	}
	if payload.AssetId != uint32(stagingAssetId) || payload.Quantity != 5 {
		t.Errorf("payload asset: got %d x%d, want %d x5", payload.AssetId, payload.Quantity, stagingAssetId)
	}

	// The announcement belongs to StageSucceeded. Announcing here would show
	// both dialogs an item that a failing saga never actually escrowed.
	assertNoEventOfType(t, e, trademsg.StatusTypeItemStaged)
}

// TestPendingStageHoldsItsDialogSlot pins why a pending item is kept in the
// room at all. The transfer's release step unlocks the client BEFORE the escrow
// row exists, so the player can send a second PUT_ITEM into the same slot while
// the first is still in flight; without the hold, both would be accepted and the
// dialog would carry two items in one slot.
func TestPendingStageHoldsItsDialogSlot(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 3); err != nil {
		t.Fatalf("first put item: %v", err)
	}
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), 2, 1, 3); err != nil {
		t.Fatalf("second put item: %v", err)
	}

	if got := len(stagedItemsOf(t, p, 100)); got != 1 {
		t.Errorf("staged items: got %d, want the original 1 — a pending item did not hold its slot", got)
	}
	if got := len(stageSagas(t, e)); got != 1 {
		t.Errorf("transfer_to_trade sagas: got %d, want 1 — the refused stage still moved an item into escrow", got)
	}
}

// TestPendingStageCountsTowardMaxStagedItems pins the cap half of the same
// rule: a pending item occupies one of the tenant's staging slots, so a stage
// that would overshoot the cap while earlier stages are still in flight is
// refused rather than escrowed and then found to have nowhere to go.
func TestPendingStageCountsTowardMaxStagedItems(t *testing.T) {
	p, e := testOpenRoomWithConfig(t, configuration.DefaultConfig().WithMaxStagedItems(2))
	for i, sourceSlot := range []slot.Position{stagingSourceSlot, 2, 3} {
		if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), sourceSlot, 1, byte(i+1)); err != nil {
			t.Fatalf("put item %d: %v", i, err)
		}
	}
	if got := len(stagedItemsOf(t, p, 100)); got != 2 {
		t.Errorf("staged items: got %d, want the configured cap of 2", got)
	}
	if got := len(stageSagas(t, e)); got != 2 {
		t.Errorf("transfer_to_trade sagas: got %d, want 2 — the refused stage must not escrow", got)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeItemStaged)
}

// TestPutItemRejectsOccupiedOrOutOfRangeSlot pins FR-3.3: the dialog's slots are
// 1..maxStagedItems and each holds at most one item. The first stage is driven
// all the way through custody, so this exercises the CONFIRMED-slot arm; the
// pending arm is TestPendingStageHoldsItsDialogSlot.
func TestPutItemRejectsOccupiedOrOutOfRangeSlot(t *testing.T) {
	for name, target := range map[string]byte{"occupied": 3, "zero": 0, "above nine": 10} {
		t.Run(name, func(t *testing.T) {
			p, e := testOpenRoom(t)
			stageOne(t, p, 100, stagingSourceSlot, 5, 3)
			if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), 2, 1, target); err != nil {
				t.Fatalf("second put item: %v", err)
			}
			if got := len(stagedItemsOf(t, p, 100)); got != 1 {
				t.Errorf("staged items: got %d, want the original 1", got)
			}
			if got := len(stageSagas(t, e)); got != 1 {
				t.Errorf("transfer_to_trade sagas: got %d, want 1", got)
			}
		})
	}
}

// TestRestrictionRefusalEmitsItemRefused pins BOTH halves of design §7 + §5A.6,
// which pull in opposite directions and must both hold.
//
// SILENT: the reference client has no put-item-time error for "untradeable", so
// the empty trade slot IS the feedback. No ITEM_STAGED, no ERROR, nothing
// escrowed — a visible error here would be a packet the reference server does
// not send.
//
// BUT NOT NOTHING: the client armed m_bExclRequestSent when it sent PUT_ITEM and
// CanSendExclRequest refuses every later exclusive request until a packet clears
// it. ITEM_REFUSED is that packet, and atlas-channel renders it as the bare
// unlock and nothing else. This test asserted only the silent half until the
// escrow amendment, which is why a player who tried to trade an untradeable item
// wedged their own dialog for the rest of the session.
func TestRestrictionRefusalEmitsItemRefused(t *testing.T) {
	untradeable := inventorydata.NewAsset(stagingAssetId, stagingSourceSlot, stagingTemplateId, 100, uint16(asset.FlagUntradeable))
	tradeBlocked := inventorydata.NewAsset(stagingAssetId, stagingSourceSlot, stagingTemplateId, 100, 0)
	equipped := inventorydata.NewAsset(stagingAssetId, -11, stagingTemplateId, 1, 0)

	for name, tc := range map[string]struct {
		a          inventorydata.Asset
		sourceSlot slot.Position
		blocked    bool
		dataErr    error
	}{
		"untradeable flag": {a: untradeable, sourceSlot: stagingSourceSlot},
		"tradeBlock":       {a: tradeBlocked, sourceSlot: stagingSourceSlot, blocked: true},
		"equipped":         {a: equipped, sourceSlot: -11},
		"unreadable data":  {a: tradeBlocked, sourceSlot: stagingSourceSlot, dataErr: errors.New("atlas-data unreachable")},
	} {
		t.Run(name, func(t *testing.T) {
			p, e := testOpenRoom(t)
			p.invp = &fakeInventory{assets: map[assetKey]inventorydata.Asset{
				{100, inventory.TypeValueUse, tc.sourceSlot}: tc.a,
			}}
			blocked := make(map[item.Id]bool)
			if tc.blocked {
				blocked[stagingTemplateId] = true
			}
			p.idp = &fakeItemData{blocked: blocked, err: tc.dataErr}

			if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), tc.sourceSlot, 1, 1); err != nil {
				t.Fatalf("put item: %v", err)
			}

			assertNoEventOfType(t, e, trademsg.StatusTypeItemStaged)
			assertNoEventOfType(t, e, trademsg.StatusTypeError)
			assertNoStageSubmitted(t, e)
			if got := len(stagedItemsOf(t, p, 100)); got != 0 {
				t.Errorf("staged items: got %d, want 0", got)
			}

			refusals := statusEvents[trademsg.ItemRefusedEventBody](t, e, trademsg.StatusTypeItemRefused)
			if len(refusals) != 1 {
				t.Fatalf("ITEM_REFUSED events: got %d, want 1 — a silent drop leaves the client locked", len(refusals))
			}
			// Addressed to the stager alone: the counterparty was never told the
			// item existed, so it has nothing to correct.
			if refusals[0].CharacterId != 100 {
				t.Errorf("refusal addressed to character %d, want 100", refusals[0].CharacterId)
			}
		})
	}
}

// TestFreezeRuleRefusalEmitsItemRefused is the §3.2 half of the same obligation.
//
// Staging is frozen from the first CONFIRM, and the reference client blocks it
// locally, so reaching the server means a modified client (FR-3.6). The stage is
// still refused rather than honoured — but it is answered, because even a
// modified client is holding a lock that only the server can release, and a
// wedged dialog is not the intended punishment for sending a stale packet.
func TestFreezeRuleRefusalEmitsItemRefused(t *testing.T) {
	p, e := testStagedRoom(t)
	if err := p.Confirm(uuid.New(), 100, nil); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	// The harness already staged and announced items, so the staging assertions
	// below are DELTAS against that baseline. Only the refusal count is absolute,
	// because nothing before this point refuses anything.
	beforeItems := len(stagedItemsOf(t, p, 100))
	beforeStaged := len(statusEvents[trademsg.ItemStagedEventBody](t, e, trademsg.StatusTypeItemStaged))
	beforeSagas := len(stageSagas(t, e))

	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 1, 2); err != nil {
		t.Fatalf("put item: %v", err)
	}

	if got := len(stagedItemsOf(t, p, 100)); got != beforeItems {
		t.Errorf("staged items: got %d, want the pre-confirm %d — a frozen room must not accept a stage", got, beforeItems)
	}
	if got := len(statusEvents[trademsg.ItemStagedEventBody](t, e, trademsg.StatusTypeItemStaged)); got != beforeStaged {
		t.Errorf("ITEM_STAGED events: got %d, want the pre-confirm %d", got, beforeStaged)
	}
	if got := len(stageSagas(t, e)); got != beforeSagas {
		t.Errorf("transfer_to_trade sagas: got %d, want the pre-confirm %d", got, beforeSagas)
	}
	if got := len(statusEvents[trademsg.ItemRefusedEventBody](t, e, trademsg.StatusTypeItemRefused)); got != 1 {
		t.Fatalf("ITEM_REFUSED events: got %d, want 1", got)
	}
}

// TestEveryPutItemRefusalPathAnswersTheClient sweeps the remaining refusal
// branches in one place.
//
// It exists because the defect this task fixes was not one missing emit but a
// CLASS of them: eleven branches that logged "Dropping." and returned nil. A
// per-branch test would have been written for whichever branch someone happened
// to think of, so this asserts the invariant itself — no branch of PUT_ITEM
// leaves the client without an answer.
func TestEveryPutItemRefusalPathAnswersTheClient(t *testing.T) {
	for name, put := range map[string]func(p *ProcessorImpl) error{
		"trade slot out of range": func(p *ProcessorImpl) error {
			return p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 1, 0)
		},
		"trade slot already occupied": func(p *ProcessorImpl) error {
			if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 1, 1); err != nil {
				return err
			}
			return p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 1, 1)
		},
		"unstageable inventory byte": func(p *ProcessorImpl) error {
			return p.PutItem(uuid.New(), 100, 200, stagingSourceSlot, 1, 1)
		},
		"quantity zero": func(p *ProcessorImpl) error {
			return p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 0, 1)
		},
		"quantity above the stack": func(p *ProcessorImpl) error {
			return p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5_000, 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			p, e := testOpenRoom(t)
			if err := put(p); err != nil {
				t.Fatalf("put item: %v", err)
			}
			if got := len(statusEvents[trademsg.ItemRefusedEventBody](t, e, trademsg.StatusTypeItemRefused)); got != 1 {
				t.Fatalf("ITEM_REFUSED events: got %d, want 1 — this branch leaves the client locked", got)
			}
		})
	}
}

// TestPutItemDropsWhenTheAssetCannotBeRead pins that an unreadable inventory is a
// refusal, not a stage of a zero-valued asset.
func TestPutItemDropsWhenTheAssetCannotBeRead(t *testing.T) {
	p, e := testOpenRoom(t)
	p.invp = &fakeInventory{err: errors.New("atlas-inventory unreachable")}
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 1, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeItemStaged)
	assertNoStageSubmitted(t, e)
	if got := len(stagedItemsOf(t, p, 100)); got != 0 {
		t.Errorf("staged items: got %d, want 0", got)
	}
}

// TestPutItemDropsAnEmptySlot pins that staging from a slot the character does
// not occupy is refused rather than escrowing a phantom asset.
func TestPutItemDropsAnEmptySlot(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), 42, 1, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	assertNoStageSubmitted(t, e)
	if got := len(stagedItemsOf(t, p, 100)); got != 0 {
		t.Errorf("staged items: got %d, want 0", got)
	}
}

// TestPutItemRejectsQuantityBeyondTheStack pins FR-3.2: a client may stage a
// partial stack, but never more than the slot holds, and the availability check
// nets off what this room already claimed from the SAME slot — two 60s out of a
// 100 stack must not both succeed.
func TestPutItemRejectsQuantityBeyondTheStack(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 60, 1); err != nil {
		t.Fatalf("first put item: %v", err)
	}
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 60, 2); err != nil {
		t.Fatalf("second put item: %v", err)
	}
	if got := len(stagedItemsOf(t, p, 100)); got != 1 {
		t.Errorf("staged items: got %d, want 1 — 120 of a 100 stack was accepted", got)
	}
	if got := len(stageSagas(t, e)); got != 1 {
		t.Errorf("transfer_to_trade sagas: got %d, want 1", got)
	}
}

// TestPutItemRejectsQuantityBeyondTheWiresInt16 pins the width the staging
// payload's source slot and the orchestrator's release step actually carry: a
// uint16 above math.MaxInt16 would wrap negative on the wire and be widened by
// atlas-inventory into a near-4-billion release from the player's own stack.
func TestPutItemRejectsQuantityBeyondTheWiresInt16(t *testing.T) {
	p, e := testOpenRoom(t)
	p.invp = &fakeInventory{assets: map[assetKey]inventorydata.Asset{
		{100, inventory.TypeValueUse, stagingSourceSlot}: inventorydata.NewAsset(stagingAssetId, stagingSourceSlot, stagingTemplateId, 65_000, 0),
	}}
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 40_000, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	assertNoStageSubmitted(t, e)
	if got := len(stagedItemsOf(t, p, 100)); got != 0 {
		t.Errorf("staged items: got %d, want 0", got)
	}
}

// TestPutItemRejectsZeroQuantity pins the other end of the same range: a zero
// transfer would move nothing and still occupy a dialog slot.
func TestPutItemRejectsZeroQuantity(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 0, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	assertNoStageSubmitted(t, e)
	if got := len(stagedItemsOf(t, p, 100)); got != 0 {
		t.Errorf("staged items: got %d, want 0", got)
	}
}

// TestPutItemDropsWithoutARoom pins that a PUT_ITEM from a character in no trade
// neither errors nor emits.
func TestPutItemDropsWithoutARoom(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 300, byte(inventory.TypeValueUse), stagingSourceSlot, 1, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeItemStaged)
	assertNoStageSubmitted(t, e)
}

// TestStagingIsFrozenAfterFirstConfirm pins FR-3.6 / design §3.2: from the
// moment the room leaves OPEN, both PUT_ITEM and ADD_MESO are rejected for BOTH
// sides. The reference client enforces this locally too, so a server-side
// rejection here means a modified client — no clientbound response.
func TestStagingIsFrozenAfterFirstConfirm(t *testing.T) {
	p, e := testOpenRoom(t)
	room, _ := p.RoomForCharacter(100)
	if _, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		return cur.WithParticipant(0, func(v Participant) Participant { return v.WithConfirmed(true) }), nil
	}); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	if err := p.PutItem(uuid.New(), 200, byte(inventory.TypeValueUse), stagingSourceSlot, 1, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	if err := p.AddMeso(uuid.New(), 200, 1000); err != nil {
		t.Fatalf("add meso: %v", err)
	}

	assertNoEventOfType(t, e, trademsg.StatusTypeItemStaged)
	assertNoEventOfType(t, e, trademsg.StatusTypeMesoStaged)
	assertNoStageSubmitted(t, e)
	assertNoSagaOfAction(t, e, sharedsaga.AwardMesos)
	if got := len(stagedItemsOf(t, p, 200)); got != 0 {
		t.Errorf("staged items: got %d, want 0", got)
	}
	if got := mesoStagedBy(t, p, 200); got != 0 {
		t.Errorf("mesoStaged: got %d, want 0", got)
	}
}

// --- the staging saga's terminal status ---------------------------------------

// TestStageSucceededClearsPendingAndAnnouncesOnce pins the confirming half of a
// stage: the escrow row exists, so the item stops being pending and BOTH dialogs
// finally hear about it — exactly once. A redelivered terminal status must not
// add a second copy of the item to either window, so it is still claimed (true)
// but announces nothing.
func TestStageSucceededClearsPendingAndAnnouncesOnce(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 3); err != nil {
		t.Fatalf("put item: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	i := pendingItemIn(t, p, 100, 3)
	escrowOf(t, p).accept(room.Id(), 100, i)

	claimed, err := p.StageSucceeded(uuid.New(), i.EscrowId())
	if err != nil {
		t.Fatalf("stage succeeded: %v", err)
	}
	if !claimed {
		t.Fatal("StageSucceeded did not claim a transaction that names a staged item")
	}
	if pendingItemIn(t, p, 100, 3).Pending() {
		t.Error("the staged item is still pending after its escrow row was confirmed")
	}

	staged := statusEvents[trademsg.ItemStagedEventBody](t, e, trademsg.StatusTypeItemStaged)
	if len(staged) != 1 {
		t.Fatalf("ITEM_STAGED events: got %d, want 1", len(staged))
	}
	if staged[0].Body.Position != 0 || staged[0].Body.TradeSlot != 3 || staged[0].Body.Snapshot.Quantity != 5 {
		t.Errorf("ITEM_STAGED body: got %+v, want position 0 trade slot 3 quantity 5", staged[0].Body)
	}
	// The snapshot must come from the ESCROW ROW, not from the room's staged
	// item: the room knows the trade slot and the quantity, but the asset that
	// backs them has already been deleted from its owner's compartment, and only
	// the custody row still carries its cash serial, expiry and pet identity.
	// This is the whole payload atlas-channel renders the trade frame from.
	if s := staged[0].Body.Snapshot; s.CashId != escrowCashId || s.Owner != escrowOwnerName || s.PetId != escrowPetId {
		t.Errorf("ITEM_STAGED snapshot: got %+v, want the escrow row's cashId %d owner %q petId %d", s, escrowCashId, escrowOwnerName, escrowPetId)
	}

	// The redelivery.
	claimed, err = p.StageSucceeded(uuid.New(), i.EscrowId())
	if err != nil {
		t.Fatalf("redelivered stage succeeded: %v", err)
	}
	if !claimed {
		t.Error("the redelivery was not claimed; it would fall through to the settlement path")
	}
	if got := len(statusEvents[trademsg.ItemStagedEventBody](t, e, trademsg.StatusTypeItemStaged)); got != 1 {
		t.Errorf("ITEM_STAGED events after the redelivery: got %d, want 1", got)
	}
	if got := len(stagedItemsOf(t, p, 100)); got != 1 {
		t.Errorf("staged items after the redelivery: got %d, want 1", got)
	}
}

// TestStageFailedFreesTheSlotAndRefusesTheStagingClientOnly pins design §5A.6.
// The refusal renders nothing — the empty slot is the whole player-visible
// feedback — but it has to ARRIVE: the client armed m_bExclRequestSent when it
// sent PUT_ITEM, and nothing else on this path clears it, so a silent drop
// wedges the dialog for the rest of the session.
//
// The counterparty gets nothing at all, because a pending item was never
// announced to them: there is nothing for their window to correct.
func TestStageFailedFreesTheSlotAndRefusesTheStagingClientOnly(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 3); err != nil {
		t.Fatalf("put item: %v", err)
	}
	i := pendingItemIn(t, p, 100, 3)

	claimed, err := p.StageFailed(uuid.New(), i.EscrowId(), "RELEASE_FAILED")
	if err != nil {
		t.Fatalf("stage failed: %v", err)
	}
	if !claimed {
		t.Fatal("StageFailed did not claim a transaction that names a pending stage")
	}

	if got := len(stagedItemsOf(t, p, 100)); got != 0 {
		t.Errorf("staged items: got %d, want 0 — the refused stage kept its dialog slot", got)
	}
	refused := statusEvents[trademsg.ItemRefusedEventBody](t, e, trademsg.StatusTypeItemRefused)
	if len(refused) != 1 {
		t.Fatalf("ITEM_REFUSED events: got %d, want 1", len(refused))
	}
	if refused[0].CharacterId != 100 {
		t.Errorf("ITEM_REFUSED addressee: got %d, want the staging character 100", refused[0].CharacterId)
	}
	if refused[0].Body.Position != 0 || refused[0].Body.TradeSlot != 3 {
		t.Errorf("ITEM_REFUSED body: got %+v, want position 0 trade slot 3", refused[0].Body)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeItemStaged)

	// And the freed slot is genuinely usable again.
	if err = p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), 2, 1, 3); err != nil {
		t.Fatalf("re-stage: %v", err)
	}
	if got := len(stagedItemsOf(t, p, 100)); got != 1 {
		t.Errorf("staged items after re-staging into the freed slot: got %d, want 1", got)
	}
}

// TestStageFailedLeavesAConfirmedItemToTheSagasCompensation pins the arm that
// tells the two failure shapes apart. A saga that fails AFTER its accept was
// acked has the item genuinely in escrow, and the orchestrator's reverse walk
// owns returning it; dropping the dialog slot here would desynchronise the
// dialog from the custody store and settle the trade short.
func TestStageFailedLeavesAConfirmedItemToTheSagasCompensation(t *testing.T) {
	p, e := testOpenRoom(t)
	escrowId := stageOne(t, p, 100, stagingSourceSlot, 5, 3)

	claimed, err := p.StageFailed(uuid.New(), escrowId, "LATE_FAILURE")
	if err != nil {
		t.Fatalf("stage failed: %v", err)
	}
	if !claimed {
		t.Error("a late failure for a confirmed stage was not claimed; it would fall through to the settlement path")
	}
	if got := len(stagedItemsOf(t, p, 100)); got != 1 {
		t.Errorf("staged items: got %d, want the confirmed 1 left in place", got)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeItemRefused)
}

// TestStageSucceededForAnUnknownTransactionIsNotClaimed pins the routing
// contract the saga status consumer depends on. It tries StageSucceeded first
// and falls through to the meso stake and then to the settlement, so a
// transaction id that names neither a staged item NOR an escrow row must report
// false — claiming it would swallow every settlement's terminal status.
func TestStageSucceededForAnUnknownTransactionIsNotClaimed(t *testing.T) {
	p, e := testOpenRoom(t)
	stageOne(t, p, 100, stagingSourceSlot, 5, 3)

	claimed, err := p.StageSucceeded(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("stage succeeded: %v", err)
	}
	if claimed {
		t.Error("an unknown transaction was claimed as a stage; settlement statuses would never be routed")
	}
	claimed, err = p.StageFailed(uuid.New(), uuid.New(), "UNKNOWN")
	if err != nil {
		t.Fatalf("stage failed: %v", err)
	}
	if claimed {
		t.Error("an unknown transaction was claimed as a failed stage")
	}
	assertNoUnwindSubmitted(t, e)
}

// TestStageSucceededForAnOrphanedEscrowRowReturnsTheItem pins the leak that
// escrow-at-staging would otherwise create. A teardown unwinds what is escrowed
// AT THAT MOMENT, so a stage still in flight has no row for it to find; the row
// then appears seconds later with no dialog slot and no trade referencing it.
// Without this return the player's item sits in the custody table forever, with
// no error anywhere.
func TestStageSucceededForAnOrphanedEscrowRowReturnsTheItem(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 3); err != nil {
		t.Fatalf("put item: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	i := pendingItemIn(t, p, 100, 3)

	// The room dies while the stage is in flight, and the row lands afterwards.
	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	escrowOf(t, p).accept(room.Id(), 100, i)

	claimed, err := p.StageSucceeded(uuid.New(), i.EscrowId())
	if err != nil {
		t.Fatalf("stage succeeded: %v", err)
	}
	if !claimed {
		t.Fatal("an orphaned escrow row was not claimed; the item would be stranded in custody")
	}

	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 {
		t.Fatalf("trade_unwind sagas: got %d, want 1 — the teardown itself had nothing escrowed to unwind", len(unwinds))
	}
	payload := unwindPayloadOf(t, unwinds[0])
	if len(payload.Items) != 1 || len(payload.Mesos) != 0 {
		t.Fatalf("unwind payload: got %d items and %d meso refunds, want exactly the one orphaned item", len(payload.Items), len(payload.Mesos))
	}
	if payload.Items[0].OwnerId != 100 {
		t.Errorf("unwind ownerId: got %d, want the staging character 100", payload.Items[0].OwnerId)
	}
	if payload.Items[0].Item.EscrowId != i.EscrowId() {
		t.Errorf("unwind escrowId: got %s, want %s", payload.Items[0].Item.EscrowId, i.EscrowId())
	}
}

// --- ADD_MESO ----------------------------------------------------------------

// TestAddMesoStagesTheDeltaAgainstEscrowNotTheRoom pins design §5A.5's
// arithmetic. Mode 16 ASSIGNS, so what moves is the difference between the
// requested total and what is already in custody — and it is the custody row,
// not the room, that is authoritative: the room's mesoStaged only advances once
// a stake settles, so deriving the delta from it would re-debit a stake that is
// still in flight.
//
// The sign is the other half: a negative award_mesos debits the staking player.
func TestAddMesoStagesTheDeltaAgainstEscrowNotTheRoom(t *testing.T) {
	for name, tc := range map[string]struct {
		escrowed int64
		request  int32
		want     int32
	}{
		"raising the stake debits the difference":   {escrowed: 1_000_000, request: 2_000_000, want: -1_000_000},
		"lowering the stake refunds the difference": {escrowed: 1_000_000, request: 400_000, want: 600_000},
		"a first stake debits the whole amount":     {escrowed: 0, request: 750_000, want: -750_000},
	} {
		t.Run(name, func(t *testing.T) {
			p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
			room, _ := p.RoomForCharacter(100)
			if tc.escrowed > 0 {
				commitEscrowedMeso(t, p, room.Id(), 100, tc.escrowed)
			}

			if err := p.AddMeso(uuid.New(), 100, tc.request); err != nil {
				t.Fatalf("add meso: %v", err)
			}

			sagas := sagasWithAction(t, e, sharedsaga.AwardMesos)
			if len(sagas) != 1 {
				t.Fatalf("award_mesos sagas: got %d, want 1", len(sagas))
			}
			payload := awardMesosPayloadOf(t, sagas[0])
			if payload.Amount != tc.want {
				t.Errorf("award_mesos amount: got %d, want %d", payload.Amount, tc.want)
			}
			if payload.CharacterId != 100 || payload.ActorId != 100 || payload.ActorType != mesoActorType {
				t.Errorf("award_mesos actor: got character %d actor %d/%s, want 100 / 100 / %s", payload.CharacterId, payload.ActorId, payload.ActorType, mesoActorType)
			}

			// Nothing is announced until the debit lands: the counterparty's
			// dialog must never render a stake the debit then failed to take.
			assertNoEventOfType(t, e, trademsg.StatusTypeMesoStaged)
			if got := mesoStagedBy(t, p, 100); got != 0 {
				t.Errorf("mesoStaged: got %d, want 0 until the stake settles", got)
			}
			if got := pendingMesoOf(t, p, 100); got != uint32(tc.request) {
				t.Errorf("pending meso: got %d, want the requested %d", got, tc.request)
			}
		})
	}
}

// pendingMesoOf returns the in-flight stake amount recorded on the character's
// participant.
func pendingMesoOf(t *testing.T, p *ProcessorImpl, characterId character.Id) uint32 {
	t.Helper()
	room, ok := p.RoomForCharacter(characterId)
	if !ok {
		t.Fatalf("character %d holds no room", characterId)
	}
	pt, _ := room.ParticipantFor(characterId)
	return pt.PendingMesoAmount()
}

// TestAddMesoOfTheEscrowedAmountReEchoesWithoutMovingMeso pins the delta == 0
// arm. Nothing has to move, but the client armed its exclusive-request lock the
// moment it sent PUT_MONEY, so the re-echo is not optional — MESO_STAGED is what
// carries the unlock (design §5A.6).
func TestAddMesoOfTheEscrowedAmountReEchoesWithoutMovingMeso(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	room, _ := p.RoomForCharacter(100)
	commitEscrowedMeso(t, p, room.Id(), 100, 1_000_000)

	if err := p.AddMeso(uuid.New(), 100, 1_000_000); err != nil {
		t.Fatalf("add meso: %v", err)
	}

	assertNoSagaOfAction(t, e, sharedsaga.AwardMesos)
	staged := statusEvents[trademsg.MesoStagedEventBody](t, e, trademsg.StatusTypeMesoStaged)
	if len(staged) != 1 {
		t.Fatalf("MESO_STAGED events: got %d, want 1", len(staged))
	}
	if staged[0].Body.Amount != 1_000_000 || staged[0].Body.Position != 0 {
		t.Errorf("MESO_STAGED body: got %+v, want position 0 amount 1000000", staged[0].Body)
	}
}

// TestMesoStageSucceededCommitsTheStakeAndAnnouncesIt pins the committing half:
// the durable stake row is resolved by compare-and-set FIRST, and only then does
// the room follow and both dialogs hear the new total.
func TestMesoStageSucceededCommitsTheStakeAndAnnouncesIt(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	if err := p.AddMeso(uuid.New(), 100, 1_000_000); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	stakeId := sagasWithAction(t, e, sharedsaga.AwardMesos)[0].TransactionId

	claimed, err := p.MesoStageSucceeded(uuid.New(), stakeId)
	if err != nil {
		t.Fatalf("meso stage succeeded: %v", err)
	}
	if !claimed {
		t.Fatal("MesoStageSucceeded did not claim its own stake")
	}
	if got := mesoStagedBy(t, p, 100); got != 1_000_000 {
		t.Errorf("mesoStaged: got %d, want 1000000", got)
	}
	staged := statusEvents[trademsg.MesoStagedEventBody](t, e, trademsg.StatusTypeMesoStaged)
	if len(staged) != 1 {
		t.Fatalf("MESO_STAGED events: got %d, want 1", len(staged))
	}
	if staged[0].Body.Amount != 1_000_000 {
		t.Errorf("MESO_STAGED amount: got %d, want 1000000", staged[0].Body.Amount)
	}

	// The redelivery is inert. It reports NOT claimed, and deliberately so: the
	// commit cleared pendingStakeId in the same statement that moved the amount,
	// so the durable row no longer answers to this stake id at all. The saga
	// status consumer then falls through to SettlementSucceeded, which finds no
	// settlement record for it and drops — the same harmless end, reached by the
	// same compare-and-set that makes a superseded stake inert.
	claimed, err = p.MesoStageSucceeded(uuid.New(), stakeId)
	if err != nil {
		t.Fatalf("redelivered meso stage succeeded: %v", err)
	}
	if claimed {
		t.Error("a redelivered stake status was claimed a second time; the durable compare-and-set did not consume it")
	}
	if got := len(statusEvents[trademsg.MesoStagedEventBody](t, e, trademsg.StatusTypeMesoStaged)); got != 1 {
		t.Errorf("MESO_STAGED events after the redelivery: got %d, want 1", got)
	}
	if got := mesoStagedBy(t, p, 100); got != 1_000_000 {
		t.Errorf("mesoStaged after the redelivery: got %d, want 1000000", got)
	}
}

// TestMesoStageFailedReEchoesTheLastValidAmount pins the refusal. Mode 16 is an
// assignment, so the staking client has ALREADY moved its own view to the amount
// it typed; only an authoritative re-echo of the last amount that actually
// settled will snap it back.
func TestMesoStageFailedReEchoesTheLastValidAmount(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	room, _ := p.RoomForCharacter(100)

	// A stake that already settled: 400000 is the last valid amount.
	if err := p.AddMeso(uuid.New(), 100, 400_000); err != nil {
		t.Fatalf("first add meso: %v", err)
	}
	first := sagasWithAction(t, e, sharedsaga.AwardMesos)[0].TransactionId
	if _, err := p.MesoStageSucceeded(uuid.New(), first); err != nil {
		t.Fatalf("first stake succeeded: %v", err)
	}
	commitEscrowedMeso(t, p, room.Id(), 100, 400_000)

	// The second stake is debited and then fails.
	if err := p.AddMeso(uuid.New(), 100, 2_000_000); err != nil {
		t.Fatalf("second add meso: %v", err)
	}
	stakes := sagasWithAction(t, e, sharedsaga.AwardMesos)
	if len(stakes) != 2 {
		t.Fatalf("award_mesos sagas: got %d, want 2", len(stakes))
	}
	claimed, err := p.MesoStageFailed(uuid.New(), stakes[1].TransactionId, "NOT_ENOUGH_MESOS")
	if err != nil {
		t.Fatalf("meso stage failed: %v", err)
	}
	if !claimed {
		t.Fatal("MesoStageFailed did not claim its own stake")
	}

	refused := statusEvents[trademsg.MesoRefusedEventBody](t, e, trademsg.StatusTypeMesoRefused)
	if len(refused) != 1 {
		t.Fatalf("MESO_REFUSED events: got %d, want 1", len(refused))
	}
	if refused[0].Body.LastValidAmount != 400_000 {
		t.Errorf("lastValidAmount: got %d, want the last stake that actually settled (400000)", refused[0].Body.LastValidAmount)
	}
	if got := mesoStagedBy(t, p, 100); got != 400_000 {
		t.Errorf("mesoStaged: got %d, want the last valid 400000", got)
	}
}

// commitEscrowedMeso puts a COMMITTED escrow total in front of both readers the
// meso path uses: the fake custody store the processor reads deltas and unwinds
// from, and the REAL escrow row the stake bookkeeping arms and resolves against.
// Production has one store; the harness fakes only the reads, so a test that set
// just one of them would be describing a state that cannot occur.
func commitEscrowedMeso(t *testing.T, p *ProcessorImpl, roomId uuid.UUID, ownerId character.Id, amount int64) {
	t.Helper()
	escrowOf(t, p).setMeso(roomId, ownerId, amount)
	if err := escrow.UpsertMeso(p.db, p.t)(roomId, ownerId, amount); err != nil {
		t.Fatalf("seed committed escrow meso: %v", err)
	}
}

// TestAnOrphanedStakeRefundsOnlyTheDeltaTheSagaMoved drives the meso-minting
// sequence end to end, no restart required:
//
//  1. character 100 has 1000000 committed in escrow;
//  2. they retype the box to 1500000, so an award_mesos of -500000 goes out and
//     they have now been debited 1500000 in total;
//  3. the counterparty leaves. The teardown refunds the COMMITTED 1000000 and
//     zeroes the row's Amount while deliberately leaving the stake armed;
//  4. the stake's saga completes with no room left to apply it to.
//
// The refund owed at step 4 is the 500000 the saga actually moved. Deriving it
// from the row's Amount at that point instead — which step 3 zeroed — refunds
// the whole 1500000, handing back 2500000 against a debit of 1500000.
func TestAnOrphanedStakeRefundsOnlyTheDeltaTheSagaMoved(t *testing.T) {
	const (
		committed = uint32(1_000_000)
		retyped   = int32(1_500_000)
		moved     = uint32(500_000)
	)

	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	room, _ := p.RoomForCharacter(100)
	commitEscrowedMeso(t, p, room.Id(), 100, int64(committed))

	if err := p.AddMeso(uuid.New(), 100, retyped); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	stakes := sagasWithAction(t, e, sharedsaga.AwardMesos)
	if len(stakes) != 1 {
		t.Fatalf("award_mesos sagas: got %d, want 1", len(stakes))
	}
	if got := awardMesosPayloadOf(t, stakes[0]).Amount; got != -int32(moved) {
		t.Fatalf("the stake debited %d, want -%d — the rest of this test is about refunding exactly what it moved", got, moved)
	}
	stakeId := stakes[0].TransactionId

	// The counterparty walks out while the debit is still in flight.
	if err := p.TeardownCharacter(uuid.New(), 200, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 {
		t.Fatalf("trade_unwind sagas after the teardown: got %d, want 1", len(unwinds))
	}
	teardown := unwindPayloadOf(t, unwinds[0])
	if len(teardown.Mesos) != 1 || teardown.Mesos[0].Amount != committed {
		t.Fatalf("the teardown refunded %+v, want the committed %d", teardown.Mesos, committed)
	}

	claimed, err := p.MesoStageSucceeded(uuid.New(), stakeId)
	if err != nil {
		t.Fatalf("meso stage succeeded: %v", err)
	}
	if !claimed {
		t.Fatal("the orphaned stake's terminal status was not claimed; nothing would ever refund it")
	}

	unwinds = unwindSagas(t, e)
	if len(unwinds) != 2 {
		t.Fatalf("trade_unwind sagas after the stake settled: got %d, want 2 (the teardown's and the orphan refund)", len(unwinds))
	}
	refund := unwindPayloadOf(t, unwinds[1])
	if len(refund.Mesos) != 1 || len(refund.Items) != 0 {
		t.Fatalf("orphan refund payload: got %+v, want exactly one meso leg", refund)
	}
	leg := refund.Mesos[0]
	if leg.CharacterId != 100 {
		t.Errorf("orphan refund addressed to character %d, want 100", leg.CharacterId)
	}
	if leg.Amount != moved {
		t.Errorf("orphan refund: got %d, want the %d the saga moved — the player was debited %d and has now been handed back %d, minting %d",
			leg.Amount, moved, uint32(retyped), committed+leg.Amount, int64(committed)+int64(leg.Amount)-int64(retyped))
	}
}

// mesoRowsIn counts every escrow meso row in the database, across rooms and
// tenants — the figure AllMesos hands the boot sweep, and therefore the one that
// must not grow with lifetime trade volume.
func mesoRowsIn(t *testing.T, db *gorm.DB) int {
	t.Helper()
	rows, err := escrow.AllMesos(db)
	if err != nil {
		t.Fatalf("read escrow mesos: %v", err)
	}
	return len(rows)
}

// TestACompletedMesoStagingCycleLeavesNoEscrowRow pins that custody rows are
// RETIRED, not merely emptied.
//
// A meso row exists to record that the service is holding a player's meso. The
// stage creates one, the teardown refunds what it holds — and at that point it
// records nothing anybody can act on. Left behind it is read again by every
// future boot: AllMesos is an unfiltered scan of the whole table, so a row per
// meso trade ever made is a startup cost that grows without bound.
func TestACompletedMesoStagingCycleLeavesNoEscrowRow(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	room, _ := p.RoomForCharacter(100)

	if got := mesoRowsIn(t, p.db); got != 0 {
		t.Fatalf("escrow meso rows before the trade: got %d, want 0", got)
	}

	if err := p.AddMeso(uuid.New(), 100, 1_000_000); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	stakes := sagasWithAction(t, e, sharedsaga.AwardMesos)
	if len(stakes) != 1 {
		t.Fatalf("award_mesos sagas: got %d, want 1", len(stakes))
	}
	if _, err := p.MesoStageSucceeded(uuid.New(), stakes[0].TransactionId); err != nil {
		t.Fatalf("meso stage succeeded: %v", err)
	}
	// The committed total, mirrored onto the fake the teardown reads its unwind
	// payload from. Production has one store; the harness fakes only the reads.
	commitEscrowedMeso(t, p, room.Id(), 100, 1_000_000)

	if got := mesoRowsIn(t, p.db); got != 1 {
		t.Fatalf("escrow meso rows with a live stage: got %d, want the one row holding the escrow", got)
	}

	if err := p.TeardownCharacter(uuid.New(), 200, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 || len(unwindPayloadOf(t, unwinds[0]).Mesos) != 1 {
		t.Fatalf("trade_unwind sagas: got %+v, want one refunding the staged meso", unwinds)
	}

	if got := mesoRowsIn(t, p.db); got != 0 {
		t.Errorf("escrow meso rows after the trade: got %d, want 0 — a resolved row survived its own trade, and one per meso trade ever made is read by every later boot", got)
	}
}

// TestATeardownKeepsARefundedRowUntilItsStakeResolves pins the exception the
// conditional delete is built around, and then the delete it eventually allows.
//
// A teardown that zeroes a row whose stake is STILL IN FLIGHT must leave the row
// standing: the terminal status resolves against it through the stake's own row,
// and deleting it strands a debit the player has already been charged. Once that
// status arrives and its refund is submitted the row records nothing, and only
// then does it go.
func TestATeardownKeepsARefundedRowUntilItsStakeResolves(t *testing.T) {
	const (
		committed = uint32(1_000_000)
		retyped   = int32(1_500_000)
		moved     = uint32(500_000)
	)

	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	room, _ := p.RoomForCharacter(100)
	commitEscrowedMeso(t, p, room.Id(), 100, int64(committed))

	if err := p.AddMeso(uuid.New(), 100, retyped); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	stakes := sagasWithAction(t, e, sharedsaga.AwardMesos)
	if len(stakes) != 1 {
		t.Fatalf("award_mesos sagas: got %d, want 1", len(stakes))
	}
	stakeId := stakes[0].TransactionId

	if err := p.TeardownCharacter(uuid.New(), 200, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}

	row, ok := mesoRow(t, p.db, p.t, room.Id(), 100)
	if !ok {
		t.Fatal("the refunded row was deleted while its stake was still in flight; the debit already taken from the player has nothing left to resolve against")
	}
	if row.Amount() != 0 {
		t.Errorf("refunded row amount: got %d, want 0", row.Amount())
	}
	stakeRows, err := escrow.MesoStakesByOwner(p.db, p.t.Id())(room.Id(), 100)
	if err != nil {
		t.Fatalf("MesoStakesByOwner: %v", err)
	}
	if len(stakeRows) != 1 || stakeRows[0].Id() != stakeId {
		t.Errorf("outstanding stakes: got %+v, want just the armed %s", stakeRows, stakeId)
	}

	if _, err := p.MesoStageSucceeded(uuid.New(), stakeId); err != nil {
		t.Fatalf("meso stage succeeded: %v", err)
	}

	// The stake still resolves correctly: the delta the saga moved comes back on
	// top of the teardown's refund of the committed part.
	unwinds := unwindSagas(t, e)
	if len(unwinds) != 2 {
		t.Fatalf("trade_unwind sagas: got %d, want 2 (the teardown's and the orphan refund)", len(unwinds))
	}
	refund := unwindPayloadOf(t, unwinds[1]).Mesos
	if len(refund) != 1 || refund[0].CharacterId != 100 || refund[0].Amount != moved {
		t.Fatalf("orphan refund: got %+v, want %d back to character 100", refund, moved)
	}

	if got := mesoRowsIn(t, p.db); got != 0 {
		t.Errorf("escrow meso rows once the stake resolved: got %d, want 0 — nothing is escrowed and no stake is pending", got)
	}
}

// TestAnOrphanedStakeLeavesNothingForTheNextBootToRefund is the double refund the
// retirement prevents, driven end to end through the sequence that produces it:
//
//  1. character 100 has 1000000 committed in escrow;
//  2. they retype the box to 1500000, so an award_mesos of -500000 goes out and
//     they have now been debited 1500000 in total;
//  3. the counterparty leaves. The teardown refunds the committed 1000000 and
//     zeroes the row, deliberately leaving the stake armed;
//  4. the stake's saga completes with no room left. CommitMesoStake assigns its
//     absolute total — 1500000 — into a row that holds nothing, and the delta of
//     500000 is refunded. The player is now whole: 1000000 + 500000 = 1500000.
//
// The stale 1500000 left on the row is what the next boot reads. ReconcileEscrow
// cannot tell a stranded asset from bookkeeping nobody cleared, so it refunds the
// figure a second time — 3000000 handed back against a debit of 1500000.
func TestAnOrphanedStakeLeavesNothingForTheNextBootToRefund(t *testing.T) {
	const (
		committed = uint32(1_000_000)
		retyped   = int32(1_500_000)
		debited   = uint32(1_500_000)
	)

	// The boot sweep builds its own processor and resolves each owner's field
	// over REST, so the location service has to be real for it — unlike the
	// staging processor, which is handed a fake. Without it the sweep would fail
	// to locate character 100 and submit nothing for reasons that have nothing to
	// do with the row.
	serveLocations(t, map[character.Id][2]byte{100: {1, 1}})

	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	room, _ := p.RoomForCharacter(100)
	commitEscrowedMeso(t, p, room.Id(), 100, int64(committed))

	if err := p.AddMeso(uuid.New(), 100, retyped); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	stakes := sagasWithAction(t, e, sharedsaga.AwardMesos)
	if len(stakes) != 1 {
		t.Fatalf("award_mesos sagas: got %d, want 1", len(stakes))
	}
	if err := p.TeardownCharacter(uuid.New(), 200, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	if _, err := p.MesoStageSucceeded(uuid.New(), stakes[0].TransactionId); err != nil {
		t.Fatalf("meso stage succeeded: %v", err)
	}
	beforeBoot := len(unwindSagas(t, e))

	// The restart.
	if err := ReconcileEscrow(reconcileLogger(), context.Background(), p.db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}

	unwinds := unwindSagas(t, e)
	if len(unwinds) != beforeBoot {
		swept := unwindPayloadOf(t, unwinds[len(unwinds)-1]).Mesos
		t.Errorf("the boot sweep submitted %d further unwind(s) for a room whose escrow is settled: %+v", len(unwinds)-beforeBoot, swept)
	}

	var refunded uint32
	for _, u := range unwinds {
		for _, m := range unwindPayloadOf(t, u).Mesos {
			if m.CharacterId == 100 {
				refunded += m.Amount
			}
		}
	}
	if refunded != debited {
		t.Errorf("character 100 was debited %d and handed back %d, minting %d", debited, refunded, int64(refunded)-int64(debited))
	}
}

// TestAnOrphanedStakeThatLoweredTheBoxRefundsNothing is the other sign. A retype
// DOWNWARD submits a credit, so the player is already whole the moment it
// completes; a refund on top of it would mint the difference a second way.
func TestAnOrphanedStakeThatLoweredTheBoxRefundsNothing(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	room, _ := p.RoomForCharacter(100)
	commitEscrowedMeso(t, p, room.Id(), 100, 1_000_000)

	if err := p.AddMeso(uuid.New(), 100, 400_000); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	stakes := sagasWithAction(t, e, sharedsaga.AwardMesos)
	if len(stakes) != 1 {
		t.Fatalf("award_mesos sagas: got %d, want 1", len(stakes))
	}
	if got := awardMesosPayloadOf(t, stakes[0]).Amount; got != 600_000 {
		t.Fatalf("the stake moved %d, want a credit of 600000", got)
	}

	if err := p.TeardownCharacter(uuid.New(), 200, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	if _, err := p.MesoStageSucceeded(uuid.New(), stakes[0].TransactionId); err != nil {
		t.Fatalf("meso stage succeeded: %v", err)
	}

	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 {
		t.Fatalf("trade_unwind sagas: got %d, want only the teardown's 1 — a lowering stake is owed nothing", len(unwinds))
	}
}

// TestAnOrphanedFirstStakeRefundsItsWholeAmount guards the fix from over-reach.
// With nothing committed yet there is no teardown refund to net against, so the
// whole stake IS the delta and all of it must come back.
func TestAnOrphanedFirstStakeRefundsItsWholeAmount(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)

	if err := p.AddMeso(uuid.New(), 100, 750_000); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	stakes := sagasWithAction(t, e, sharedsaga.AwardMesos)
	if len(stakes) != 1 {
		t.Fatalf("award_mesos sagas: got %d, want 1", len(stakes))
	}

	// Nothing is committed, so this teardown submits no unwind of its own and
	// never touches the row — the stake is orphaned with its arm-time state
	// completely intact.
	if err := p.TeardownCharacter(uuid.New(), 200, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	assertNoUnwindSubmitted(t, e)

	if _, err := p.MesoStageSucceeded(uuid.New(), stakes[0].TransactionId); err != nil {
		t.Fatalf("meso stage succeeded: %v", err)
	}

	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 {
		t.Fatalf("trade_unwind sagas: got %d, want 1", len(unwinds))
	}
	refund := unwindPayloadOf(t, unwinds[0])
	if len(refund.Mesos) != 1 || refund.Mesos[0].CharacterId != 100 || refund.Mesos[0].Amount != 750_000 {
		t.Errorf("orphan refund: got %+v, want the whole 750000 back to character 100", refund.Mesos)
	}
}

// TestAddMesoRefusedReEchoesTheLastValidAmount pins FR-4.8 / design §4.2 on the
// VALIDATION path: an amount the character does not hold is refused before any
// debit is attempted, and the re-echo carries what is still legitimately staged.
func TestAddMesoRefusedReEchoesTheLastValidAmount(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	room, _ := p.RoomForCharacter(100)
	if err := p.AddMeso(uuid.New(), 100, 1_000_000); err != nil {
		t.Fatalf("first add meso: %v", err)
	}
	first := sagasWithAction(t, e, sharedsaga.AwardMesos)[0].TransactionId
	if _, err := p.MesoStageSucceeded(uuid.New(), first); err != nil {
		t.Fatalf("first stake succeeded: %v", err)
	}
	commitEscrowedMeso(t, p, room.Id(), 100, 1_000_000)

	if err := p.AddMeso(uuid.New(), 100, 9_999_999); err != nil { // more than the character holds
		t.Fatalf("second add meso: %v", err)
	}

	refused := statusEvents[trademsg.MesoRefusedEventBody](t, e, trademsg.StatusTypeMesoRefused)
	if len(refused) != 1 {
		t.Fatalf("MESO_REFUSED events: got %d, want 1", len(refused))
	}
	if refused[0].Body.LastValidAmount != 1_000_000 {
		t.Errorf("lastValidAmount: got %d, want 1000000", refused[0].Body.LastValidAmount)
	}
	if got := mesoStagedBy(t, p, 100); got != 1_000_000 {
		t.Errorf("mesoStaged: got %d, want the last valid 1000000", got)
	}
	if got := len(sagasWithAction(t, e, sharedsaga.AwardMesos)); got != 1 {
		t.Errorf("award_mesos sagas: got %d, want only the first, valid one", got)
	}
}

// TestAddMesoRefusesWhenTheCounterpartyWouldOverflow pins the delivery-side half
// of FR-4.8: the settlement saga refuses a payload above MaxInt32, so a stage
// that would push the receiver past the ceiling is refused at stage time rather
// than surfacing as LEAVE 8 after both sides confirmed.
func TestAddMesoRefusesWhenTheCounterpartyWouldOverflow(t *testing.T) {
	p, e := newStagingProcessor(t, configuration.DefaultConfig(),
		testCharacter{Id: 100, Name: "Owner", Hp: 100, Level: 30, Meso: 2_000_000_000},
		testCharacter{Id: 200, Name: "Guest", Hp: 100, Level: 30, Meso: 2_000_000_000},
	)
	if err := p.AddMeso(uuid.New(), 100, 1_000_000_000); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	if got := len(statusEvents[trademsg.MesoRefusedEventBody](t, e, trademsg.StatusTypeMesoRefused)); got != 1 {
		t.Errorf("MESO_REFUSED events: got %d, want 1", got)
	}
	assertNoSagaOfAction(t, e, sharedsaga.AwardMesos)
	if got := mesoStagedBy(t, p, 100); got != 0 {
		t.Errorf("mesoStaged: got %d, want 0", got)
	}
}

// TestAddMesoAllowsADeliverableAmountToTheCounterparty guards the overflow rule
// against over-reach: the very same holdings accept a stage that fits.
func TestAddMesoAllowsADeliverableAmountToTheCounterparty(t *testing.T) {
	p, e := newStagingProcessor(t, configuration.DefaultConfig(),
		testCharacter{Id: 100, Name: "Owner", Hp: 100, Level: 30, Meso: 2_000_000_000},
		testCharacter{Id: 200, Name: "Guest", Hp: 100, Level: 30, Meso: 100_000_000},
	)
	if err := p.AddMeso(uuid.New(), 100, 1_000_000_000); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeMesoRefused)
	stakes := sagasWithAction(t, e, sharedsaga.AwardMesos)
	if len(stakes) != 1 {
		t.Fatalf("award_mesos sagas: got %d, want 1", len(stakes))
	}
	if _, err := p.MesoStageSucceeded(uuid.New(), stakes[0].TransactionId); err != nil {
		t.Fatalf("stake succeeded: %v", err)
	}
	if got := mesoStagedBy(t, p, 100); got != 1_000_000_000 {
		t.Errorf("mesoStaged: got %d, want 1000000000", got)
	}
}

// TestAddMesoRejectsNegative pins the NFR: every client value is untrusted. A
// negative cannot come from the input box, so no meso moves.
//
// It DOES get the re-echo, though, and that is the point of the test since the
// escrow amendment. PutMoney armed m_bExclRequestSent on send, and only a server
// packet clears it, so returning nothing here would leave a forged packet able
// to wedge its own dialog for the rest of the session (design §5A.6). This test
// previously asserted the opposite — that nothing was emitted — which is how the
// wedge reached a live client.
func TestAddMesoRejectsNegative(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	if err := p.AddMeso(uuid.New(), 100, -1); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeMesoStaged)
	assertNoSagaOfAction(t, e, sharedsaga.AwardMesos)

	refusals := statusEvents[trademsg.MesoRefusedEventBody](t, e, trademsg.StatusTypeMesoRefused)
	if len(refusals) != 1 {
		t.Fatalf("MESO_REFUSED events: got %d, want 1", len(refusals))
	}
	if refusals[0].CharacterId != 100 {
		t.Errorf("refusal addressed to character %d, want 100", refusals[0].CharacterId)
	}
	if refusals[0].Body.LastValidAmount != 0 {
		t.Errorf("lastValidAmount: got %d, want 0 — the re-echo must carry what is actually staked", refusals[0].Body.LastValidAmount)
	}
	if got := mesoStagedBy(t, p, 100); got != 0 {
		t.Errorf("mesoStaged: got %d, want 0", got)
	}
}

// TestAddMesoRefusesWhenTheCharacterCannotBeRead pins that an unreadable meso
// balance is a refusal, not a stage against the client's own claim.
func TestAddMesoRefusesWhenTheCharacterCannotBeRead(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	p.cp = &fakeCharacters{err: errors.New("atlas-character unreachable")}
	if err := p.AddMeso(uuid.New(), 100, 1_000); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	if got := len(statusEvents[trademsg.MesoRefusedEventBody](t, e, trademsg.StatusTypeMesoRefused)); got != 1 {
		t.Errorf("MESO_REFUSED events: got %d, want 1", got)
	}
	assertNoSagaOfAction(t, e, sharedsaga.AwardMesos)
	if got := mesoStagedBy(t, p, 100); got != 0 {
		t.Errorf("mesoStaged: got %d, want 0", got)
	}
}

// TestAddMesoRefusesWhenTheEscrowedTotalCannotBeRead pins the read the delta
// arithmetic depends on. An unreadable custody row makes the delta unknowable,
// and guessing zero would re-debit the whole requested amount on top of what the
// player has already staked.
func TestAddMesoRefusesWhenTheEscrowedTotalCannotBeRead(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	// Injected at the REAL store, not into the fake. The netting read
	// deliberately bypasses the fake-able seam so that it cannot diverge from
	// the arms, so an error planted in the fake would never reach it and this
	// test would pass while guarding nothing.
	databasetest.FailReadsOn(t, p.db, "trade_escrow_mesos")

	if err := p.AddMeso(uuid.New(), 100, 1_000); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	if got := len(statusEvents[trademsg.MesoRefusedEventBody](t, e, trademsg.StatusTypeMesoRefused)); got != 1 {
		t.Errorf("MESO_REFUSED events: got %d, want 1", got)
	}
	assertNoSagaOfAction(t, e, sharedsaga.AwardMesos)
}

// TestAddMesoDropsWithoutARoom pins that an ADD_MESO from a character in no
// trade neither errors nor emits.
func TestAddMesoDropsWithoutARoom(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.AddMeso(uuid.New(), 300, 1_000); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeMesoStaged)
	assertNoEventOfType(t, e, trademsg.StatusTypeMesoRefused)
}

// --- teardown and the unwind ---------------------------------------------------

// TestTeardownUnwindsEveryEscrowedItemAndMesoInOneSaga pins design §5A.8's
// return path, which replaces the reservation cancels the previous model owed.
// A staged asset has genuinely LEFT its owner's compartment, so a room that
// disappears without unwinding does not merely leave a lock behind — it keeps
// the player's item.
//
// It is ONE saga, not one per row: the whole room's custody goes back together,
// and the orchestrator's own reverse walk is what handles a partial failure.
func TestTeardownUnwindsEveryEscrowedItemAndMesoInOneSaga(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	room, _ := p.RoomForCharacter(100)

	ownerEscrow := stageOne(t, p, 100, stagingSourceSlot, 5, 1)
	visitorEscrow := stageOne(t, p, 200, 2, 7, 1)
	commitEscrowedMeso(t, p, room.Id(), 100, 250_000)

	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}

	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 {
		t.Fatalf("trade_unwind sagas: got %d, want exactly 1 for the whole room", len(unwinds))
	}
	if unwinds[0].SagaType != sharedsaga.TradeTransaction {
		t.Errorf("unwind sagaType: got %s, want %s", unwinds[0].SagaType, sharedsaga.TradeTransaction)
	}
	payload := unwindPayloadOf(t, unwinds[0])
	if len(payload.Items) != 2 {
		t.Fatalf("unwind items: got %d, want one per escrowed item (2)", len(payload.Items))
	}
	want := map[uuid.UUID]character.Id{ownerEscrow: 100, visitorEscrow: 200}
	for _, it := range payload.Items {
		owner, ok := want[it.Item.EscrowId]
		if !ok {
			t.Errorf("unwind returned escrow row %s, which was never staged", it.Item.EscrowId)
			continue
		}
		if it.OwnerId != owner {
			t.Errorf("unwind ownerId for %s: got %d, want %d", it.Item.EscrowId, it.OwnerId, owner)
		}
		s := it.Item.Snapshot
		if s.WeaponAttack != escrowStatWeaponAttack || s.Slots != escrowStatSlots || s.Owner != escrowOwnerName {
			t.Errorf("unwind item %s lost its equip stats: %+v", it.Item.EscrowId, s)
		}
		// The cash serial, expiry and pet id ride the same snapshot. They are
		// asserted separately because they were the fields the previous escrow
		// shape omitted, and losing them degrades the item rather than failing.
		if s.CashId != escrowCashId || !s.Expiration.Equal(escrowExpiration) || s.PetId != escrowPetId {
			t.Errorf("unwind item %s lost its cash/expiry/pet state: %+v", it.Item.EscrowId, s)
		}
		delete(want, it.Item.EscrowId)
	}
	if len(want) != 0 {
		t.Errorf("escrow rows left in custody after the teardown: %v", want)
	}

	if len(payload.Mesos) != 1 {
		t.Fatalf("unwind meso refunds: got %d, want 1", len(payload.Mesos))
	}
	if payload.Mesos[0].CharacterId != 100 || payload.Mesos[0].Amount != 250_000 {
		t.Errorf("unwind meso refund: got %+v, want 250000 back to character 100", payload.Mesos[0])
	}
	if payload.Mesos[0].WorldId != room.Field().WorldId() || payload.Mesos[0].ChannelId != room.Field().ChannelId() {
		t.Errorf("unwind meso field: got world %d channel %d, want the room's %d / %d", payload.Mesos[0].WorldId, payload.Mesos[0].ChannelId, room.Field().WorldId(), room.Field().ChannelId())
	}
}

// TestTeardownOfAnEmptyRoomSubmitsNoSaga pins the other half: most cancelled
// trades never staged anything, and a saga per empty teardown would be pure
// noise for the orchestrator to expand into nothing.
func TestTeardownOfAnEmptyRoomSubmitsNoSaga(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	assertNoUnwindSubmitted(t, e)
	assertCancelledWithReason(t, e, ReasonTradeCancelled)
}

// TestTeardownUnwindsWhatIsEscrowedNotWhatTheDialogShows pins that the unwind
// reads the CUSTODY STORE rather than the room.
//
// The disagreement it pins is one-directional and specific: a stage whose
// accept_to_trade has not yet written its row holds a dialog slot with nothing
// behind it, and returning that would hand the player an item that never left
// their inventory. Its own staging saga resolves it instead (see
// TestStageSucceededForAnOrphanedEscrowRowReturnsTheItem).
//
// The converse does NOT hold, and reading this test as though it did is what
// produced the duplicate-return defect: an item's dialog-side `pending` flag is
// cleared only when the stage's saga status reaches atlas-trades, many hops
// AFTER the custody consumer wrote its row, so a pending item usually does have
// a row. That window is what the return claim covers — see
// TestTeardownThenLateStageStatusReturnsTheItemOnce.
func TestTeardownUnwindsWhatIsEscrowedNotWhatTheDialogShows(t *testing.T) {
	p, e := testOpenRoom(t)
	stageOne(t, p, 100, stagingSourceSlot, 5, 1)
	// A second stage whose accept_to_trade has not landed: a dialog slot, and
	// deliberately no escrow row behind it.
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), 2, 1, 2); err != nil {
		t.Fatalf("pending put item: %v", err)
	}
	if got := len(stagedItemsOf(t, p, 100)); got != 2 {
		t.Fatalf("staged items: got %d, want 2 (one confirmed, one pending)", got)
	}

	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}

	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 {
		t.Fatalf("trade_unwind sagas: got %d, want 1", len(unwinds))
	}
	if got := len(unwindPayloadOf(t, unwinds[0]).Items); got != 1 {
		t.Errorf("unwind items: got %d, want only the ESCROWED one (1)", got)
	}
}

// unwoundEscrowIds returns every escrow id that appears in ANY trade_unwind the
// service submitted, in submission order and WITH duplicates. Counting across
// submissions is the point: a row returned by two different sagas is granted to
// its owner twice, and each saga on its own looks perfectly well-formed.
func unwoundEscrowIds(t *testing.T, e *emitted) []uuid.UUID {
	t.Helper()
	var out []uuid.UUID
	for _, s := range unwindSagas(t, e) {
		for _, i := range unwindPayloadOf(t, s).Items {
			out = append(out, i.Item.EscrowId)
		}
	}
	return out
}

// assertReturnedExactlyOnce requires the escrow row to appear in exactly one
// unwind item across every submission.
func assertReturnedExactlyOnce(t *testing.T, e *emitted, escrowId uuid.UUID) {
	t.Helper()
	n := 0
	for _, id := range unwoundEscrowIds(t, e) {
		if id == escrowId {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("trade_unwind submissions carrying escrow row [%s]: got %d, want exactly 1 — every submission ends in its own accept_to_character, so %d means the owner is handed the item %d times", escrowId, n, n, n)
	}
}

// TestTeardownThenLateStageStatusReturnsTheItemOnce pins the duplicate-return
// defect in its natural order.
//
// The row is written by the custody consumer as soon as accept_to_trade lands;
// atlas-trades only learns the stage succeeded several hops later, when the
// saga's terminal status arrives. Throughout that window the item is escrowed
// AND still `pending` on the dialog side. A teardown in that window reads
// ItemsByRoom — which has no pending filter — and unwinds the row; the terminal
// status then arrives, finds no room to claim it, and unwinds it again. Both
// sagas run accept_to_character, and nothing downstream can tell them apart:
// each mints its own transaction id, and the duplicate release_from_trade is
// silently fine because a no-match delete is success.
//
// Only the escrow row's return claim stands between that sequence and a
// duplicated item.
func TestTeardownThenLateStageStatusReturnsTheItemOnce(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	i := pendingItemIn(t, p, 100, 1)

	// accept_to_trade landed: the row exists. The stage is still pending on the
	// dialog side, because its terminal status has not reached atlas-trades.
	escrowOf(t, p).accept(room.Id(), 100, i)

	// The counterparty leaves. The teardown unwinds what is escrowed right now,
	// which includes this row.
	if err := p.TeardownCharacter(uuid.New(), 200, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}

	// The stage's terminal status finally arrives. Its room is gone.
	claimed, err := p.StageSucceeded(uuid.New(), i.EscrowId())
	if err != nil {
		t.Fatalf("stage succeeded: %v", err)
	}
	if !claimed {
		t.Fatal("the late stage status was not recognised as a stage; the caller would go on to treat it as a settlement's")
	}

	assertReturnedExactlyOnce(t, e, i.EscrowId())
}

// TestLateStageStatusThenTeardownReturnsTheItemOnce pins the same row under the
// opposite interleaving.
//
// It is reachable because the room registry is in-memory and NOT transactional:
// teardownRoom removes the room first (claimRoom) and only then reads the escrow
// to unwind it, so a stage's terminal status delivered in between finds no room,
// takes the orphan path, and claims the row before the teardown that removed the
// room ever looks at it. The test drives teardownRoom's own two halves in that
// order rather than approximating them.
func TestLateStageStatusThenTeardownReturnsTheItemOnce(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	i := pendingItemIn(t, p, 100, 1)
	escrowOf(t, p).accept(room.Id(), 100, i)

	// The teardown's first half: the room is claimed and removed from the
	// registry. Its unwind has not read the escrow yet.
	claimedRoom, ok := p.claimRoom(room, notSettling)
	if !ok {
		t.Fatal("the teardown lost its own room claim")
	}

	// The stage's terminal status lands in that gap: no room, so the orphan path
	// claims the row and returns it.
	staged, err := p.StageSucceeded(uuid.New(), i.EscrowId())
	if err != nil {
		t.Fatalf("stage succeeded: %v", err)
	}
	if !staged {
		t.Fatal("the late stage status was not recognised as a stage")
	}

	// The teardown's second half now runs and must find nothing left to return.
	if err := p.emit(func(txp *ProcessorImpl, mb *message.Buffer) error {
		return txp.emitUnwind(mb, claimedRoom.room)
	}); err != nil {
		t.Fatalf("emit unwind: %v", err)
	}

	assertReturnedExactlyOnce(t, e, i.EscrowId())
}

// TestTeardownOfASettlingRoomUnwindsNothing pins FR-6.5 on the custody side: a
// settling room's escrow belongs to the settlement saga, and returning it under
// that saga would hand back items it is about to deliver.
func TestTeardownOfASettlingRoomUnwindsNothing(t *testing.T) {
	p, e := testOpenRoom(t)
	stageOne(t, p, 100, stagingSourceSlot, 5, 1)
	room, _ := p.RoomForCharacter(100)
	if _, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		return cur.WithState(StateSettling), nil
	}); err != nil {
		t.Fatalf("move to settling: %v", err)
	}

	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	assertNoUnwindSubmitted(t, e)
}

// compile-time assurance the staging fakes satisfy the seams they stand in for.
var (
	_ inventoryProvider = (*fakeInventory)(nil)
	_ itemDataProvider  = (*fakeItemData)(nil)
	_ escrowStore       = (*fakeEscrow)(nil)
)

// TestRetypingTheMesoBoxMidSagaConservesMeso is the reported destruction bug,
// driven end to end through the exact sequence a player produces.
//
// The client permits it: CTradingRoomDlg::PutMoney arms CWvsContext's excl
// latch on send, but the debit's own STAT_CHANGED clears that latch before the
// trade-level MESO_STAGED lands, so a player who retypes the box faster than a
// saga round trip has two award_mesos in flight at once.
//
// The invariant is arithmetic and total: the sum of what the sagas debited must
// equal what escrow ends up holding. Netting the second stake against committed
// escrow alone made the second saga debit the full 200000 on top of the first
// 100000 while escrow settled at 200000 — 100000 destroyed, with no error
// anywhere.
func TestRetypingTheMesoBoxMidSagaConservesMeso(t *testing.T) {
	const (
		firstTyped  = int32(100_000)
		secondTyped = int32(200_000)
	)

	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	room, _ := p.RoomForCharacter(100)

	if err := p.AddMeso(uuid.New(), 100, firstTyped); err != nil {
		t.Fatalf("first add meso: %v", err)
	}
	// Deliberately NOT resolving the first stake before the second is staged.
	if err := p.AddMeso(uuid.New(), 100, secondTyped); err != nil {
		t.Fatalf("second add meso: %v", err)
	}

	stakes := sagasWithAction(t, e, sharedsaga.AwardMesos)
	if len(stakes) != 2 {
		t.Fatalf("award_mesos sagas: got %d, want 2 — the retype must submit its own movement", len(stakes))
	}

	// award_mesos amounts are signed from the PLAYER's side: a stake debits, so
	// the amounts are negative and their sum is what left the player's pocket.
	var debited int64
	for _, s := range stakes {
		debited -= int64(awardMesosPayloadOf(t, s).Amount)
	}
	if debited != int64(secondTyped) {
		t.Errorf("the two stakes debited %d in total, but the player only ever typed %d — netting the retype against committed escrow alone double-debits", debited, secondTyped)
	}

	// Both debits landed, so both terminal statuses must be honoured.
	for _, s := range stakes {
		claimed, err := p.MesoStageSucceeded(uuid.New(), s.TransactionId)
		if err != nil {
			t.Fatalf("resolve stake %s: %v", s.TransactionId, err)
		}
		if !claimed {
			t.Fatalf("stake %s was not claimed; its debit already moved real meso and nothing else will account for it", s.TransactionId)
		}
	}

	held, _, err := escrow.MesoByOwner(p.db, p.t.Id())(room.Id(), 100)
	if err != nil {
		t.Fatalf("MesoByOwner: %v", err)
	}
	if held != debited {
		t.Errorf("conservation broken: the player was debited %d but escrow holds %d (%d meso unaccounted for)", debited, held, debited-held)
	}
	if held != int64(secondTyped) {
		t.Errorf("escrow holds %d, want the %d the player last typed", held, secondTyped)
	}
}

// TestLoweringTheMesoBoxNeverPaysOutAgainstAnUnconfirmedStake is the mint that
// composition opened, and the reason a PAYOUT may not be netted against money
// that has not landed.
//
// Composition nets a new stake against committed PLUS in flight, which is what
// makes a retype conserve. But a stake's movement is submitted immediately, and
// a NEGATIVE delta is a credit to the player's wallet (award_mesos carries
// -delta). So lowering the box while a raise is still in flight pays real meso
// out on the strength of a debit that has not happened yet:
//
//	stake 1000  -> delta +1000, debit 1000 submitted, unresolved
//	retype 200  -> delta  -800 against (0 + 1000), CREDIT 800 submitted
//	the debit FAILS (an ordinary outcome: atlas-character re-checks the live
//	balance at execution time and rejects if the player spent in the meantime)
//
// The player is now 800 up having been debited nothing. Escrow's own books land
// at -800, which every consumer correctly reads as "nothing owed", so the
// settlement gate can refuse to deliver the trade but cannot claw back a credit
// that already reached the wallet.
//
// Raises do NOT have this problem and are deliberately left composing: if an
// earlier debit fails, later debits still only took what they took, and the
// committed total still equals the sum that landed.
func TestLoweringTheMesoBoxNeverPaysOutAgainstAnUnconfirmedStake(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)

	if err := p.AddMeso(uuid.New(), 100, 1_000); err != nil {
		t.Fatalf("raise: %v", err)
	}
	// Retyped DOWN before the raise resolved.
	if err := p.AddMeso(uuid.New(), 100, 200); err != nil {
		t.Fatalf("lower: %v", err)
	}

	for _, s := range sagasWithAction(t, e, sharedsaga.AwardMesos) {
		if got := awardMesosPayloadOf(t, s).Amount; got > 0 {
			t.Fatalf("a credit of %d was submitted while a debit was still in flight; if that debit fails the player keeps it and nothing ever claws it back", got)
		}
	}
}
