package handler

import (
	"atlas-channel/asset"
	"atlas-channel/compartment"
	dueyparcel "atlas-channel/parcel"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	parcelcb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound"
	parcelsb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// dueyRecvRecordedAnnounce is one (writer, resolved reason, kind) triple the
// test's wp captured — kind is only meaningful for PARCEL_REMOVED.
type dueyRecvRecordedAnnounce struct {
	writer string
	reason string
	kind   byte
}

// dueyRecvTestWP resolves every arm this table exercises to a distinct
// byte, mirroring dueySendTestWP. PARCEL_REMOVED carries its kind byte
// immediately after the mode byte (parcelcb.NewParcelRemoved), so the
// fixture also records byte[1] when present.
func dueyRecvTestWP(announced *[]dueyRecvRecordedAnnounce) writer.Producer {
	codes := map[string]interface{}{
		parcelcb.ParcelOperationIncorrectRequest:   float64(1),
		parcelcb.ParcelOperationRecvNoFreeSlots:    float64(2),
		parcelcb.ParcelOperationRecvUniqueConflict: float64(3),
		parcelcb.ParcelOperationParcelRemoved:      float64(4),
	}
	byCode := map[byte]string{
		1: parcelcb.ParcelOperationIncorrectRequest,
		2: parcelcb.ParcelOperationRecvNoFreeSlots,
		3: parcelcb.ParcelOperationRecvUniqueConflict,
		4: parcelcb.ParcelOperationParcelRemoved,
	}
	return func(name string) (swriter.BodyFunc, error) {
		return func(bl logrus.FieldLogger, bctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(bl, bctx)(map[string]interface{}{"operations": codes})
				reason := ""
				var kind byte
				if len(b) > 0 {
					reason = byCode[b[0]]
				}
				if reason == parcelcb.ParcelOperationParcelRemoved && len(b) > 5 {
					kind = b[5]
				}
				*announced = append(*announced, dueyRecvRecordedAnnounce{writer: name, reason: reason, kind: kind})
				return b
			}
		}, nil
	}
}

func newDueyRecvTestSession(t *testing.T, characterId uint32, worldId world.Id) (session.Model, context.Context, *int, func()) {
	t.Helper()
	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	closes := 0
	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, closeCountingConn{closes: &closes})
	session.AddSessionToRegistry(ten.Id(), s)

	sp := session.NewProcessor(logrus.New(), ctx)
	sp.SetCharacterId(sessionId, characterId)
	f := field.NewBuilder(worldId, channel.Id(0), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	return updated, ctx, &closes, func() { session.ClearRegistryForTenant(ten.Id()) }
}

// mustParcelModel builds a channel-side parcel.Model via Extract (the only
// constructor the package exposes) — mirrors dueySendFixture's asset.Model
// construction pattern.
func mustParcelModel(t *testing.T, id uuid.UUID, recipientId uint32, worldId byte, status string, receivableAt time.Time, mesoAmount uint32, itemId *uint32, itemType byte) dueyparcel.Model {
	t.Helper()
	m, err := dueyparcel.Extract(dueyparcel.RestModel{
		Id:           id.String(),
		WorldId:      worldId,
		RecipientId:  recipientId,
		MesoAmount:   mesoAmount,
		ItemId:       itemId,
		ItemType:     itemType,
		Status:       status,
		ReceivableAt: receivableAt,
	})
	if err != nil {
		t.Fatalf("building parcel fixture: %v", err)
	}
	return m
}

func mustCompartmentModel(t *testing.T, characterId uint32, it inventory.Type, capacity uint32, assets []asset.Model) compartment.Model {
	t.Helper()
	m, err := compartment.NewBuilder(uuid.New(), characterId, it, capacity).SetAssets(assets).Build()
	if err != nil {
		t.Fatalf("building compartment fixture: %v", err)
	}
	return m
}

func mustAssetModel(t *testing.T, templateId uint32, slot int16) asset.Model {
	t.Helper()
	a, err := asset.NewBuilder(uuid.New(), templateId).SetId(1).SetSlot(slot).SetQuantity(1).Build()
	if err != nil {
		t.Fatalf("building asset fixture: %v", err)
	}
	return a
}

func decodeActionReceive(parcelId uint32) *parcelsb.ActionReceive {
	l := logrus.New()
	l.SetOutput(io.Discard)
	buf := []byte{byte(parcelId), byte(parcelId >> 8), byte(parcelId >> 16), byte(parcelId >> 24)}
	req := request.Request(buf)
	r := request.NewRequestReader(&req, 0)
	sp := &parcelsb.ActionReceive{}
	sp.Decode(l, context.Background())(&r, map[string]interface{}{})
	return sp
}

