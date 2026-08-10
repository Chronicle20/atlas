package trade

import (
	"atlas-trades/configuration"
	inventorydata "atlas-trades/data/inventory"
	compartmentmsg "atlas-trades/kafka/message/compartment"
	trademsg "atlas-trades/kafka/message/trade"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
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
	// onGetCompartment fires on each compartment read. The refresh does its
	// reads between snapshotting the registry and taking the write lock, so
	// this is the seam a test uses to land a concurrent teardown in exactly
	// that window.
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
// stored Asset, so a test can relocate an item exactly as an inventory swap
// would — by re-keying it — and the returned compartment reports the new slot.
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
	return inventorydata.NewModel(uuid.New(), inventoryType, 24, assets), nil
}

// relocate moves an asset to another slot the way an atlas-inventory swap
// would: the asset keeps its id, and only its key changes.
func (f *fakeInventory) relocate(characterId character.Id, inventoryType inventory.Type, from slot.Position, to slot.Position) {
	a, ok := f.assets[assetKey{characterId, inventoryType, from}]
	if !ok {
		return
	}
	delete(f.assets, assetKey{characterId, inventoryType, from})
	f.assets[assetKey{characterId, inventoryType, to}] = a
}

type fakeItemData struct {
	blocked map[item.Id]bool
	err     error
}

func (f *fakeItemData) TradeBlock(_ inventory.Type, templateId item.Id) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.blocked[templateId], nil
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
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle()); err != nil {
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

// compartmentCommands decodes every COMMAND_TOPIC_COMPARTMENT message of the
// given type, in publication order.
func compartmentCommands[E any](t *testing.T, e *emitted, commandType string) []compartmentmsg.Command[E] {
	t.Helper()
	var out []compartmentmsg.Command[E]
	for _, raw := range e.messages(t, compartmentmsg.EnvCommandTopic) {
		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			t.Fatalf("probe compartment command: %v", err)
		}
		if probe.Type != commandType {
			continue
		}
		var cmd compartmentmsg.Command[E]
		if err := json.Unmarshal(raw, &cmd); err != nil {
			t.Fatalf("decode %s: %v", commandType, err)
		}
		out = append(out, cmd)
	}
	return out
}

// assertReserveCommand requires exactly one REQUEST_RESERVE holding `quantity`
// of the asset in (inventoryType, sourceSlot) for the character, and returns it.
func assertReserveCommand(t *testing.T, e *emitted, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position, quantity int16) compartmentmsg.Command[compartmentmsg.RequestReserveCommandBody] {
	t.Helper()
	cmds := compartmentCommands[compartmentmsg.RequestReserveCommandBody](t, e, compartmentmsg.CommandRequestReserve)
	if len(cmds) != 1 {
		t.Fatalf("REQUEST_RESERVE commands: got %d, want 1", len(cmds))
	}
	cmd := cmds[0]
	if cmd.CharacterId != uint32(characterId) {
		t.Errorf("reserve characterId: got %d, want %d", cmd.CharacterId, characterId)
	}
	if cmd.InventoryType != byte(inventoryType) {
		t.Errorf("reserve inventoryType: got %d, want %d", cmd.InventoryType, byte(inventoryType))
	}
	if len(cmd.Body.Items) != 1 {
		t.Fatalf("reserve items: got %d, want 1", len(cmd.Body.Items))
	}
	if cmd.Body.Items[0].Source != int16(sourceSlot) {
		t.Errorf("reserve source slot: got %d, want %d", cmd.Body.Items[0].Source, sourceSlot)
	}
	if cmd.Body.Items[0].Quantity != quantity {
		t.Errorf("reserve quantity: got %d, want %d", cmd.Body.Items[0].Quantity, quantity)
	}
	return cmd
}

