package handler

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/compartment"
	dueyparcel "atlas-channel/parcel"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

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

// closeCountingConn is discardConn plus a Close() counter, so a rejection
// test can assert the session was NEVER torn down (NFR-5): Cosmic
// disconnects and autobans on the packet-edit cases this table exercises,
// Atlas rejects and logs.
type closeCountingConn struct {
	discardConn
	closes *int
}

func (c closeCountingConn) Close() error { *c.closes++; return nil }

func newDueySendTestSession(t *testing.T, characterId uint32, worldId world.Id) (session.Model, context.Context, *int, func()) {
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

// dueySendRecordedAnnounce is one (writer, resolved reject reason) pair the
// test's wp captured.
type dueySendRecordedAnnounce struct {
	writer string
	reason string
}

// dueySendTestWP resolves every PARCEL reject body to a DISTINCT byte, so
// the decoded mode maps back unambiguously to a reject reason — the same
// shape checkNameHandlerEnv.wp uses.
func dueySendTestWP(announced *[]dueySendRecordedAnnounce) writer.Producer {
	codes := map[string]interface{}{
		parcelcb.ParcelOperationIncorrectRequest:    float64(1),
		parcelcb.ParcelOperationNotEnoughMesos:      float64(2),
		parcelcb.ParcelOperationMesoLimit:           float64(3),
		parcelcb.ParcelOperationNameDoesNotExist:    float64(4),
		parcelcb.ParcelOperationSameAccount:         float64(5),
		parcelcb.ParcelOperationReceiverStorageFull: float64(6),
	}
	byCode := map[byte]string{
		1: parcelcb.ParcelOperationIncorrectRequest,
		2: parcelcb.ParcelOperationNotEnoughMesos,
		3: parcelcb.ParcelOperationMesoLimit,
		4: parcelcb.ParcelOperationNameDoesNotExist,
		5: parcelcb.ParcelOperationSameAccount,
		6: parcelcb.ParcelOperationReceiverStorageFull,
	}
	return func(name string) (swriter.BodyFunc, error) {
		return func(bl logrus.FieldLogger, bctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(bl, bctx)(map[string]interface{}{"operations": codes})
				reason := ""
				if len(b) > 0 {
					reason = byCode[b[0]]
				}
				*announced = append(*announced, dueySendRecordedAnnounce{writer: name, reason: reason})
				return b
			}
		}, nil
	}
}

func mustCharacterModel(t *testing.T, id uint32, accountId uint32, worldId world.Id, name string, level byte, meso uint32) character.Model {
	t.Helper()
	m, err := character.NewModelBuilder().SetId(id).SetAccountId(accountId).SetWorldId(worldId).SetName(name).SetLevel(level).SetMeso(meso).Build()
	if err != nil {
		t.Fatalf("building character fixture: %v", err)
	}
	return m
}

// dueySendFixture is the baseline from the plan's test table: sender 100 /
// account 1 / world 0 / level 30 / meso 1,000,000; recipient "Bob" resolves
// to character 200 / account 2 / world 0 with 0 pending parcels; the sender
// holds one Quick Delivery Ticket.
type dueySendFixture struct {
	sender              character.Model
	recipientCandidates []character.Model
	recipientErr        error
	pending             int
	pendingErr          error
	hasTicket           bool
	ticketErr           error
	item                asset.Model
	itemErr             error
	sagas               []saga.Saga
	sagaErr             error
}

func newDueySendFixture(t *testing.T) *dueySendFixture {
	t.Helper()
	sender := mustCharacterModel(t, 100, 1, world.Id(0), "Sender", 30, 1_000_000)
	recipient := mustCharacterModel(t, 200, 2, world.Id(0), "Bob", 30, 0)
	item, err := asset.NewBuilder(uuid.New(), 2000000).SetId(5).SetSlot(5).SetQuantity(1).Build()
	if err != nil {
		t.Fatalf("building item fixture: %v", err)
	}
	return &dueySendFixture{
		sender:              sender,
		recipientCandidates: []character.Model{recipient},
		hasTicket:           true,
		item:                item,
	}
}