func decodeActionDiscard(parcelId uint32) *parcelsb.ActionDiscard {
	l := logrus.New()
	l.SetOutput(io.Discard)
	buf := []byte{byte(parcelId), byte(parcelId >> 8), byte(parcelId >> 16), byte(parcelId >> 24)}
	req := request.Request(buf)
	r := request.NewRequestReader(&req, 0)
	sp := &parcelsb.ActionDiscard{}
	sp.Decode(l, context.Background())(&r, map[string]interface{}{})
	return sp
}

func TestDueyActionReceive(t *testing.T) {
	itemId := uint32(1002140)
	pendingId := uuid.New()

	type expect struct {
		reason  string
		sagaLen int
		invType byte
	}

	cases := []struct {
		name    string
		parcel  func() dueyparcel.Model
		compFul bool // EQUIP compartment reports 0 free slots
		compDup bool // EQUIP compartment already holds itemId
		want    expect
	}{
		{
			name: "receive happy path",
			parcel: func() dueyparcel.Model {
				return mustParcelModel(t, pendingId, 100, 0, "", time.Now().Add(-time.Hour), 5000, &itemId, byte(inventory.TypeValueEquip))
			},
			want: expect{sagaLen: 1, invType: byte(inventory.TypeValueEquip)},
		},
		{
			name: "receive meso only",
			parcel: func() dueyparcel.Model {
				return mustParcelModel(t, pendingId, 100, 0, "", time.Now().Add(-time.Hour), 5000, nil, 0)
			},
			want: expect{sagaLen: 1, invType: 0},
		},
		{
			name: "no free slot",
			parcel: func() dueyparcel.Model {
				return mustParcelModel(t, pendingId, 100, 0, "", time.Now().Add(-time.Hour), 5000, &itemId, byte(inventory.TypeValueEquip))
			},
			compFul: true,
			want:    expect{reason: parcelcb.ParcelOperationRecvNoFreeSlots},
		},
		{
			name: "unique conflict",
			parcel: func() dueyparcel.Model {
				return mustParcelModel(t, pendingId, 100, 0, "", time.Now().Add(-time.Hour), 5000, &itemId, byte(inventory.TypeValueEquip))
			},
			compDup: true,
			want:    expect{reason: parcelcb.ParcelOperationRecvUniqueConflict},
		},
		{
			name: "not receivable yet",
			parcel: func() dueyparcel.Model {
				return mustParcelModel(t, pendingId, 100, 0, "", time.Now().Add(time.Hour), 5000, &itemId, byte(inventory.TypeValueEquip))
			},
			want: expect{reason: parcelcb.ParcelOperationIncorrectRequest},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, ctx, closes, cleanup := newDueyRecvTestSession(t, 100, world.Id(0))
			defer cleanup()

			var announced []dueyRecvRecordedAnnounce
			wp := dueyRecvTestWP(&announced)

			p := tc.parcel()

			var sagas []saga.Saga
			deps := dueyReceiveDeps{
				getParcel: func(_ uint32, _ world.Id, _ uint32) (dueyparcel.Model, error) { return p, nil },
				getCompartment: func(_ uint32, it inventory.Type) (compartment.Model, error) {
					capacity := uint32(24)
					var assets []asset.Model
					if tc.compFul {
						capacity = 1
						assets = []asset.Model{mustAssetModel(t, 9999999, 0)}
					}
					if tc.compDup {
						assets = append(assets, mustAssetModel(t, itemId, int16(len(assets))))
					}
					return mustCompartmentModel(t, 100, it, capacity, assets), nil
				},
				createSaga: func(sg saga.Saga) error {
					sagas = append(sagas, sg)
					return nil
				},
			}

			sp := decodeActionReceive(1)
			receiveParcel(logrus.New(), ctx, wp, s, sp, deps)

			if len(sagas) != tc.want.sagaLen {
				t.Fatalf("sagas created = %d, want %d", len(sagas), tc.want.sagaLen)
			}
			if tc.want.sagaLen > 0 {
				sg := sagas[0]
				if sg.SagaType != saga.ParcelReceive {
					t.Errorf("SagaType = %q, want %q", sg.SagaType, saga.ParcelReceive)
				}
				if len(sg.Steps) != 2 {
					t.Fatalf("steps = %d, want 2 (%+v)", len(sg.Steps), sg.Steps)
				}
				if sg.Steps[0].Action != saga.WithdrawFromParcel {
					t.Errorf("step 1 action = %q, want %q", sg.Steps[0].Action, saga.WithdrawFromParcel)
				}
				wp0, ok := sg.Steps[0].Payload.(saga.WithdrawFromParcelPayload)
				if !ok {
					t.Fatalf("step 1 payload type = %T, want WithdrawFromParcelPayload", sg.Steps[0].Payload)
				}
				if wp0.ParcelId != p.Id() {
					t.Errorf("withdraw_from_parcel parcelId = %s, want %s", wp0.ParcelId, p.Id())
				}
				if wp0.InventoryType != tc.want.invType {
					t.Errorf("withdraw_from_parcel inventoryType = %d, want %d", wp0.InventoryType, tc.want.invType)
				}
				if sg.Steps[1].Action != saga.AwardMesos {
					t.Errorf("step 2 action = %q, want %q", sg.Steps[1].Action, saga.AwardMesos)
				}
				amp, ok := sg.Steps[1].Payload.(saga.AwardMesosPayload)
				if !ok {
					t.Fatalf("step 2 payload type = %T, want AwardMesosPayload", sg.Steps[1].Payload)
				}
				if amp.Amount != 5000 {
					t.Errorf("award_mesos amount = %d, want 5000", amp.Amount)
				}
			}

			if tc.want.reason == "" {
				if len(announced) != 0 {
					t.Errorf("announced = %+v, want none", announced)
				}
			} else {
				if len(announced) != 1 {
					t.Fatalf("announced = %+v, want exactly one", announced)
				}
				if announced[0].writer != parcelcb.ParcelWriter {
					t.Errorf("writer = %q, want %q", announced[0].writer, parcelcb.ParcelWriter)
				}
				if announced[0].reason != tc.want.reason {
					t.Errorf("reason = %q, want %q", announced[0].reason, tc.want.reason)
				}
			}

			// FR-15/FR-16: every rejection leaves the parcel pending in
			// atlas-parcel (this test never PATCHes it) and never closes
			// the session (NFR-5).
			if *closes != 0 {
				t.Errorf("session was closed %d times, want 0", *closes)
			}
		})
	}
}