// assertNoCompartmentCommandOfType requires that no compartment command of the
// given type was published.
func assertNoCompartmentCommandOfType(t *testing.T, e *emitted, commandType string) {
	t.Helper()
	if cmds := compartmentCommands[json.RawMessage](t, e, commandType); len(cmds) != 0 {
		t.Errorf("%s commands: got %d, want 0", commandType, len(cmds))
	}
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

// TestPutItemReservesRatherThanEscrows pins design §5.3 Option A: staging is a
// reservation, NOT a release. Nothing leaves the owner's inventory until
// settlement, so a crash orphans nothing.
func TestPutItemReservesRatherThanEscrows(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 3); err != nil {
		t.Fatalf("put item: %v", err)
	}
	assertReserveCommand(t, e, 100, inventory.TypeValueUse, stagingSourceSlot, 5)
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandCancelReservation)

	items := stagedItemsOf(t, p, 100)
	if len(items) != 1 {
		t.Fatalf("staged items: got %d, want 1", len(items))
	}
	if items[0].TradeSlot() != 3 || items[0].Quantity() != 5 || items[0].TemplateId() != stagingTemplateId {
		t.Errorf("staged item: got %+v, want trade slot 3 quantity 5 of template %d", items[0], stagingTemplateId)
	}
	if items[0].ReservationId() == uuid.Nil {
		t.Error("staged item carries no reservation id; the reservation could never be cancelled")
	}

	staged := statusEvents[trademsg.ItemStagedEventBody](t, e, trademsg.StatusTypeItemStaged)
	if len(staged) != 1 {
		t.Fatalf("ITEM_STAGED events: got %d, want 1", len(staged))
	}
	if staged[0].Body.Position != 0 || staged[0].Body.TradeSlot != 3 || staged[0].Body.Quantity != 5 {
		t.Errorf("ITEM_STAGED body: got %+v, want position 0 trade slot 3 quantity 5", staged[0].Body)
	}
}

// TestPutItemReserveCarriesTheStagedItemsHandle pins the reservation lifecycle's
// only linkage: the id on the command must be the id the staged item records, or
// teardown cancels a reservation that does not exist and leaks the real one.
func TestPutItemReserveCarriesTheStagedItemsHandle(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 3); err != nil {
		t.Fatalf("put item: %v", err)
	}
	cmd := assertReserveCommand(t, e, 100, inventory.TypeValueUse, stagingSourceSlot, 5)
	want := stagedItemsOf(t, p, 100)[0].ReservationId()
	if cmd.TransactionId != want {
		t.Errorf("reserve envelope transactionId: got %s, want the staged item's reservation %s", cmd.TransactionId, want)
	}
	if cmd.Body.TransactionId != want {
		t.Errorf("reserve body transactionId: got %s, want %s", cmd.Body.TransactionId, want)
	}
}

// TestPutItemReserveUsesTheConfiguredTtl pins design §5.3: 300s by default, not
// the 30s drop-reservation lifetime atlas-inventory falls back to when the field
// is zero.
func TestPutItemReserveUsesTheConfiguredTtl(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 3); err != nil {
		t.Fatalf("put item: %v", err)
	}
	cmd := assertReserveCommand(t, e, 100, inventory.TypeValueUse, stagingSourceSlot, 5)
	if cmd.Body.ExpirySeconds != 300 {
		t.Errorf("expirySeconds: got %d, want 300", cmd.Body.ExpirySeconds)
	}
}

// TestPutItemReserveHonoursATenantTtl pins that the TTL is config-resolved
// rather than a constant that happens to equal the default.
func TestPutItemReserveHonoursATenantTtl(t *testing.T) {
	p, e := testOpenRoomWithConfig(t, configuration.DefaultConfig().WithReservationTtl(90*time.Second))
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 3); err != nil {
		t.Fatalf("put item: %v", err)
	}
	cmd := assertReserveCommand(t, e, 100, inventory.TypeValueUse, stagingSourceSlot, 5)
	if cmd.Body.ExpirySeconds != 90 {
		t.Errorf("expirySeconds: got %d, want the tenant's 90", cmd.Body.ExpirySeconds)
	}
}

