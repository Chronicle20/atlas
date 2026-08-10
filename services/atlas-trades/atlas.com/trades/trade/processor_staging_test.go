package trade

import (
	"atlas-trades/configuration"
	inventorydata "atlas-trades/data/inventory"
	"atlas-trades/escrow"
	trademsg "atlas-trades/kafka/message/trade"
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
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

// escrowStatWeaponAttack / escrowStatSlots / escrowOwnerName are the stat block
// every accepted escrow row carries in these tests. They are deliberately
// non-zero and mutually distinct so a settlement payload that dropped the
// snapshot — or read it from the room rather than from the custody row — shows
// up as a concrete wrong number rather than as a plausible zero.
const (
	escrowStatWeaponAttack = uint16(17)
	escrowStatSlots        = uint16(7)
	escrowOwnerName        = "Chronicle"
)

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
	mesos map[mesoOwner]uint32
	err   error
}

func newFakeEscrow() *fakeEscrow {
	return &fakeEscrow{mesos: make(map[mesoOwner]uint32)}
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

func (f *fakeEscrow) MesoByOwner(roomId uuid.UUID, ownerId character.Id) (uint32, bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.err != nil {
		return 0, false, f.err
	}
	amount, ok := f.mesos[mesoOwner{roomId: roomId, ownerId: ownerId}]
	return amount, ok, nil
}

// accept writes the custody row a completed transfer_to_trade would have
// written, from the staged item that named it.
func (f *fakeEscrow) accept(roomId uuid.UUID, ownerId character.Id, i StagedItem) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.items = append(f.items, escrow.NewItemBuilder(i.EscrowId(), roomId, ownerId).
		SetTradeSlot(i.TradeSlot()).
		SetSource(i.InventoryType(), i.SourceSlot(), i.AssetId()).
		SetTemplateId(i.TemplateId()).
		SetQuantity(i.Quantity()).
		SetWeaponAttack(escrowStatWeaponAttack).
		SetSlots(escrowStatSlots).
		SetOwner(escrowOwnerName).
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

// setMeso records a participant's COMMITTED escrowed meso total, which is what
// a completed award_mesos leaves behind.
func (f *fakeEscrow) setMeso(roomId uuid.UUID, ownerId character.Id, amount uint32) {
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

// TestPutItemDropsSilentlyOnRestrictionFailure pins design §7: the reference
// client has no put-item-time error for "untradeable", so the empty slot IS the
// feedback. No clientbound update, no error event, and nothing escrowed.
func TestPutItemDropsSilentlyOnRestrictionFailure(t *testing.T) {
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
	if staged[0].Body.Position != 0 || staged[0].Body.TradeSlot != 3 || staged[0].Body.Quantity != 5 {
		t.Errorf("ITEM_STAGED body: got %+v, want position 0 trade slot 3 quantity 5", staged[0].Body)
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
		escrowed uint32
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
				escrowOf(t, p).setMeso(room.Id(), 100, tc.escrowed)
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
	escrowOf(t, p).setMeso(room.Id(), 100, 1_000_000)

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
	escrowOf(t, p).setMeso(room.Id(), 100, 400_000)

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
	escrowOf(t, p).setMeso(room.Id(), 100, 1_000_000)

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
// negative cannot come from the input box, so there is nothing to re-echo — the
// command is simply dropped.
func TestAddMesoRejectsNegative(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	if err := p.AddMeso(uuid.New(), 100, -1); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeMesoStaged)
	assertNoEventOfType(t, e, trademsg.StatusTypeMesoRefused)
	assertNoSagaOfAction(t, e, sharedsaga.AwardMesos)
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
	escrowOf(t, p).err = errors.New("escrow store unreachable")

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
	escrowOf(t, p).setMeso(room.Id(), 100, 250_000)

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
		if it.Item.WeaponAttack != escrowStatWeaponAttack || it.Item.Slots != escrowStatSlots || it.Item.Owner != escrowOwnerName {
			t.Errorf("unwind item %s lost its stat snapshot: %+v", it.Item.EscrowId, it.Item)
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
// reads the CUSTODY STORE rather than the room. The two legitimately disagree
// while a stage is in flight: a pending item holds a dialog slot but has no row
// yet, and returning it would hand the player an item that never left their
// inventory. Its own staging saga resolves it instead (see
// TestStageSucceededForAnOrphanedEscrowRowReturnsTheItem).
func TestTeardownUnwindsWhatIsEscrowedNotWhatTheDialogShows(t *testing.T) {
	p, e := testOpenRoom(t)
	stageOne(t, p, 100, stagingSourceSlot, 5, 1)
	// A second stage that is still in flight: a dialog slot, but no escrow row.
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