func TestDueyActionDiscard(t *testing.T) {
	parcelId := uuid.New()
	p := mustParcelModel(t, parcelId, 100, 0, "", time.Now().Add(-time.Hour), 5000, nil, 0)

	type expect struct {
		reason  string
		kind    byte
		patched bool
	}

	cases := []struct {
		name       string
		getParcel  func(t *testing.T) (dueyparcel.Model, error)
		discardErr error
		want       expect
	}{
		{
			name:      "discard happy path",
			getParcel: func(_ *testing.T) (dueyparcel.Model, error) { return p, nil },
			want:      expect{reason: parcelcb.ParcelOperationParcelRemoved, kind: parcelcb.ParcelRemovedKindDiscarded, patched: true},
		},
		{
			name:      "discard not mine",
			getParcel: func(_ *testing.T) (dueyparcel.Model, error) { return dueyparcel.Model{}, errParcelNotResolved },
			want:      expect{reason: parcelcb.ParcelOperationIncorrectRequest},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, ctx, closes, cleanup := newDueyRecvTestSession(t, 100, world.Id(0))
			defer cleanup()

			var announced []dueyRecvRecordedAnnounce
			wp := dueyRecvTestWP(&announced)

			patchedIds := 0
			deps := dueyReceiveDeps{
				getParcel: func(_ uint32, _ world.Id, _ uint32) (dueyparcel.Model, error) { return tc.getParcel(t) },
				discardParcel: func(id uuid.UUID, recipientId uint32) (dueyparcel.Model, error) {
					patchedIds++
					if id != p.Id() {
						t.Errorf("discardParcel id = %s, want %s", id, p.Id())
					}
					if recipientId != 100 {
						t.Errorf("discardParcel recipientId = %d, want 100", recipientId)
					}
					return p, nil
				},
			}

			sp := decodeActionDiscard(1)
			discardParcel(logrus.New(), ctx, wp, s, sp, deps)

			wantPatches := 0
			if tc.want.patched {
				wantPatches = 1
			}
			if patchedIds != wantPatches {
				t.Errorf("PATCH calls = %d, want %d", patchedIds, wantPatches)
			}

			if len(announced) != 1 {
				t.Fatalf("announced = %+v, want exactly one", announced)
			}
			if announced[0].writer != parcelcb.ParcelWriter {
				t.Errorf("writer = %q, want %q", announced[0].writer, parcelcb.ParcelWriter)
			}
			if announced[0].reason != tc.want.reason {
				t.Errorf("reason = %q, want %q", announced[0].reason, tc.want.reason)
			}
			if tc.want.reason == parcelcb.ParcelOperationParcelRemoved && announced[0].kind != tc.want.kind {
				t.Errorf("kind = %d, want %d", announced[0].kind, tc.want.kind)
			}

			if *closes != 0 {
				t.Errorf("session was closed %d times, want 0", *closes)
			}
		})
	}
}

func TestDueyActionClose(t *testing.T) {
	s, ctx, closes, cleanup := newDueyRecvTestSession(t, 100, world.Id(0))
	defer cleanup()

	var announced []dueyRecvRecordedAnnounce
	wp := dueyRecvTestWP(&announced)

	handleDueyActionClose(logrus.New(), ctx, wp)(s, &parcelsb.ActionClose{})

	if len(announced) != 0 {
		t.Errorf("announced = %+v, want none", announced)
	}
	if *closes != 0 {
		t.Errorf("session was closed %d times, want 0", *closes)
	}
}