// TestPutItemRejectsOccupiedOrOutOfRangeSlot pins FR-3.3: the dialog's slots are
// 1..maxStagedItems and each holds at most one item.
func TestPutItemRejectsOccupiedOrOutOfRangeSlot(t *testing.T) {
	for name, target := range map[string]byte{"occupied": 3, "zero": 0, "above nine": 10} {
		t.Run(name, func(t *testing.T) {
			p, e := testOpenRoom(t)
			if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 3); err != nil {
				t.Fatalf("first put item: %v", err)
			}
			if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), 2, 1, target); err != nil {
				t.Fatalf("second put item: %v", err)
			}
			if got := len(stagedItemsOf(t, p, 100)); got != 1 {
				t.Errorf("staged items: got %d, want the original 1", got)
			}
			// The refused stage must not have reserved anything either.
			assertReserveCommand(t, e, 100, inventory.TypeValueUse, stagingSourceSlot, 5)
		})
	}
}

// TestPutItemHonoursMaxStagedItems pins FR-9.1's configurable cap: a third stage
// under a cap of 2 is refused even though its dialog slot is free.
func TestPutItemHonoursMaxStagedItems(t *testing.T) {
	p, e := testOpenRoomWithConfig(t, configuration.DefaultConfig().WithMaxStagedItems(2))
	for i, sourceSlot := range []slot.Position{stagingSourceSlot, 2, 3} {
		if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), sourceSlot, 1, byte(i+1)); err != nil {
			t.Fatalf("put item %d: %v", i, err)
		}
	}
	if got := len(stagedItemsOf(t, p, 100)); got != 2 {
		t.Errorf("staged items: got %d, want the configured cap of 2", got)
	}
	if got := len(compartmentCommands[compartmentmsg.RequestReserveCommandBody](t, e, compartmentmsg.CommandRequestReserve)); got != 2 {
		t.Errorf("REQUEST_RESERVE commands: got %d, want 2 — the refused stage must not reserve", got)
	}
	// Trade slot 3 was inside the client's 1..9 dialog but outside the tenant's
	// cap, so the third stage is refused on the cap, not on the slot.
	staged := statusEvents[trademsg.ItemStagedEventBody](t, e, trademsg.StatusTypeItemStaged)
	if len(staged) != 2 {
		t.Errorf("ITEM_STAGED events: got %d, want 2", len(staged))
	}
}

// TestPutItemDropsSilentlyOnRestrictionFailure pins design §7: the reference
// client has no put-item-time error for "untradeable", so the empty slot IS the
// feedback. No clientbound update, no error event, and no reservation.
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
			assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandRequestReserve)
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
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandRequestReserve)
	if got := len(stagedItemsOf(t, p, 100)); got != 0 {
		t.Errorf("staged items: got %d, want 0", got)
	}
}

// TestPutItemDropsAnEmptySlot pins that staging from a slot the character does
// not occupy is refused rather than reserving a phantom asset.
func TestPutItemDropsAnEmptySlot(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), 42, 1, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandRequestReserve)
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
	if got := len(compartmentCommands[compartmentmsg.RequestReserveCommandBody](t, e, compartmentmsg.CommandRequestReserve)); got != 1 {
		t.Errorf("REQUEST_RESERVE commands: got %d, want 1", got)
	}
}

// TestPutItemRejectsQuantityBeyondTheWiresInt16 pins the width the reserve
// command actually carries: ItemBody.Quantity is int16, so a uint16 above
// math.MaxInt16 would wrap negative and atlas-inventory would widen it into a
// near-4-billion hold on the player's own stack.
func TestPutItemRejectsQuantityBeyondTheWiresInt16(t *testing.T) {
	p, e := testOpenRoom(t)
	p.invp = &fakeInventory{assets: map[assetKey]inventorydata.Asset{
		{100, inventory.TypeValueUse, stagingSourceSlot}: inventorydata.NewAsset(stagingAssetId, stagingSourceSlot, stagingTemplateId, 65_000, 0),
	}}
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 40_000, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandRequestReserve)
	if got := len(stagedItemsOf(t, p, 100)); got != 0 {
		t.Errorf("staged items: got %d, want 0", got)
	}
}

