package handler

import (
	"atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/inventory/compartment"
	"atlas-channel/cashshop/item"
	"atlas-channel/character"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	notecb "github.com/Chronicle20/atlas/libs/atlas-packet/note/clientbound"
	notesb "github.com/Chronicle20/atlas/libs/atlas-packet/note/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestBuildGiftForwardSaga pins the gift-forward saga's shape: exactly one
// step (no destroy step -- the note is paid for by the gift purchase, not a
// Note item), CreateNote with Flag 0.
func TestBuildGiftForwardSaga(t *testing.T) {
	txn := uuid.New()
	now := time.Now()
	s := buildGiftForwardSaga(txn, now, 200, 100, "thanks!")

	if s.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", s.TransactionId, txn)
	}
	if s.SagaType != saga.NoteSend {
		t.Errorf("sagaType: got %s, want %s", s.SagaType, saga.NoteSend)
	}
	if len(s.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1 (gift-forward destroys nothing)", len(s.Steps))
	}

	if s.Steps[0].Action != saga.CreateNote {
		t.Errorf("step action: got %s, want %s", s.Steps[0].Action, saga.CreateNote)
	}
	np, ok := s.Steps[0].Payload.(saga.CreateNotePayload)
	if !ok {
		t.Fatalf("step payload type: %T", s.Steps[0].Payload)
	}
	if np.SenderId != 200 || np.ReceiverId != 100 || np.Message != "thanks!" || np.Flag != 0 {
		t.Errorf("create-note payload mismatch: %+v", np)
	}
	if !np.GiftNote {
		t.Errorf("create-note payload: GiftNote = false, want true (fame already settled at acceptance)")
	}
}

// giftAsset builds a cash-shop compartment holding a single asset with the
// given cashId/giftFrom/giftNoteSent, for findGiftAsset and
// handleNoteGiftForward tests.
func giftAsset(t *testing.T, cashId int64, giftFrom string, giftNoteSent bool) compartment.Model {
	t.Helper()
	i, err := item.NewModelBuilder().SetId(1).SetCashId(cashId).SetTemplateId(5000).Build()
	if err != nil {
		t.Fatalf("item: %v", err)
	}
	a, err := asset.NewModelBuilder(1, uuid.New(), i).SetGiftFrom(giftFrom).SetGiftNoteSent(giftNoteSent).Build()
	if err != nil {
		t.Fatalf("asset: %v", err)
	}
	cp, err := compartment.NewModelBuilder(uuid.New(), 900, compartment.TypeExplorer, 100).AddAsset(a).Build()
	if err != nil {
		t.Fatalf("compartment: %v", err)
	}
	return cp
}

// TestFindGiftAsset_Matching pins the success case: an asset with the
// forwarded GiftSN carries the sender name, which the caller then compares
// against ToName.
func TestFindGiftAsset_Matching(t *testing.T) {
	cp := giftAsset(t, 10002321, "Gifter", false)

	giftFrom, giftNoteSent, found := findGiftAsset(cp, 10002321)
	if !found {
		t.Fatal("expected asset to be found by GiftSN")
	}
	if giftFrom != "Gifter" {
		t.Fatalf("giftFrom = %q, want %q", giftFrom, "Gifter")
	}
	if giftNoteSent {
		t.Fatal("expected giftNoteSent = false for a fresh asset")
	}
}

// TestFindGiftAsset_UnknownSN pins FR: a GiftSN the character does not hold
// must not resolve to any giftFrom.
func TestFindGiftAsset_UnknownSN(t *testing.T) {
	cp := giftAsset(t, 10002321, "Gifter", false)

	_, _, found := findGiftAsset(cp, 99999999)
	if found {
		t.Fatal("expected no asset to be found for an unowned GiftSN")
	}
}