func (f *dueySendFixture) deps() dueySendDeps {
	return dueySendDeps{
		getSender: func(_ uint32) (character.Model, error) { return f.sender, nil },
		resolveRecipient: func(_ string) ([]character.Model, error) {
			return f.recipientCandidates, f.recipientErr
		},
		getItem: func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
			return f.item, f.itemErr
		},
		hasTicket: func(_ uint32) (bool, error) {
			return f.hasTicket, f.ticketErr
		},
		countPending: func(_ uint32, _ world.Id) (int, error) {
			return f.pending, f.pendingErr
		},
		createSaga: func(sg saga.Saga) error {
			if f.sagaErr != nil {
				return f.sagaErr
			}
			f.sagas = append(f.sagas, sg)
			return nil
		},
	}
}

// newActionSendBytes builds the DUEY_ACTION SEND serverbound wire form
// directly (ActionSend's fields are unexported, so there is no constructor
// to call) — mirrors ActionSend.Encode byte-for-byte
// (libs/atlas-packet/parcel/serverbound/action_send.go).
func newActionSendBytes(inventoryType byte, slot uint16, quantity uint16, mesos uint32, recipientName string, quick bool, message string, ticketRef uint32) []byte {
	buf := make([]byte, 0, 64)
	buf = append(buf, inventoryType)
	buf = append(buf, byte(slot), byte(slot>>8))
	buf = append(buf, byte(quantity), byte(quantity>>8))
	buf = append(buf, byte(mesos), byte(mesos>>8), byte(mesos>>16), byte(mesos>>24))
	buf = append(buf, byte(len(recipientName)), byte(len(recipientName)>>8))
	buf = append(buf, []byte(recipientName)...)
	if quick {
		buf = append(buf, 1)
		buf = append(buf, byte(len(message)), byte(len(message)>>8))
		buf = append(buf, []byte(message)...)
		buf = append(buf, byte(ticketRef), byte(ticketRef>>8), byte(ticketRef>>16), byte(ticketRef>>24))
	} else {
		buf = append(buf, 0)
	}
	return buf
}

func decodeActionSend(raw []byte) *parcelsb.ActionSend {
	l := logrus.New()
	l.SetOutput(io.Discard)
	ctx := context.Background()
	req := request.Request(raw)
	r := request.NewRequestReader(&req, 0)
	sp := &parcelsb.ActionSend{}
	sp.Decode(l, ctx)(&r, map[string]interface{}{})
	return sp
}