// TestPutItemRejectsZeroQuantity pins the other end of the same range: a zero
// reservation would be filed and immediately mean nothing.
func TestPutItemRejectsZeroQuantity(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 0, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandRequestReserve)
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
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandRequestReserve)
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
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandRequestReserve)
	if got := len(stagedItemsOf(t, p, 200)); got != 0 {
		t.Errorf("staged items: got %d, want 0", got)
	}
	if got := mesoStagedBy(t, p, 200); got != 0 {
		t.Errorf("mesoStaged: got %d, want 0", got)
	}
}

// --- ADD_MESO ----------------------------------------------------------------

// TestAddMesoIsAbsoluteNotADelta pins design §1.6: PutMoney sends the absolute
// total from the input box, and the clientbound echo is an assignment.
func TestAddMesoIsAbsoluteNotADelta(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	if err := p.AddMeso(uuid.New(), 100, 1_000_000); err != nil {
		t.Fatalf("first add meso: %v", err)
	}
	if err := p.AddMeso(uuid.New(), 100, 2_000_000); err != nil {
		t.Fatalf("second add meso: %v", err)
	}
	if got := mesoStagedBy(t, p, 100); got != 2_000_000 {
		t.Errorf("mesoStaged: got %d, want 2000000 (absolute, not 3000000 accumulated)", got)
	}
	staged := statusEvents[trademsg.MesoStagedEventBody](t, e, trademsg.StatusTypeMesoStaged)
	if len(staged) != 2 {
		t.Fatalf("MESO_STAGED events: got %d, want 2", len(staged))
	}
	if staged[1].Body.Amount != 2_000_000 || staged[1].Body.Position != 0 {
		t.Errorf("MESO_STAGED body: got %+v, want position 0 amount 2000000", staged[1].Body)
	}
}

// TestAddMesoRefusedReEchoesTheLastValidAmount pins FR-4.8 / design §4.2: an
// out-of-range stage is corrected by an AUTHORITATIVE re-echo so the client's
// view snaps back, and the staged amount is left untouched.
func TestAddMesoRefusedReEchoesTheLastValidAmount(t *testing.T) {
	p, e := testOpenRoomWithMeso(t, 100, 5_000_000)
	if err := p.AddMeso(uuid.New(), 100, 1_000_000); err != nil {
		t.Fatalf("first add meso: %v", err)
	}
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
	if got := len(statusEvents[trademsg.MesoStagedEventBody](t, e, trademsg.StatusTypeMesoStaged)); got != 1 {
		t.Errorf("MESO_STAGED events: got %d, want only the first, valid one", got)
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
	if got := mesoStagedBy(t, p, 100); got != 0 {
		t.Errorf("mesoStaged: got %d, want 0", got)
	}
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

// --- reservation lifecycle ---------------------------------------------------

// TestTeardownCancelsEveryStagedReservation pins the obligation the reserve
// model creates: a staged asset never left its owner's inventory, so a room that
// disappears without cancelling leaves that stack locked — unmovable,
// unmergeable, undroppable — until the TTL lapses.
func TestTeardownCancelsEveryStagedReservation(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("owner put item: %v", err)
	}
	if err := p.PutItem(uuid.New(), 200, byte(inventory.TypeValueUse), 2, 7, 1); err != nil {
		t.Fatalf("visitor put item: %v", err)
	}
	want := map[uuid.UUID]character.Id{}
	for _, id := range []character.Id{100, 200} {
		for _, i := range stagedItemsOf(t, p, id) {
			want[i.ReservationId()] = id
		}
	}
	if len(want) != 2 {
		t.Fatalf("staged reservations: got %d, want 2", len(want))
	}

	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}

	cancels := compartmentCommands[compartmentmsg.CancelReservationCommandBody](t, e, compartmentmsg.CommandCancelReservation)
	if len(cancels) != 2 {
		t.Fatalf("CANCEL_RESERVATION commands: got %d, want one per staged item (2)", len(cancels))
	}
	for _, c := range cancels {
		owner, ok := want[c.TransactionId]
		if !ok {
			t.Errorf("cancelled reservation %s was never staged", c.TransactionId)
			continue
		}
		if c.CharacterId != uint32(owner) {
			t.Errorf("cancel characterId: got %d, want the staging owner %d", c.CharacterId, owner)
		}
		if c.InventoryType != byte(inventory.TypeValueUse) {
			t.Errorf("cancel inventoryType: got %d, want %d", c.InventoryType, byte(inventory.TypeValueUse))
		}
		delete(want, c.TransactionId)
	}
	if len(want) != 0 {
		t.Errorf("reservations left uncancelled: %v", want)
	}
}