// withNoteGiftForwardSeams overrides the compartment/character/saga/mark-sent
// test seams for the duration of the test, and returns recorders for every
// saga created (in call order) and whether MarkGiftNoteSent was called.
func withNoteGiftForwardSeams(t *testing.T, cp compartment.Model, gifterId uint32) (sagasCreated *[]saga.Saga, markSentCalled *bool) {
	t.Helper()
	sagasCreated = new([]saga.Saga)
	markSentCalled = new(bool)

	origCompartment := noteGiftForwardCompartmentFunc
	origCharacter := noteGiftForwardCharacterFunc
	origSaga := noteGiftForwardSagaCreateFunc
	origMarkSent := noteGiftForwardMarkSentFunc

	noteGiftForwardCompartmentFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (compartment.Model, error) {
		return cp, nil
	}
	noteGiftForwardCharacterFunc = func(_ logrus.FieldLogger, _ context.Context, _ string) (character.Model, error) {
		return character.Extract(character.RestModel{Id: gifterId})
	}
	noteGiftForwardSagaCreateFunc = func(_ logrus.FieldLogger, _ context.Context, s saga.Saga) error {
		*sagasCreated = append(*sagasCreated, s)
		return nil
	}
	noteGiftForwardMarkSentFunc = func(_ logrus.FieldLogger, _ context.Context, _ session.Model, _ int64) error {
		*markSentCalled = true
		return nil
	}

	t.Cleanup(func() {
		noteGiftForwardCompartmentFunc = origCompartment
		noteGiftForwardCharacterFunc = origCharacter
		noteGiftForwardSagaCreateFunc = origSaga
		noteGiftForwardMarkSentFunc = origMarkSent
	})
	return sagasCreated, markSentCalled
}

// TestHandleNoteGiftForward_GiftFromMismatch pins the anti-tamper gate for
// real: handleNoteGiftForward must create no saga and mark nothing sent when
// the asset's GiftFrom differs from the client-supplied ToName. Unlike the
// prior version of this test, this one fails if the gate is removed from
// handleNoteGiftForward.
func TestHandleNoteGiftForward_GiftFromMismatch(t *testing.T) {
	const characterId = uint32(778899)
	cp := giftAsset(t, 10002321, "ActualGifter", false)
	sagasCreated, markSentCalled := withNoteGiftForwardSeams(t, cp, 200)

	s, _, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	handleNoteGiftForward(logrus.New(), context.Background())(s, notesbOperationSend(t, "SomeoneElse", "hi", 10002321))

	if len(*sagasCreated) != 0 {
		t.Fatalf("expected no saga to be created on a giftFrom/toName mismatch, got %d", len(*sagasCreated))
	}
	if *markSentCalled {
		t.Fatal("expected no mark-sent call on a giftFrom/toName mismatch")
	}
}

// TestHandleNoteGiftForward_AlreadySent pins task-240 Defect I's gate: an
// asset whose GiftNoteSent is already true must not mint a second note, even
// though it passes the ownership gate.
func TestHandleNoteGiftForward_AlreadySent(t *testing.T) {
	const characterId = uint32(778899)
	cp := giftAsset(t, 10002321, "Gifter", true)
	sagasCreated, markSentCalled := withNoteGiftForwardSeams(t, cp, 200)

	s, _, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	handleNoteGiftForward(logrus.New(), context.Background())(s, notesbOperationSend(t, "Gifter", "hi", 10002321))

	if len(*sagasCreated) != 0 {
		t.Fatalf("expected no saga to be created when GiftNoteSent is already true, got %d", len(*sagasCreated))
	}
	if *markSentCalled {
		t.Fatal("expected no mark-sent call when GiftNoteSent is already true")
	}
}

// TestHandleNoteGiftForward_UnknownSN pins that a GiftSN the character does
// not hold (findGiftAsset's found == false) must not mint a note or fame
// saga.
func TestHandleNoteGiftForward_UnknownSN(t *testing.T) {
	const characterId = uint32(778899)
	cp := giftAsset(t, 10002321, "Gifter", false)
	sagasCreated, markSentCalled := withNoteGiftForwardSeams(t, cp, 200)

	s, _, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	handleNoteGiftForward(logrus.New(), context.Background())(s, notesbOperationSend(t, "Gifter", "hi", 99999999))

	if len(*sagasCreated) != 0 {
		t.Fatalf("expected no saga to be created for an unowned GiftSN, got %d", len(*sagasCreated))
	}
	if *markSentCalled {
		t.Fatal("expected no mark-sent call for an unowned GiftSN")
	}
}

// TestHandleNoteGiftForward_CompartmentLookupError pins that a failure to
// load the sender's cash-shop compartment must not mint a note or fame saga.
func TestHandleNoteGiftForward_CompartmentLookupError(t *testing.T) {
	const characterId = uint32(778899)
	cp := giftAsset(t, 10002321, "Gifter", false)
	sagasCreated, markSentCalled := withNoteGiftForwardSeams(t, cp, 200)

	origCompartment := noteGiftForwardCompartmentFunc
	noteGiftForwardCompartmentFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (compartment.Model, error) {
		return compartment.Model{}, errors.New("compartment lookup failed")
	}
	t.Cleanup(func() { noteGiftForwardCompartmentFunc = origCompartment })

	s, _, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	handleNoteGiftForward(logrus.New(), context.Background())(s, notesbOperationSend(t, "Gifter", "hi", 10002321))

	if len(*sagasCreated) != 0 {
		t.Fatalf("expected no saga to be created when the compartment lookup fails, got %d", len(*sagasCreated))
	}
	if *markSentCalled {
		t.Fatal("expected no mark-sent call when the compartment lookup fails")
	}
}

