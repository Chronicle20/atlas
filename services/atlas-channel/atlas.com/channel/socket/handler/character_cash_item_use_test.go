package handler

import (
	"atlas-channel/session"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func mustTenant(t *testing.T, region string, major uint16, minor uint16) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return m
}

// installCashItemInSlotSeam swaps cashItemInSlotFunc for the test and returns
// a restore func (precedent: installItemInSlotSeam in teleport_rock_use_test.go).
func installCashItemInSlotSeam(t *testing.T, matchSlot int16, matchTemplateId uint32) func() {
	t.Helper()
	orig := cashItemInSlotFunc
	cashItemInSlotFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, slot int16) (uint32, error) {
		if slot != matchSlot {
			return 0, nil
		}
		return matchTemplateId, nil
	}
	return func() {
		cashItemInSlotFunc = orig
	}
}

// newCashItemUseTestSession builds a v83 GMS session + matching tenant ctx
// (v83 so CharacterCashItemUseHandleFunc's updateTimeFirst gate resolves
// false, matching the raw payload shapes below — same pattern as
// newTeleportRockUseTestSession in teleport_rock_use_test.go, but this one
// also returns the ctx since the cash handler resolves the tenant from it).
func newCashItemUseTestSession(t *testing.T, characterId uint32) (session.Model, context.Context, func()) {
	t.Helper()
	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)

	sp := session.NewProcessor(logrus.New(), ctx)
	sp.SetCharacterId(sessionId, characterId)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	return updated, ctx, func() { session.ClearRegistryForTenant(ten.Id()) }
}

// cashRockSlot is the fixed slot position used by the teleport-rock
// disambiguation tests below.
const cashRockSlot = int16(2)

// cashItemUsePrefix encodes the common v83 cashsb.ItemUse prefix (no leading
// updateTime — that only appears from GMS v87 onward, see
// character_cash_item_use.go:42): int16 source (slot), then uint32 itemId.
func cashItemUsePrefix(slot int16, itemId uint32) []byte {
	return []byte{
		byte(slot), byte(slot >> 8),
		byte(itemId), byte(itemId >> 8), byte(itemId >> 16), byte(itemId >> 24),
	}
}

func TestCharacterCashItemUseHandleFunc_Rock5040000InvokesUseRock(t *testing.T) {
	const itemId = uint32(5040000)
	restoreSlot := installCashItemInSlotSeam(t, cashRockSlot, itemId)
	defer restoreSlot()
	calls, restoreUse := installUseRockSeam(t)
	defer restoreUse()

	s, ctx, cleanup := newCashItemUseTestSession(t, 555)
	defer cleanup()

	raw := append(cashItemUsePrefix(cashRockSlot, itemId),
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // targetMap = 100000000
		0x2A, 0x00, 0x00, 0x00, // updateTime = 42
	)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	handlerFunc := CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)
	handlerFunc(s, &reader, map[string]interface{}{})

	if len(*calls) != 1 {
		t.Fatalf("useRockFunc call count = %d, want 1", len(*calls))
	}
	if (*calls)[0].itemId != item.Id(itemId) {
		t.Fatalf("useRockFunc itemId = %d, want %d", (*calls)[0].itemId, itemId)
	}
	if (*calls)[0].target.ByName() || (*calls)[0].target.TargetMap() != 100000000 {
		t.Fatalf("useRockFunc target = %+v, want map target 100000000", (*calls)[0].target)
	}
}