// TestTeardownOfASettlingRoomCancelsNothing pins FR-6.5 on the reservation side:
// a settling room's holds belong to the settlement saga, and cancelling them
// under it would release the assets it is about to consume.
func TestTeardownOfASettlingRoomCancelsNothing(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	if _, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		return cur.WithState(StateSettling), nil
	}); err != nil {
		t.Fatalf("move to settling: %v", err)
	}

	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandCancelReservation)
}

// TestRefreshReservationsCancelsBeforeReReserving pins the shape the refresh
// MUST take. atlas-inventory has no refresh primitive and its AddReservation
// appends unconditionally, so a bare re-send would stack a second hold on the
// same slot every tick until the player's own stack read as fully reserved.
func TestRefreshReservationsCancelsBeforeReReserving(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	reservationId := stagedItemsOf(t, p, 100)[0].ReservationId()

	if err := p.RefreshReservations(uuid.New()); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	cancels := compartmentCommands[compartmentmsg.CancelReservationCommandBody](t, e, compartmentmsg.CommandCancelReservation)
	if len(cancels) != 1 {
		t.Fatalf("CANCEL_RESERVATION commands: got %d, want 1", len(cancels))
	}
	if cancels[0].TransactionId != reservationId {
		t.Errorf("cancel reservation id: got %s, want %s", cancels[0].TransactionId, reservationId)
	}

	reserves := compartmentCommands[compartmentmsg.RequestReserveCommandBody](t, e, compartmentmsg.CommandRequestReserve)
	if len(reserves) != 2 {
		t.Fatalf("REQUEST_RESERVE commands: got %d, want the original plus the refresh (2)", len(reserves))
	}
	if reserves[1].TransactionId != reservationId {
		t.Errorf("refreshed reservation id: got %s, want the same handle %s", reserves[1].TransactionId, reservationId)
	}
	if reserves[1].Body.ExpirySeconds != 300 {
		t.Errorf("refreshed expirySeconds: got %d, want 300", reserves[1].Body.ExpirySeconds)
	}
	if len(reserves[1].Body.Items) != 1 || reserves[1].Body.Items[0].Quantity != 5 {
		t.Errorf("refreshed reserve items: got %+v, want the original quantity 5", reserves[1].Body.Items)
	}

	// Ordering matters: the cancel has to reach atlas-inventory first, or the
	// re-reserve is the duplicate this whole shape exists to avoid.
	raws := e.messages(t, compartmentmsg.EnvCommandTopic)
	if len(raws) != 3 {
		t.Fatalf("compartment commands: got %d, want 3", len(raws))
	}
	var second struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raws[1], &second); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if second.Type != compartmentmsg.CommandCancelReservation {
		t.Errorf("second command: got %s, want the cancel to precede the re-reserve", second.Type)
	}
}

