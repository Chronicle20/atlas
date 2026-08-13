package handler

import (
	"atlas-channel/data/cash"
	"atlas-channel/remotemerchant"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// remoteMerchantNoopWP is a writer.Producer that satisfies any writer name
// with an encoder that discards its output. Rejection paths in
// handleRemoteMerchantUse call session.EnableActions, which invokes the
// producer even when the resulting bytes are never inspected — a nil
// producer panics on that call, so these tests need a real (if trivial) one.
func remoteMerchantNoopWP(_ string) (swriter.BodyFunc, error) {
	return func(_ logrus.FieldLogger, _ context.Context) func(encoder packet.Encode) []byte {
		return func(_ packet.Encode) []byte { return nil }
	}, nil
}

// newRemoteMerchantRejectionTestSession is newCashItemUseTestSessionForVersion
// with a live discardConn instead of nil, because session.EnableActions
// writes an encrypted announce straight to the session's connection.
func newRemoteMerchantRejectionTestSession(t *testing.T, characterId uint32, region string, major uint16) (session.Model, context.Context, func()) {
	t.Helper()
	ten := mustTenant(t, region, major, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, discardConn{})
	session.AddSessionToRegistry(ten.Id(), s)

	sp := session.NewProcessor(logrus.New(), ctx)
	sp.SetCharacterId(sessionId, characterId)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	return updated, ctx, func() { session.ClearRegistryForTenant(ten.Id()) }
}

var _ writer.Producer = remoteMerchantNoopWP

// installCashItemDataSeam swaps the atlas-data cash-item read.
func installCashItemDataSeam(t *testing.T, m cash.RestModel, err error) func() {
	t.Helper()
	orig := cashItemDataFunc
	cashItemDataFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (cash.RestModel, error) {
		return m, err
	}
	return func() { cashItemDataFunc = orig }
}

// installRemoteMerchantSagaSeam records created sagas instead of producing.
func installRemoteMerchantSagaSeam(t *testing.T) (*[]saga.Saga, func()) {
	t.Helper()
	var got []saga.Saga
	orig := remoteMerchantSagaCreateFunc
	remoteMerchantSagaCreateFunc = func(_ logrus.FieldLogger, _ context.Context, s saga.Saga) error {
		got = append(got, s)
		return nil
	}
	return &got, func() { remoteMerchantSagaCreateFunc = orig }
}

func TestRemoteMerchantEnabled_MatchesTheIdbEvidenceMatrix(t *testing.T) {
	cases := []struct {
		region string
		major  uint16
		want   bool
	}{
		// Design §1.3. The three disabled GMS builds have no send arm for the
		// cash-slot type, so a request can only come from a crafted packet.
		{"GMS", 12, false},
		{"GMS", 48, false},
		{"GMS", 61, false},
		{"GMS", 72, true},
		{"GMS", 79, true},
		{"GMS", 83, true},
		{"GMS", 84, true},
		{"GMS", 87, true},
		{"GMS", 92, true},
		{"GMS", 95, true},
		// JMS maps classification 545 to cash-slot type 36, not 37/38, and this
		// task does not enable it (design §7.3).
		{"JMS", 185, false},
	}
	for _, c := range cases {
		ten := mustTenant(t, c.region, c.major, 1)
		if got := remoteMerchantEnabled(ten); got != c.want {
			t.Errorf("remoteMerchantEnabled(%s %d) = %v, want %v", c.region, c.major, got, c.want)
		}
	}
}

func TestRemoteMerchantCashSlotType_MatchesGetCashSlotItemType(t *testing.T) {
	if got := remoteMerchantCashSlotType(mustTenant(t, "GMS", 83, 1)); got != CashSlotItemType(37) {
		t.Errorf("GMS 83 = %d, want 37", got)
	}
	if got := remoteMerchantCashSlotType(mustTenant(t, "GMS", 95, 1)); got != CashSlotItemType(38) {
		t.Errorf("GMS 95 = %d, want 38", got)
	}
}

func TestHandleRemoteMerchantUse_HappyPathRegistersAndCreatesSaga(t *testing.T) {
	const itemId = uint32(5450000)
	const charId = uint32(555)
	const srcSlot = int16(3)

	restoreSlot := installCashItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	restoreData := installCashItemDataSeam(t, cash.RestModel{Id: itemId, Npc: 9090000}, nil)
	defer restoreData()
	sagas, restoreSaga := installRemoteMerchantSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, charId)
	defer cleanup()

	req := request.Request(cashItemUsePrefix(srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

	if len(*sagas) != 1 {
		t.Fatalf("sagas created = %d, want 1", len(*sagas))
	}
	sg := (*sagas)[0]
	if sg.SagaType != saga.RemoteMerchant {
		t.Errorf("SagaType = %q, want %q", sg.SagaType, saga.RemoteMerchant)
	}
	if len(sg.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(sg.Steps))
	}
	if sg.Steps[0].Action != saga.OpenNpcShop {
		t.Errorf("step 0 action = %q, want open_npc_shop", sg.Steps[0].Action)
	}
	op, ok := sg.Steps[0].Payload.(saga.OpenNpcShopPayload)
	if !ok || op.NpcTemplateId != 9090000 || op.CharacterId != charId {
		t.Errorf("step 0 payload = %+v", sg.Steps[0].Payload)
	}
	if sg.Steps[1].Action != saga.DestroyAssetFromSlot {
		t.Errorf("step 1 action = %q, want destroy_asset_from_slot", sg.Steps[1].Action)
	}
	dp, ok := sg.Steps[1].Payload.(saga.DestroyAssetFromSlotPayload)
	if !ok || dp.InventoryType != 5 || dp.Slot != srcSlot || dp.Quantity != 1 || dp.TemplateId != itemId {
		t.Errorf("step 1 payload = %+v", sg.Steps[1].Payload)
	}

	// The pending unlock must be registered BEFORE the saga is created, or a
	// fast ENTERED could arrive with nothing to match.
	ten := tenant.MustFromContext(ctx)
	if _, ok := remotemerchant.GetRegistry().Take(ten, charId); !ok {
		t.Error("no pending unlock registered")
	}
}

func TestHandleRemoteMerchantUse_DisabledVersionDoesNotCreateSaga(t *testing.T) {
	const itemId = uint32(5450000)
	restoreSlot := installCashItemInSlotSeam(t, 3, itemId)
	defer restoreSlot()
	restoreData := installCashItemDataSeam(t, cash.RestModel{Id: itemId, Npc: 9090000}, nil)
	defer restoreData()
	sagas, restoreSaga := installRemoteMerchantSagaSeam(t)
	defer restoreSaga()

	// gms_61: case 37 sits in SendConsumeCashItemUseRequest's default list
	// (@0x832af3) — no client emits this.
	s, ctx, cleanup := newRemoteMerchantRejectionTestSession(t, 777, "GMS", 61)
	defer cleanup()

	req := request.Request(cashItemUsePrefix(3, itemId))
	reader := request.NewRequestReader(&req, 0)
	CharacterCashItemUseHandleFunc(logrus.New(), ctx, remoteMerchantNoopWP)(s, &reader, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 on a disabled version", len(*sagas))
	}
}

func TestHandleRemoteMerchantUse_TicketNeverConsumes(t *testing.T) {
	const itemId = uint32(5451000)
	restoreSlot := installCashItemInSlotSeam(t, 3, itemId)
	defer restoreSlot()
	sagas, restoreSaga := installRemoteMerchantSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newRemoteMerchantRejectionTestSession(t, 888, "GMS", 83)
	defer cleanup()

	req := request.Request(cashItemUsePrefix(3, itemId))
	reader := request.NewRequestReader(&req, 0)
	CharacterCashItemUseHandleFunc(logrus.New(), ctx, remoteMerchantNoopWP)(s, &reader, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 — no client build emits the remote gachapon ticket", len(*sagas))
	}
}

func TestHandleRemoteMerchantUse_DataErrorDoesNotCreateSaga(t *testing.T) {
	const itemId = uint32(5450000)
	restoreSlot := installCashItemInSlotSeam(t, 3, itemId)
	defer restoreSlot()
	restoreData := installCashItemDataSeam(t, cash.RestModel{}, errors.New("boom"))
	defer restoreData()
	sagas, restoreSaga := installRemoteMerchantSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newRemoteMerchantRejectionTestSession(t, 999, "GMS", 83)
	defer cleanup()

	req := request.Request(cashItemUsePrefix(3, itemId))
	reader := request.NewRequestReader(&req, 0)
	CharacterCashItemUseHandleFunc(logrus.New(), ctx, remoteMerchantNoopWP)(s, &reader, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 when atlas-data fails", len(*sagas))
	}
}

func TestHandleRemoteMerchantUse_ZeroNpcDoesNotCreateSaga(t *testing.T) {
	const itemId = uint32(5450000)
	restoreSlot := installCashItemInSlotSeam(t, 3, itemId)
	defer restoreSlot()
	restoreData := installCashItemDataSeam(t, cash.RestModel{Id: itemId, Npc: 0}, nil)
	defer restoreData()
	sagas, restoreSaga := installRemoteMerchantSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newRemoteMerchantRejectionTestSession(t, 1001, "GMS", 83)
	defer cleanup()

	req := request.Request(cashItemUsePrefix(3, itemId))
	reader := request.NewRequestReader(&req, 0)
	CharacterCashItemUseHandleFunc(logrus.New(), ctx, remoteMerchantNoopWP)(s, &reader, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 when the item resolves to npc 0", len(*sagas))
	}
}