func TestCharacterCashItemUseHandleFunc_Rock5041000InvokesUseRock(t *testing.T) {
	const itemId = uint32(5041000)
	restoreSlot := installCashItemInSlotSeam(t, cashRockSlot, itemId)
	defer restoreSlot()
	calls, restoreUse := installUseRockSeam(t)
	defer restoreUse()

	s, ctx, cleanup := newCashItemUseTestSession(t, 555)
	defer cleanup()

	raw := append(cashItemUsePrefix(cashRockSlot, itemId),
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // targetMap = 100000000
		0x2A, 0x00, 0x00, 0x00, // updateTime = 42
	)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	handlerFunc := CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)
	handlerFunc(s, &reader, map[string]interface{}{})

	if len(*calls) != 1 {
		t.Fatalf("useRockFunc call count = %d, want 1", len(*calls))
	}
	if (*calls)[0].itemId != item.Id(itemId) {
		t.Fatalf("useRockFunc itemId = %d, want %d", (*calls)[0].itemId, itemId)
	}
}

// TestCharacterCashItemUseHandleFunc_MegaphoneEnum12NotInvoked verifies the
// megaphone-preservation guard: 5071000 is classification 507
// (ClassificationMegaphones) and GetCashSlotItemType's ClassificationMegaphones
// branch maps otherCategory==(itemId%10000)/1000==1 to the SAME enum 12 as
// teleport rocks (character_cash_item_use.go GetCashSlotItemType, category ==
// item.ClassificationMegaphones -> otherCategory == 1 -> CashSlotItemType(12)).
// Confirmed: item.GetClassification(5071000) == 507 == item.ClassificationMegaphones,
// and (5071000 % 10000) / 1000 == 1. Because the handler's enum-12 branch
// additionally gates on item.GetClassification(itemId) ==
// item.ClassificationTeleportRock (504), this megaphone must fall through to
// the warn-and-drop path instead of invoking useRockFunc.
func TestCharacterCashItemUseHandleFunc_MegaphoneEnum12NotInvoked(t *testing.T) {
	const itemId = uint32(5071000)
	if item.GetClassification(item.Id(itemId)) != item.ClassificationMegaphones {
		t.Fatalf("test fixture invalid: GetClassification(%d) = %d, want ClassificationMegaphones (507)", itemId, item.GetClassification(item.Id(itemId)))
	}
	restoreSlot := installCashItemInSlotSeam(t, cashRockSlot, itemId)
	defer restoreSlot()
	calls, restoreUse := installUseRockSeam(t)
	defer restoreUse()

	s, ctx, cleanup := newCashItemUseTestSession(t, 555)
	defer cleanup()

	// Confirm GetCashSlotItemType actually resolves this id to enum 12 under
	// the tenant this test uses, so the disambiguation is exercised for real.
	ten := tenant.MustFromContext(ctx)
	if got := GetCashSlotItemType(ten)(item.Id(itemId)); got != CashSlotItemTypeTeleportRock {
		t.Fatalf("test fixture invalid: GetCashSlotItemType(%d) = %d, want enum 12 (CashSlotItemTypeTeleportRock)", itemId, got)
	}

	raw := cashItemUsePrefix(cashRockSlot, itemId)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	handlerFunc := CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)
	handlerFunc(s, &reader, map[string]interface{}{})

	if len(*calls) != 0 {
		t.Fatalf("useRockFunc call count = %d, want 0 for megaphone enum-12 alias (classification %d, not teleport-rock 504)", len(*calls), item.ClassificationMegaphones)
	}
}

func TestCharacterCashItemUseHandleFunc_RockAbsentTargetNotInvoked(t *testing.T) {
	const itemId = uint32(5040000)
	restoreSlot := installCashItemInSlotSeam(t, cashRockSlot, itemId)
	defer restoreSlot()
	calls, restoreUse := installUseRockSeam(t)
	defer restoreUse()

	s, ctx, cleanup := newCashItemUseTestSession(t, 555)
	defer cleanup()

	// Only the trailing updateTime remains — no target payload (mirrors
	// TestTeleportRockUseHandleFunc_AbsentTargetNotInvoked's fixture shape).
	raw := append(cashItemUsePrefix(cashRockSlot, itemId),
		0x2A, 0x00, 0x00, 0x00, // updateTime = 42
	)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	handlerFunc := CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)
	handlerFunc(s, &reader, map[string]interface{}{})

	if len(*calls) != 0 {
		t.Fatalf("useRockFunc call count = %d, want 0 on absent target payload", len(*calls))
	}
}