// TestRefreshReservationsSkipsASettlingRoom pins that the ticker does not race
// the settlement saga's consume by cancelling the holds it is about to spend.
func TestRefreshReservationsSkipsASettlingRoom(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	if _, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
		return cur.WithState(StateSettling), nil
	}); err != nil {
		t.Fatalf("move to settling: %v", err)
	}

	if err := p.RefreshReservations(uuid.New()); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandCancelReservation)
	if got := len(compartmentCommands[compartmentmsg.RequestReserveCommandBody](t, e, compartmentmsg.CommandRequestReserve)); got != 1 {
		t.Errorf("REQUEST_RESERVE commands: got %d, want only the original stage's", got)
	}
}

// --- a staged item that moves in the owner's inventory ------------------------

// TestTeardownCancelsAtTheAssetsCurrentSlotAfterAMove pins the leak an inventory
// rearrangement would otherwise cause. Nothing gates an inventory move while the
// trade dialog is open (atlas-channel's move handler has no mini-room check),
// and atlas-inventory re-keys a reserved slot's hold to the destination on a
// swap. A cancel aimed at the slot the item was staged FROM would therefore
// no-op while the real hold lived on until its TTL.
func TestTeardownCancelsAtTheAssetsCurrentSlotAfterAMove(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	// The player drags the staged stack from slot 1 to slot 8.
	p.invp.(*fakeInventory).relocate(100, inventory.TypeValueUse, stagingSourceSlot, 8)

	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}

	cancels := compartmentCommands[compartmentmsg.CancelReservationCommandBody](t, e, compartmentmsg.CommandCancelReservation)
	if len(cancels) != 1 {
		t.Fatalf("CANCEL_RESERVATION commands: got %d, want 1", len(cancels))
	}
	if cancels[0].Body.Slot != 8 {
		t.Errorf("cancel slot: got %d, want the asset's CURRENT slot 8 — cancelling the vacated slot %d leaks the real hold", cancels[0].Body.Slot, stagingSourceSlot)
	}
}

// TestRefreshFollowsTheAssetAndDoesNotPoisonTheVacatedSlot pins the other half:
// a refresh aimed at the vacated slot would both miss the real hold AND file a
// fresh 300s hold on whatever item was swapped in.
func TestRefreshFollowsTheAssetAndDoesNotPoisonTheVacatedSlot(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	p.invp.(*fakeInventory).relocate(100, inventory.TypeValueUse, stagingSourceSlot, 8)

	if err := p.RefreshReservations(uuid.New()); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	cancels := compartmentCommands[compartmentmsg.CancelReservationCommandBody](t, e, compartmentmsg.CommandCancelReservation)
	if len(cancels) != 1 {
		t.Fatalf("CANCEL_RESERVATION commands: got %d, want 1", len(cancels))
	}
	if cancels[0].Body.Slot != 8 {
		t.Errorf("refresh cancel slot: got %d, want 8", cancels[0].Body.Slot)
	}

	reserves := compartmentCommands[compartmentmsg.RequestReserveCommandBody](t, e, compartmentmsg.CommandRequestReserve)
	if len(reserves) != 2 {
		t.Fatalf("REQUEST_RESERVE commands: got %d, want 2", len(reserves))
	}
	if got := reserves[1].Body.Items[0].Source; got != 8 {
		t.Errorf("refresh reserve slot: got %d, want 8 — re-reserving the vacated slot %d poisons whatever item now occupies it", got, stagingSourceSlot)
	}
}

