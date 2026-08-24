package handler

import (
	sagaMsg "atlas-channel/kafka/message/saga"
	"atlas-channel/saga"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const (
	jukeboxItemId        = uint32(5100000)
	jukeboxSlot          = int16(4)
	jukeboxCharacterId   = uint32(777)
	jukeboxSoundLengthMs = uint32(45000)
)

// jukeboxUseRequest builds the real wire payload for a category-510 (song
// player) cash item use on GMS v83: the common cashsb.ItemUse prefix (slot,
// itemId -- no leading updateTime at v83) followed by the type-20 sub-body,
// which per ItemUseSongPlayer is one int32 sound length plus a trailing
// updateTime on GMS <= v84. Built with the same response.Writer primitives
// ItemUseSongPlayer.Encode uses internally, because its fields are private
// with no public constructor argument for them.
func jukeboxUseRequest(l logrus.FieldLogger, source int16, itemId uint32, soundLengthMs uint32) *request.Reader {
	w := response.NewWriter(l)
	w.WriteInt(soundLengthMs)
	w.WriteInt(0) // trailing updateTime, GMS v83 (updateTimeFirst == false)
	body := append(cashItemUsePrefix(source, itemId), w.Bytes()...)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

// TestJukeboxArmSuccessCreatesTwoStepSaga: consume first, play second -- a
// failed play compensates by restoring the song-player item rather than
// leaving the client's inventory silently short.
func TestJukeboxArmSuccessCreatesTwoStepSaga(t *testing.T) {
	restoreSlot := installCashItemInSlotSeam(t, jukeboxSlot, jukeboxItemId)
	defer restoreSlot()
	restoreEnv := newKiteCharacterServer(t, jukeboxCharacterId, "Chronicle", 0, 0)
	defer restoreEnv()

	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, jukeboxCharacterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	r := jukeboxUseRequest(logrus.New(), jukeboxSlot, jukeboxItemId, jukeboxSoundLengthMs)

	CharacterCashItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, r, map[string]interface{}{})

	if rec.calls != 0 {
		t.Fatalf("announced %d packets on the success path, want 0 (no announce on the success path)", rec.calls)
	}

	msgs := (*captured)[sagaMsg.EnvCommandTopic]
	if len(msgs) != 1 {
		t.Fatalf("saga commands emitted = %d, want exactly 1", len(msgs))
	}

	var got saga.Saga
	if err := json.Unmarshal(msgs[0].Value, &got); err != nil {
		t.Fatalf("unmarshal saga: %v", err)
	}

	if got.SagaType != saga.FieldEffectUse {
		t.Errorf("sagaType = %s, want %s", got.SagaType, saga.FieldEffectUse)
	}
	if got.InitiatedBy != "CASH_ITEM_USE" {
		t.Errorf("initiatedBy = %q, want %q", got.InitiatedBy, "CASH_ITEM_USE")
	}
	if len(got.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(got.Steps))
	}

	if got.Steps[0].Action != saga.DestroyAsset {
		t.Errorf("step 1 action = %s, want %s", got.Steps[0].Action, saga.DestroyAsset)
	}
	dp, ok := got.Steps[0].Payload.(saga.DestroyAssetPayload)
	if !ok {
		t.Fatalf("step 1 payload type = %T", got.Steps[0].Payload)
	}
	wantDestroy := saga.DestroyAssetPayload{
		CharacterId: jukeboxCharacterId,
		TemplateId:  jukeboxItemId,
		Quantity:    1,
		RemoveAll:   false,
	}
	if dp != wantDestroy {
		t.Errorf("destroy payload = %+v, want %+v", dp, wantDestroy)
	}

	if got.Steps[1].Action != saga.PlayJukebox {
		t.Errorf("step 2 action = %s, want %s", got.Steps[1].Action, saga.PlayJukebox)
	}
	pp, ok := got.Steps[1].Payload.(saga.PlayJukeboxPayload)
	if !ok {
		t.Fatalf("step 2 payload type = %T", got.Steps[1].Payload)
	}
	wantPlay := saga.PlayJukeboxPayload{
		WorldId:    0,
		ChannelId:  0,
		MapId:      100000000,
		Instance:   uuid.Nil,
		ItemId:     jukeboxItemId,
		PlayerName: "Chronicle",
		DurationMs: jukeboxSoundLengthMs,
	}
	if pp != wantPlay {
		t.Errorf("play payload = %+v, want %+v", pp, wantPlay)
	}
}

// TestJukeboxArmRejectsSlotTemplateMismatch: the slot holds a different
// template than the request claims -- the shared cashItemInSlotFunc gate
// (before dispatch even reaches the type-20 arm) must refuse.
func TestJukeboxArmRejectsSlotTemplateMismatch(t *testing.T) {
	restoreSlot := installCashItemInSlotSeam(t, jukeboxSlot, jukeboxItemId+1)
	defer restoreSlot()

	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, jukeboxCharacterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	r := jukeboxUseRequest(logrus.New(), jukeboxSlot, jukeboxItemId, jukeboxSoundLengthMs)

	CharacterCashItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, r, map[string]interface{}{})

	for topic, msgs := range *captured {
		if len(msgs) != 0 {
			t.Errorf("emitted %d commands on %q, want 0", len(msgs), topic)
		}
	}
	if rec.calls != 0 {
		t.Errorf("announced %d packets, want 0", rec.calls)
	}
}

// TestJukeboxArmRejectsZeroSoundLength: a zero-length sound would end the
// instant it started -- a broken or spoofed client. Reject before consuming.
func TestJukeboxArmRejectsZeroSoundLength(t *testing.T) {
	restoreSlot := installCashItemInSlotSeam(t, jukeboxSlot, jukeboxItemId)
	defer restoreSlot()
	restoreEnv := newKiteCharacterServer(t, jukeboxCharacterId, "Chronicle", 0, 0)
	defer restoreEnv()

	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, jukeboxCharacterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	r := jukeboxUseRequest(logrus.New(), jukeboxSlot, jukeboxItemId, 0)

	CharacterCashItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, r, map[string]interface{}{})

	for topic, msgs := range *captured {
		if len(msgs) != 0 {
			t.Errorf("emitted %d commands on %q, want 0", len(msgs), topic)
		}
	}
	if rec.calls != 0 {
		t.Errorf("announced %d packets, want 0", rec.calls)
	}
}

// TestJukeboxArmRejectsUnresolvableCharacter: PlayJukeboxPayload carries the
// starting player's name, resolved server-side -- an unresolvable character
// must refuse rather than emit a saga with an empty/wrong name.
func TestJukeboxArmRejectsUnresolvableCharacter(t *testing.T) {
	restoreSlot := installCashItemInSlotSeam(t, jukeboxSlot, jukeboxItemId)
	defer restoreSlot()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")

	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, jukeboxCharacterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	r := jukeboxUseRequest(logrus.New(), jukeboxSlot, jukeboxItemId, jukeboxSoundLengthMs)

	CharacterCashItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, r, map[string]interface{}{})

	for topic, msgs := range *captured {
		if len(msgs) != 0 {
			t.Errorf("emitted %d commands on %q, want 0", len(msgs), topic)
		}
	}
	if rec.calls != 0 {
		t.Errorf("announced %d packets, want 0", rec.calls)
	}
}