// TestGetCashSlotItemType_SealingLock_VersionShifted pins the version-shifted
// numbers GetCashSlotItemType returns for a 5061xxx sealing-lock imprint
// (classification 506 = item.ClassificationItemImprints): 64 on non-GMS95,
// 65 on GMS95 (character_cash_item_use.go ~line 397-407).
func TestGetCashSlotItemType_SealingLock_VersionShifted(t *testing.T) {
	sealingLockItemId := item.Id(5061000)

	nonGMS95 := mustTenant(t, "GMS", 87, 1)
	if got := GetCashSlotItemType(nonGMS95)(sealingLockItemId); got != CashSlotItemTypeSealTimed {
		t.Errorf("non-GMS95 sealing lock: got %v, want CashSlotItemTypeSealTimed(%v)", got, CashSlotItemTypeSealTimed)
	}

	gms95 := mustTenant(t, "GMS", 95, 1)
	if got := GetCashSlotItemType(gms95)(sealingLockItemId); got != CashSlotItemTypeSealTimedV95 {
		t.Errorf("GMS95 sealing lock: got %v, want CashSlotItemTypeSealTimedV95(%v)", got, CashSlotItemTypeSealTimedV95)
	}
}

// TestGetCashSlotItemType_CrossVersionTypeCollision documents (does not fix -
// the fix lives in the dispatch `sealTimed` selection in
// CharacterCashItemUseHandleFunc) the exact collision that made the seal arm
// unconditionally matching CashSlotItemTypeSealTimed(64)/CashSlotItemTypeSealTimedV95(65)
// unsafe:
//   - non-GMS95: a ClassificationCharacterCreation item (5430xxx-5432xxx) also
//     returns type 65 (the GMS95 seal-timed number).
//   - GMS95: a category-552 item also returns type 64 (the non-GMS95
//     seal-timed number).
//
// GetCashSlotItemType itself is version-correct (each of these values is
// exactly what that version's client switch produces); the bug was the
// handler's seal-arm dispatch matching both numbers regardless of tenant
// version. Full dispatch-level coverage (proving the handler routes a
// CharacterCreation/552 item away from the seal saga) would require mocking
// session.Model, character2.Processor, cashData.Processor and saga.Processor
// end-to-end; that scaffolding does not exist for this handler today, so
// this test is scoped to the pure GetCashSlotItemType classification that
// the fix's version-selected `sealTimed` variable relies on. Noted as a
// dispatch-level coverage gap.
func TestGetCashSlotItemType_CrossVersionTypeCollision(t *testing.T) {
	nonGMS95 := mustTenant(t, "GMS", 87, 1)
	gms95 := mustTenant(t, "GMS", 95, 1)

	// itemId/1000 == 5431 -> "else" branch of ClassificationCharacterCreation
	// (itemId.Id is uint32; itemId/1000 must be >= 5431 or the subtraction
	// underflows and always takes the >1 branch).
	characterCreationItemId := item.Id(5431000)
	if got := GetCashSlotItemType(nonGMS95)(characterCreationItemId); got != CashSlotItemType(65) {
		t.Fatalf("non-GMS95 CharacterCreation item: got %v, want 65 (collides with CashSlotItemTypeSealTimedV95)", got)
	}
	if got := GetCashSlotItemType(gms95)(characterCreationItemId); got == CashSlotItemType(65) {
		t.Fatalf("GMS95 CharacterCreation item unexpectedly returned 65 too: %v", got)
	}

	// category 552.
	category552ItemId := item.Id(5520000)
	if got := GetCashSlotItemType(gms95)(category552ItemId); got != CashSlotItemType(64) {
		t.Fatalf("GMS95 category-552 item: got %v, want 64 (collides with CashSlotItemTypeSealTimed)", got)
	}
	if got := GetCashSlotItemType(nonGMS95)(category552ItemId); got == CashSlotItemType(64) {
		t.Fatalf("non-GMS95 category-552 item unexpectedly returned 64 too: %v", got)
	}
}