// TestRefreshWritesTheCorrectedSlotBackToTheStagedItem pins that the correction
// is durable, not per-emit. The settlement payload Task 19 builds resolves the
// asset by StagedItem.SourceSlot, so a slot left stale in the registry becomes a
// wrong-asset settlement.
func TestRefreshWritesTheCorrectedSlotBackToTheStagedItem(t *testing.T) {
	p, _ := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	p.invp.(*fakeInventory).relocate(100, inventory.TypeValueUse, stagingSourceSlot, 8)

	if err := p.RefreshReservations(uuid.New()); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	items := stagedItemsOf(t, p, 100)
	if len(items) != 1 {
		t.Fatalf("staged items: got %d, want 1", len(items))
	}
	if items[0].SourceSlot() != 8 {
		t.Errorf("staged sourceSlot: got %d, want the corrected 8", items[0].SourceSlot())
	}
	if items[0].AssetId() != stagingAssetId {
		t.Errorf("staged assetId: got %d, want the unchanged %d — a relocation is a correction, not a re-stage", items[0].AssetId(), stagingAssetId)
	}
}

// TestResolveStagedSlotFallsBackWhenTheCompartmentCannotBeRead pins that an
// unreadable inventory still cancels at the recorded slot. Cancelling the wrong
// slot is a no-op; cancelling nothing guarantees the leak.
func TestResolveStagedSlotFallsBackWhenTheCompartmentCannotBeRead(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	p.invp = &fakeInventory{err: errors.New("atlas-inventory unreachable")}

	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	cancels := compartmentCommands[compartmentmsg.CancelReservationCommandBody](t, e, compartmentmsg.CommandCancelReservation)
	if len(cancels) != 1 {
		t.Fatalf("CANCEL_RESERVATION commands: got %d, want 1", len(cancels))
	}
	if cancels[0].Body.Slot != int16(stagingSourceSlot) {
		t.Errorf("cancel slot: got %d, want the recorded fallback %d", cancels[0].Body.Slot, stagingSourceSlot)
	}
}

// TestRefreshSkipsARoomTornDownConcurrently pins the liveness re-check.
// reg.All returns a snapshot and teardown emits from a different transaction, so
// without the re-check a refresh could re-file a 300s hold on a room that no
// longer exists — and nothing would ever cancel it.
func TestRefreshSkipsARoomTornDownConcurrently(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	room, _ := p.RoomForCharacter(100)

	// Land the concurrent teardown in the real window: after the refresh has
	// snapshotted the registry, while it is doing its slot reads, before it
	// takes the write lock. Removing the room BEFORE the call would only prove
	// reg.All skips it, which is not the race being pinned.
	var once sync.Once
	p.invp.(*fakeInventory).onGetCompartment = func() {
		once.Do(func() { p.reg.Remove(p.t, room.Id()) })
	}

	if err := p.RefreshReservations(uuid.New()); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if got := len(compartmentCommands[compartmentmsg.RequestReserveCommandBody](t, e, compartmentmsg.CommandRequestReserve)); got != 1 {
		t.Errorf("REQUEST_RESERVE commands: got %d, want only the original stage's — the refresh re-filed a hold on a dead room", got)
	}
}

// TestRefreshSkipsARoomThatBeginsSettlingConcurrently pins the SETTLING half of
// the refreshability re-check. refreshReservations' own state test runs against
// the pre-read snapshot, so a room that transitions to SETTLING during the
// compartment reads would otherwise be corrected and then have CANCEL+
// REQUEST_RESERVE emitted against holds the settlement saga is about to consume
// — the exact race the snapshot check exists to prevent, through a window the
// REST reads made much wider.
func TestRefreshSkipsARoomThatBeginsSettlingConcurrently(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, 5, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	room, _ := p.RoomForCharacter(100)

	// Settle the room in the real window: after the snapshot, during the slot
	// reads, before the write lock.
	var once sync.Once
	p.invp.(*fakeInventory).onGetCompartment = func() {
		once.Do(func() {
			if _, err := p.reg.Update(p.t, room.Id(), func(cur Room) (Room, error) {
				return cur.WithState(StateSettling), nil
			}); err != nil {
				t.Errorf("move to settling: %v", err)
			}
		})
	}

	if err := p.RefreshReservations(uuid.New()); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandCancelReservation)
	if got := len(compartmentCommands[compartmentmsg.RequestReserveCommandBody](t, e, compartmentmsg.CommandRequestReserve)); got != 1 {
		t.Errorf("REQUEST_RESERVE commands: got %d, want only the original stage's — the refresh raced the settlement saga's consume", got)
	}
}