func TestDueyActionSend(t *testing.T) {
	type expect struct {
		reason  string // "" if no announce is expected
		sagaLen int
		warn    bool

		// Saga step assertions, checked only when sagaLen > 0. quick,
		// message, mesoAmount, sourceInventoryType, assetId and quantity
		// are the inputs buildParcelSendSaga's steps are expected to carry
		// through unchanged.
		quick               bool
		message             string
		mesoAmount          uint32
		sourceInventoryType inventory.Type
		assetId             uint32
		quantity            uint32
	}

	cases := []struct {
		name   string
		mutate func(f *dueySendFixture)
		sp     []byte
		want   expect
	}{
		{
			name: "npc send item and meso",
			sp:   newActionSendBytes(byte(inventory.TypeValueUse), 5, 1, 1000, "Bob", false, "", 0),
			want: expect{sagaLen: 1, mesoAmount: 1000, sourceInventoryType: inventory.TypeValueUse, assetId: 5, quantity: 1},
		},
		{
			name: "quick send",
			sp:   newActionSendBytes(byte(inventory.TypeValueUse), 5, 1, 1000, "Bob", true, "hi", 0),
			want: expect{sagaLen: 1, quick: true, message: "hi", mesoAmount: 1000, sourceInventoryType: inventory.TypeValueUse, assetId: 5, quantity: 1},
		},
		{
			name: "meso only",
			sp:   newActionSendBytes(0, 0, 0, 1000, "Bob", false, "", 0),
			want: expect{sagaLen: 1, mesoAmount: 1000},
		},
		{
			name: "nothing attached",
			sp:   newActionSendBytes(0, 0, 0, 0, "Bob", false, "", 0),
			want: expect{reason: parcelcb.ParcelOperationIncorrectRequest},
		},
		{
			name:   "cannot afford",
			mutate: func(f *dueySendFixture) { f.sender = mustCharacterModel(t, 100, 1, world.Id(0), "Sender", 30, 100) },
			sp:     newActionSendBytes(0, 0, 0, 1000, "Bob", false, "", 0),
			want:   expect{reason: parcelcb.ParcelOperationNotEnoughMesos},
		},
		{
			name: "meso limit",
			mutate: func(f *dueySendFixture) {
				f.sender = mustCharacterModel(t, 100, 1, world.Id(0), "Sender", 15, 10_000_000)
			},
			sp:   newActionSendBytes(0, 0, 0, 1_000_001, "Bob", false, "", 0),
			want: expect{reason: parcelcb.ParcelOperationMesoLimit},
		},
		{
			name:   "unknown recipient",
			mutate: func(f *dueySendFixture) { f.recipientCandidates = nil },
			sp:     newActionSendBytes(0, 0, 0, 1000, "Nobody", false, "", 0),
			want:   expect{reason: parcelcb.ParcelOperationNameDoesNotExist},
		},
		{
			name: "recipient in another world",
			mutate: func(f *dueySendFixture) {
				f.recipientCandidates = []character.Model{mustCharacterModel(t, 200, 2, world.Id(1), "Bob", 30, 0)}
			},
			sp:   newActionSendBytes(0, 0, 0, 1000, "Bob", false, "", 0),
			want: expect{reason: parcelcb.ParcelOperationNameDoesNotExist},
		},
		{
			name: "same account",
			mutate: func(f *dueySendFixture) {
				f.recipientCandidates = []character.Model{mustCharacterModel(t, 200, 1, world.Id(0), "Bob", 30, 0)}
			},
			sp:   newActionSendBytes(0, 0, 0, 1000, "Bob", false, "", 0),
			want: expect{reason: parcelcb.ParcelOperationSameAccount},
		},
		{
			name:   "mailbox full",
			mutate: func(f *dueySendFixture) { f.pending = 10 },
			sp:     newActionSendBytes(0, 0, 0, 1000, "Bob", false, "", 0),
			want:   expect{reason: parcelcb.ParcelOperationReceiverStorageFull},
		},
		{
			name:   "mailbox at nine",
			mutate: func(f *dueySendFixture) { f.pending = 9 },
			sp:     newActionSendBytes(0, 0, 0, 1000, "Bob", false, "", 0),
			want:   expect{sagaLen: 1, mesoAmount: 1000},
		},
		{
			name:   "quick without a ticket",
			mutate: func(f *dueySendFixture) { f.hasTicket = false },
			sp:     newActionSendBytes(0, 0, 0, 1000, "Bob", true, "hi", 0),
			want:   expect{reason: parcelcb.ParcelOperationIncorrectRequest, warn: true},
		},
		{
			name: "message too long",
			sp:   newActionSendBytes(0, 0, 0, 1000, "Bob", true, strings.Repeat("x", 101), 0),
			want: expect{reason: parcelcb.ParcelOperationIncorrectRequest},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDueySendFixture(t)
			if tc.mutate != nil {
				tc.mutate(f)
			}

			s, ctx, closes, cleanup := newDueySendTestSession(t, 100, world.Id(0))
			defer cleanup()

			var announced []dueySendRecordedAnnounce
			wp := dueySendTestWP(&announced)

			logs := &bytes.Buffer{}
			l := logrus.New()
			l.SetOutput(logs)
			l.SetLevel(logrus.WarnLevel)

			sp := decodeActionSend(tc.sp)
			sendParcel(l, ctx, wp, s, sp, f.deps())

			if len(f.sagas) != tc.want.sagaLen {
				t.Errorf("sagas created = %d, want %d", len(f.sagas), tc.want.sagaLen)
			}
			if tc.want.sagaLen > 0 {
				sg := f.sagas[0]
				if sg.SagaType != saga.ParcelSend {
					t.Errorf("SagaType = %q, want %q", sg.SagaType, saga.ParcelSend)
				}

				// Step count and order: award_mesos first, an optional
				// destroy_asset only when quick, transfer_to_parcel last
				// (design §4.3, buildParcelSendSaga).
				wantStepCount := 2
				if tc.want.quick {
					wantStepCount = 3
				}
				if len(sg.Steps) != wantStepCount {
					t.Fatalf("steps = %d, want %d (%+v)", len(sg.Steps), wantStepCount, sg.Steps)
				}

				if sg.Steps[0].Action != saga.AwardMesos {
					t.Errorf("step 1 action = %q, want %q", sg.Steps[0].Action, saga.AwardMesos)
				}
				amp, ok := sg.Steps[0].Payload.(saga.AwardMesosPayload)
				if !ok {
					t.Fatalf("step 1 payload type = %T, want AwardMesosPayload", sg.Steps[0].Payload)
				}
				total, _ := dueyparcel.TotalCost(tc.want.mesoAmount, tc.want.quick)
				if amp.Amount != -int32(total) {
					t.Errorf("award mesos amount = %d, want %d", amp.Amount, -int32(total))
				}

				lastIdx := 1
				if tc.want.quick {
					if sg.Steps[1].Action != saga.DestroyAsset {
						t.Errorf("step 2 action = %q, want %q", sg.Steps[1].Action, saga.DestroyAsset)
					}
					if _, ok := sg.Steps[1].Payload.(saga.DestroyAssetPayload); !ok {
						t.Errorf("step 2 payload type = %T, want DestroyAssetPayload", sg.Steps[1].Payload)
					}
					lastIdx = 2
				} else {
					for _, st := range sg.Steps {
						if st.Action == saga.DestroyAsset {
							t.Errorf("non-quick send must not include a destroy_asset step, got: %+v", st)
						}
					}
				}

				if sg.Steps[lastIdx].Action != saga.TransferToParcel {
					t.Errorf("last step action = %q, want %q", sg.Steps[lastIdx].Action, saga.TransferToParcel)
				}
				tp, ok := sg.Steps[lastIdx].Payload.(saga.TransferToParcelPayload)
				if !ok {
					t.Fatalf("last step payload type = %T, want TransferToParcelPayload", sg.Steps[lastIdx].Payload)
				}
				if tp.AssetId != tc.want.assetId {
					t.Errorf("transfer_to_parcel assetId = %d, want %d", tp.AssetId, tc.want.assetId)
				}
				if tp.Quantity != tc.want.quantity {
					t.Errorf("transfer_to_parcel quantity = %d, want %d", tp.Quantity, tc.want.quantity)
				}
				if tp.Quick != tc.want.quick {
					t.Errorf("transfer_to_parcel quick = %v, want %v", tp.Quick, tc.want.quick)
				}
				if tp.Message != tc.want.message {
					t.Errorf("transfer_to_parcel message = %q, want %q", tp.Message, tc.want.message)
				}
				if tp.SourceInventoryType != tc.want.sourceInventoryType {
					t.Errorf("transfer_to_parcel sourceInventoryType = %d, want %d", tp.SourceInventoryType, tc.want.sourceInventoryType)
				}
				if tp.RecipientName != f.recipientCandidates[0].Name() {
					t.Errorf("transfer_to_parcel recipientName = %q, want %q", tp.RecipientName, f.recipientCandidates[0].Name())
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

			// NFR-5: every rejection subtest must leave the session open.
			if *closes != 0 {
				t.Errorf("session was closed %d times, want 0 (never disconnect on reject)", *closes)
			}

			if tc.want.warn && !strings.Contains(logs.String(), "Quick Delivery Ticket") {
				t.Errorf("expected a warn-level log about the missing ticket, got: %s", logs.String())
			}
		})
	}
}

// TestHasQuickDeliveryTicketQueriesCashCompartment pins the compartment
// type the PRODUCTION wiring requests, not a stubbed bool. Item 5330000 is
// classification 533, a cash item, so hasQuickDeliveryTicket must ask for
// TypeValueCash (5) — asking TypeValueETC (4), as a prior version of this
// lookup did, made every quick send unreachable in production even though
// dueySendFixture.hasTicket always stubbed true (task-241 bug).
func TestHasQuickDeliveryTicketQueriesCashCompartment(t *testing.T) {
	orig := dueyQuickTicketCompartmentFetch
	defer func() { dueyQuickTicketCompartmentFetch = orig }()

	var gotType inventory.Type
	var gotCharacterId uint32
	dueyQuickTicketCompartmentFetch = func(_ logrus.FieldLogger, _ context.Context, characterId uint32, it inventory.Type) (compartment.Model, error) {
		gotCharacterId = characterId
		gotType = it
		return compartment.Model{}, nil
	}

	l := logrus.New()
	l.SetOutput(io.Discard)
	ctx := context.Background()

	found, err := hasQuickDeliveryTicket(l, ctx, 100)
	if err != nil {
		t.Fatalf("hasQuickDeliveryTicket returned an error: %v", err)
	}
	if found {
		t.Errorf("found = true, want false (empty compartment.Model has no assets)")
	}
	if gotCharacterId != 100 {
		t.Errorf("GetByType characterId = %d, want 100", gotCharacterId)
	}
	if gotType != inventory.TypeValueCash {
		t.Errorf("GetByType inventoryType = %d, want %d (TypeValueCash)", gotType, inventory.TypeValueCash)
	}
}