// TestIsPigmyEgg pins the incubatable Pigmy Egg id range (4170000-4170009)
// that the incubator arm re-validates server-side so a crafted request
// cannot sacrifice an arbitrary item.
func TestIsPigmyEgg(t *testing.T) {
	cases := map[item.Id]bool{
		4169999: false, 4170000: true, 4170005: true, 4170009: true, 4170010: false, 2000000: false,
	}
	for id, want := range cases {
		if got := isPigmyEgg(id); got != want {
			t.Errorf("isPigmyEgg(%d) = %v, want %v", id, got, want)
		}
	}
}

func TestGetCashSlotItemTypeVegasSpell(t *testing.T) {
	pre95 := mustTenant(t, "GMS", 83, 1)
	v95 := mustTenant(t, "GMS", 95, 1)
	jms := mustTenant(t, "JMS", 185, 1)

	cases := []struct {
		name string
		tn   tenant.Model
		id   item.Id
		want CashSlotItemType
	}{
		{"v83 vega 10", pre95, item.VegasSpell10, CashSlotItemTypeVegasSpellPre95},
		{"v83 vega 60", pre95, item.VegasSpell60, CashSlotItemTypeVegasSpellPre95},
		{"v95 vega 10", v95, item.VegasSpell10, CashSlotItemTypeVegasSpell95},
		{"v95 vega 60", v95, item.VegasSpell60, CashSlotItemTypeVegasSpell95},
		{"jms vega 10", jms, item.VegasSpell10, CashSlotItemTypeVegasSpellPre95},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := GetCashSlotItemType(tc.tn)(tc.id); got != tc.want {
				t.Errorf("GetCashSlotItemType(%d) = %d, want %d", tc.id, got, tc.want)
			}
		})
	}
}

// TestCashSlotItemTypeNote pins that Note items (5090000-family,
// classification 509) map to the named note slot type on every version.
func TestCashSlotItemTypeNote(t *testing.T) {
	for _, v := range []struct {
		region string
		major  uint16
	}{{"GMS", 83}, {"GMS", 84}, {"GMS", 87}, {"GMS", 95}, {"JMS", 185}} {
		tn := mustTenant(t, v.region, v.major, 1)
		if got := GetCashSlotItemType(tn)(item.Id(5090000)); got != CashSlotItemTypeNote {
			t.Errorf("%s v%d: got %d, want %d", v.region, v.major, got, CashSlotItemTypeNote)
		}
	}
}

// TestCharacterCashItemUseHandleFuncSymbol verifies the handler constructor
// returns a non-nil closure. The constructor calls tenant.MustFromContext,
// so a tenant context is required; a nil writer.Producer is acceptable for
// the symbol check only.
func TestCharacterCashItemUseHandleFuncSymbol(t *testing.T) {
	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	if got := CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil); got == nil {
		t.Fatal("nil closure")
	}
}

// morphCouponCall records one forwarded consume request.
type morphCouponCall struct {
	itemId     item.Id
	source     slot.Position
	quantity   int16
	updateTime uint32
}

// installRequestItemConsumeSeam swaps requestItemConsumeFunc for the test
// (precedent: installUseRockSeam, installCashItemInSlotSeam) so no Kafka broker
// is needed.
func installRequestItemConsumeSeam(t *testing.T) (*[]morphCouponCall, func()) {
	t.Helper()
	calls := make([]morphCouponCall, 0)
	orig := requestItemConsumeFunc
	requestItemConsumeFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error {
		calls = append(calls, morphCouponCall{itemId, source, quantity, updateTime})
		return nil
	}
	return &calls, func() { requestItemConsumeFunc = orig }
}