// TestRefreshReadsEachCompartmentOncePerPass pins the memoization. Resolving a
// slot per staged item re-fetched a whole compartment each time: at the default
// cap that is up to 18 GETs per room per tick for at most 2 distinct
// compartments, all issued serially inside the enclosing DB transaction, so a
// pooled connection is held open across every one of those round trips.
func TestRefreshReadsEachCompartmentOncePerPass(t *testing.T) {
	p, _ := testOpenRoom(t)
	// Four staged items across exactly two (character, compartment) pairs.
	for i, sourceSlot := range []slot.Position{stagingSourceSlot, 2, 3} {
		if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), sourceSlot, 1, byte(i+1)); err != nil {
			t.Fatalf("owner put item %d: %v", i, err)
		}
	}
	if err := p.PutItem(uuid.New(), 200, byte(inventory.TypeValueUse), stagingSourceSlot, 1, 1); err != nil {
		t.Fatalf("visitor put item: %v", err)
	}

	var reads int
	p.invp.(*fakeInventory).onGetCompartment = func() { reads++ }

	if err := p.RefreshReservations(uuid.New()); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if reads != 2 {
		t.Errorf("compartment reads: got %d, want 2 — one per (character, compartment), not one per staged item (4)", reads)
	}
}

// TestTeardownReadsEachCompartmentOncePerPass pins the same memoization on the
// teardown path, which resolves every staged item on both sides.
func TestTeardownReadsEachCompartmentOncePerPass(t *testing.T) {
	p, _ := testOpenRoom(t)
	for i, sourceSlot := range []slot.Position{stagingSourceSlot, 2, 3} {
		if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), sourceSlot, 1, byte(i+1)); err != nil {
			t.Fatalf("owner put item %d: %v", i, err)
		}
	}
	if err := p.PutItem(uuid.New(), 200, byte(inventory.TypeValueUse), stagingSourceSlot, 1, 1); err != nil {
		t.Fatalf("visitor put item: %v", err)
	}

	var reads int
	p.invp.(*fakeInventory).onGetCompartment = func() { reads++ }

	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	if reads != 2 {
		t.Errorf("compartment reads: got %d, want 2 — one per (character, compartment), not one per staged item (4)", reads)
	}
}

// TestResolveStagedSlotDoesNotRetryAFailedCompartmentRead pins that a failed
// read is memoized too: every item in that compartment gets the same fallback,
// so retrying per item would only multiply the latency of an atlas-inventory
// that is already unreachable.
func TestResolveStagedSlotDoesNotRetryAFailedCompartmentRead(t *testing.T) {
	p, e := testOpenRoom(t)
	for i, sourceSlot := range []slot.Position{stagingSourceSlot, 2, 3} {
		if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), sourceSlot, 1, byte(i+1)); err != nil {
			t.Fatalf("put item %d: %v", i, err)
		}
	}

	var reads int
	p.invp = &fakeInventory{
		err:              errors.New("atlas-inventory unreachable"),
		onGetCompartment: func() { reads++ },
	}

	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	if reads != 1 {
		t.Errorf("compartment reads: got %d, want 1 — a failed read must be memoized, not retried per staged item", reads)
	}
	// All three still cancel, at their recorded fallback slots.
	if got := len(compartmentCommands[compartmentmsg.CancelReservationCommandBody](t, e, compartmentmsg.CommandCancelReservation)); got != 3 {
		t.Errorf("CANCEL_RESERVATION commands: got %d, want 3", got)
	}
}

// compile-time assurance the staging fakes satisfy the seams they stand in for.
var (
	_ inventoryProvider = (*fakeInventory)(nil)
	_ itemDataProvider  = (*fakeItemData)(nil)
)