// TestHandleNoteGiftForward_Success pins the happy path: a matching,
// not-yet-sent gift asset creates exactly two sagas -- the note_send saga,
// then the award_fame saga targeting the gifter -- and marks the note sent.
func TestHandleNoteGiftForward_Success(t *testing.T) {
	const characterId = uint32(778899)
	const gifterId = uint32(200)
	cp := giftAsset(t, 10002321, "Gifter", false)
	sagasCreated, markSentCalled := withNoteGiftForwardSeams(t, cp, gifterId)

	s, _, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	handleNoteGiftForward(logrus.New(), context.Background())(s, notesbOperationSend(t, "Gifter", "hi", 10002321))

	if len(*sagasCreated) != 2 {
		t.Fatalf("expected 2 sagas to be created for a matching, unsent gift, got %d", len(*sagasCreated))
	}
	if (*sagasCreated)[0].SagaType != saga.NoteSend {
		t.Errorf("first saga type: got %s, want %s", (*sagasCreated)[0].SagaType, saga.NoteSend)
	}

	fameSaga := (*sagasCreated)[1]
	if fameSaga.SagaType != saga.InventoryTransaction {
		t.Errorf("second saga type: got %s, want %s", fameSaga.SagaType, saga.InventoryTransaction)
	}
	if len(fameSaga.Steps) != 1 || fameSaga.Steps[0].Action != saga.AwardFame {
		t.Fatalf("second saga steps: got %+v, want one award_fame step", fameSaga.Steps)
	}
	fp, ok := fameSaga.Steps[0].Payload.(saga.AwardFamePayload)
	if !ok {
		t.Fatalf("fame step payload type: %T", fameSaga.Steps[0].Payload)
	}
	if fp.CharacterId != gifterId || fp.Amount != 1 || fp.WorldId != s.WorldId() || fp.ChannelId != s.ChannelId() {
		t.Errorf("fame payload mismatch: %+v", fp)
	}

	if !*markSentCalled {
		t.Fatal("expected the gift's note to be marked sent")
	}
}

// TestHandleNoteGiftForward_SelfGift pins that a self-gift (gifter id equals
// the session's characterId) creates the note saga but skips the fame award
// -- mirroring atlas-notes' self-note skip.
func TestHandleNoteGiftForward_SelfGift(t *testing.T) {
	const characterId = uint32(778899)
	cp := giftAsset(t, 10002321, "Gifter", false)
	sagasCreated, markSentCalled := withNoteGiftForwardSeams(t, cp, characterId)

	s, _, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	handleNoteGiftForward(logrus.New(), context.Background())(s, notesbOperationSend(t, "Gifter", "hi", 10002321))

	if len(*sagasCreated) != 1 {
		t.Fatalf("expected 1 saga (note only, no fame) for a self-gift, got %d", len(*sagasCreated))
	}
	if (*sagasCreated)[0].SagaType != saga.NoteSend {
		t.Errorf("saga type: got %s, want %s", (*sagasCreated)[0].SagaType, saga.NoteSend)
	}
	if !*markSentCalled {
		t.Fatal("expected the gift's note to be marked sent")
	}
}

// TestBuildGiftFameSaga pins the fame-award saga's shape: exactly one
// award_fame step targeting the gifter, amount 1.
func TestBuildGiftFameSaga(t *testing.T) {
	txn := uuid.New()
	now := time.Now()
	sg := buildGiftFameSaga(txn, now, 200, world.Id(0), channel.Id(0))

	if sg.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", sg.TransactionId, txn)
	}
	if sg.SagaType != saga.InventoryTransaction {
		t.Errorf("sagaType: got %s, want %s", sg.SagaType, saga.InventoryTransaction)
	}
	if len(sg.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(sg.Steps))
	}
	if sg.Steps[0].Action != saga.AwardFame {
		t.Errorf("step action: got %s, want %s", sg.Steps[0].Action, saga.AwardFame)
	}
	fp, ok := sg.Steps[0].Payload.(saga.AwardFamePayload)
	if !ok {
		t.Fatalf("step payload type: %T", sg.Steps[0].Payload)
	}
	if fp.CharacterId != 200 || fp.Amount != 1 {
		t.Errorf("fame payload mismatch: %+v", fp)
	}
}