const cashMorphSlot = int16(3)

// TestCharacterCashItemUseHandleFunc_MorphCouponInvokesConsume: a v83 5300000
// request reaches the new arm and forwards the consume command with the trailing
// updateTime decoded (FR-4.1, FR-1.2). Before this task it fell through to the
// terminal warn and nothing happened at all.
func TestCharacterCashItemUseHandleFunc_MorphCouponInvokesConsume(t *testing.T) {
	const itemId = uint32(5300000)
	if item.GetClassification(item.Id(itemId)) != item.ClassificationTransformationCoupon {
		t.Fatalf("fixture invalid: GetClassification(%d) = %d, want 530", itemId, item.GetClassification(item.Id(itemId)))
	}
	restoreSlot := installCashItemInSlotSeam(t, cashMorphSlot, itemId)
	defer restoreSlot()
	calls, restoreConsume := installRequestItemConsumeSeam(t)
	defer restoreConsume()

	s, ctx, cleanup := newCashItemUseTestSession(t, 555)
	defer cleanup()

	// v83 trails updateTime in the sub-body (cashsb.UpdateTimeFirst false).
	raw := append(cashItemUsePrefix(cashMorphSlot, itemId),
		0x2A, 0x00, 0x00, 0x00, // updateTime = 42
	)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

	if len(*calls) != 1 {
		t.Fatalf("RequestItemConsume call count = %d, want 1", len(*calls))
	}
	c := (*calls)[0]
	if c.itemId != item.Id(itemId) {
		t.Errorf("itemId = %d, want %d", c.itemId, itemId)
	}
	if c.source != slot.Position(cashMorphSlot) {
		t.Errorf("source = %d, want %d", c.source, cashMorphSlot)
	}
	if c.quantity != 1 {
		t.Errorf("quantity = %d, want 1", c.quantity)
	}
	if c.updateTime != 42 {
		t.Errorf("updateTime = %d, want 42 (decoded from the trailing sub-body int32)", c.updateTime)
	}
}

// TestCharacterCashItemUseHandleFunc_MorphCouponV95NoSubBody: on GMS v95 the
// common ItemUse header already carried updateTime, so the sub-body reads
// nothing and the header value is forwarded (FR-1.2's leading half).
func TestCharacterCashItemUseHandleFunc_MorphCouponV95NoSubBody(t *testing.T) {
	const itemId = uint32(5300000)
	restoreSlot := installCashItemInSlotSeam(t, cashMorphSlot, itemId)
	defer restoreSlot()
	calls, restoreConsume := installRequestItemConsumeSeam(t)
	defer restoreConsume()

	ten := mustTenant(t, "GMS", 95, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	sessionId := uuid.New()
	sess := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), sess)
	defer session.ClearRegistryForTenant(ten.Id())
	sp := session.NewProcessor(logrus.New(), ctx)
	sp.SetCharacterId(sessionId, 555)
	s := sp.SetField(sessionId, field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build())

	// Leading updateTime, then the ItemUse prefix. No trailing bytes.
	raw := append([]byte{0x2A, 0x00, 0x00, 0x00}, cashItemUsePrefix(cashMorphSlot, itemId)...)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

	if len(*calls) != 1 {
		t.Fatalf("RequestItemConsume call count = %d, want 1", len(*calls))
	}
	if (*calls)[0].updateTime != 42 {
		t.Errorf("updateTime = %d, want 42 (from the leading header, not a sub-body read)", (*calls)[0].updateTime)
	}
}

// TestCharacterCashItemUseHandleFunc_MorphCouponTypeByteCollisions pins FR-1.3.
//
// The cash-slot type bytes collide ACROSS versions, not within one tenant
// (GetCashSlotItemType, character_cash_item_use.go): classification 522
// gachapon -> 40 on GMS >= 95; classification 530 transformation -> 40 on
// GMS < 95 and 41 on GMS >= 95; classification 538 pet evolution -> 41 on
// GMS < 95. So byte 40 means "transformation" on v83 and "gachapon" on v95,
// and byte 41 means "pet evolution" on v83 and "transformation" on v95. A
// type-byte-keyed arm would silently swap meaning at a version bump; the arm
// gates on classification instead, and neither collider may enter it.
func TestCharacterCashItemUseHandleFunc_MorphCouponTypeByteCollisions(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		major    uint16
		itemId   uint32
		wantType CashSlotItemType
	}{
		{"v95 gachapon coupon shares byte 40 with v83 transformation", "GMS", 95, 5220000, CashSlotItemType(40)},
		{"v83 pet evolution shares byte 41 with v95 transformation", "GMS", 83, 5380000, CashSlotItemType(41)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ten := mustTenant(t, tc.region, tc.major, 1)
			ctx := tenant.WithContext(context.Background(), ten)
			// Confirm the collision is real under this tenant, so the test is
			// exercising the disambiguation rather than a vacuous case.
			if got := GetCashSlotItemType(ten)(item.Id(tc.itemId)); got != tc.wantType {
				t.Fatalf("fixture invalid: GetCashSlotItemType(%d) = %d, want %d", tc.itemId, got, tc.wantType)
			}
			if item.GetClassification(item.Id(tc.itemId)) == item.ClassificationTransformationCoupon {
				t.Fatalf("fixture invalid: %d is classification 530", tc.itemId)
			}

			restoreSlot := installCashItemInSlotSeam(t, cashMorphSlot, tc.itemId)
			defer restoreSlot()
			calls, restoreConsume := installRequestItemConsumeSeam(t)
			defer restoreConsume()

			sessionId := uuid.New()
			sess := session.NewSession(sessionId, ten, 0, nil)
			session.AddSessionToRegistry(ten.Id(), sess)
			defer session.ClearRegistryForTenant(ten.Id())
			sp := session.NewProcessor(logrus.New(), ctx)
			sp.SetCharacterId(sessionId, 555)
			s := sp.SetField(sessionId, field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build())

			raw := append(cashItemUsePrefix(cashMorphSlot, tc.itemId), 0x2A, 0x00, 0x00, 0x00)
			if cashsb.UpdateTimeFirst(ten) {
				raw = append([]byte{0x2A, 0x00, 0x00, 0x00}, cashItemUsePrefix(cashMorphSlot, tc.itemId)...)
			}
			req := request.Request(raw)
			reader := request.NewRequestReader(&req, 0)

			CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

			if len(*calls) != 0 {
				t.Errorf("RequestItemConsume call count = %d, want 0 — classification %d must not enter the morph-coupon arm", len(*calls), item.GetClassification(item.Id(tc.itemId)))
			}
		})
	}
}

// TestCharacterCashItemUseHandleFunc_MorphCouponMismatchedSlotNotInvoked: the
// arm inherits the ownership check by position (FR-4.2), and a mismatch must
// send nothing — no consume, and no unlock, exactly as every neighbouring arm.
func TestCharacterCashItemUseHandleFunc_MorphCouponMismatchedSlotNotInvoked(t *testing.T) {
	const itemId = uint32(5300000)
	// The seam reports a different template id for this slot.
	restoreSlot := installCashItemInSlotSeam(t, cashMorphSlot, 5300002)
	defer restoreSlot()
	calls, restoreConsume := installRequestItemConsumeSeam(t)
	defer restoreConsume()

	s, ctx, cleanup := newCashItemUseTestSession(t, 555)
	defer cleanup()

	raw := append(cashItemUsePrefix(cashMorphSlot, itemId), 0x2A, 0x00, 0x00, 0x00)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

	if len(*calls) != 0 {
		t.Errorf("RequestItemConsume call count = %d, want 0 on a template-id mismatch", len(*calls))
	}
}