// notesbOperationSend builds a NOTE_ACTION SEND OperationSend value by
// decoding a v83 GMS giftFlag=1 wire packet through the real codec, for
// handleNoteGiftForward unit tests.
func notesbOperationSend(t *testing.T, toName string, message string, giftSN uint64) *notesb.OperationSend {
	t.Helper()
	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	req := noteActionSendPacket(t, toName, message, 1, giftSN)
	req.ReadByte() // consume the op byte NoteOperationHandleFunc already dispatched on

	sp := &notesb.OperationSend{}
	sp.Decode(logrus.New(), ctx)(req, nil)
	if sp.GiftSN() != giftSN {
		t.Fatalf("test setup error: decoded giftSN %d != requested %d", sp.GiftSN(), giftSN)
	}
	return sp
}

// noteRecorder is a fake writer.Producer that records every announced writer
// name + body.
type noteRecorder struct {
	announced []struct {
		writer string
		body   []byte
	}
}

func (r *noteRecorder) producer() writer.Producer {
	return func(name string) (swriter.BodyFunc, error) {
		return func(l logrus.FieldLogger, ctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(l, ctx)(noteDispatchOptions())
				r.announced = append(r.announced, struct {
					writer string
					body   []byte
				}{writer: name, body: b})
				return b
			}
		}, nil
	}
}

// noteDispatchOptions binds every code this suite's packets/bodies need:
// dispatch (SEND -> op 0) and the SEND_ERROR/NO_NOTE_ITEM answer the
// giftFlag==0 gate still emits on a cash-compartment read failure.
func noteDispatchOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			NoteOperationSend:             float64(0),
			notecb.NoteOperationSendError: float64(5),
		},
		"errors": map[string]interface{}{
			notecb.NoteSendErrorNoNoteItem: float64(3),
		},
	}
}

// noteActionSendPacket builds a v83 GMS NOTE_ACTION SEND wire packet: op(1)
// + toName + message + giftFlag(1) + giftIndex(4) + giftSN(8) -- v83 is past
// the v48 cutoff, so sendHasGiftDetails is true (operation_send.go).
func noteActionSendPacket(t *testing.T, toName string, message string, giftFlag byte, giftSN uint64) *request.Reader {
	t.Helper()
	w := response.NewWriter(logrus.New())
	w.WriteByte(0) // op: SEND
	w.WriteAsciiString(toName)
	w.WriteAsciiString(message)
	w.WriteByte(giftFlag)
	w.WriteInt(uint32(0))
	w.WriteLong(giftSN)
	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

// TestNoteActionSendGiftFlagZeroStillGatesOnNoteItem pins that giftFlag == 0
// (the tamper path; no legitimate client writes it) is completely unchanged
// by the gift-forward branch: it still reaches the Note-item ownership gate,
// which -- on a cash-compartment read failure here -- answers SEND_ERROR /
// NO_NOTE_ITEM, exactly as before this fix.
func TestNoteActionSendGiftFlagZeroStillGatesOnNoteItem(t *testing.T) {
	const characterId = uint32(778899)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &noteRecorder{}
	NoteOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, noteActionSendPacket(t, "Gifter", "hi", 0, 0), noteDispatchOptions())

	if len(rec.announced) != 1 {
		t.Fatalf("announced %d packets, want 1", len(rec.announced))
	}
	got := rec.announced[0]
	if got.writer != notecb.NoteOperationWriter {
		t.Fatalf("announced writer = %q, want %q", got.writer, notecb.NoteOperationWriter)
	}
	if len(got.body) < 2 {
		t.Fatalf("body too short: %#v", got.body)
	}
	if got.body[0] != 5 {
		t.Errorf("mode byte = %d, want 5 (SEND_ERROR)", got.body[0])
	}
	if got.body[1] != 3 {
		t.Errorf("error code = %d, want 3 (NO_NOTE_ITEM)", got.body[1])
	}
}

// TestNoteActionSendGiftFlagOneDoesNotHitNoteItemGate pins the fix itself:
// giftFlag == 1 must never reach the Note-item gate (it must not announce
// SEND_ERROR / NO_NOTE_ITEM even when the gift-forward lookup itself fails,
// e.g. an unresolvable cash-shop compartment), because the gift acknowledgement
// path announces nothing on any outcome (the client already showed SP_2713).
func TestNoteActionSendGiftFlagOneDoesNotHitNoteItemGate(t *testing.T) {
	const characterId = uint32(778899)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &noteRecorder{}
	NoteOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, noteActionSendPacket(t, "Gifter", "hi", 1, 0), noteDispatchOptions())

	if len(rec.announced) != 0 {
		t.Fatalf("announced %d packets, want 0 (gift-forward never announces)", len(rec.announced))
	}
}
