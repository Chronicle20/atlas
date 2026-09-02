package handler

import (
	kiteMsg "atlas-channel/kafka/message/kite"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// newKiteCharacterServer stands up a fake CHARACTERS_SERVICE_URL that resolves
// characterId to a single JSON:API "characters" resource carrying the given
// name/x/y, matching the RestModel shape in character/rest.go, and points the
// env var at it. The returned func must be deferred to restore the env/server.
func newKiteCharacterServer(t *testing.T, characterId uint32, name string, x, y int16) func() {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"characters","id":"` +
			jsonNum(characterId) + `","attributes":{"name":"` + name + `","x":` +
			jsonNum16(x) + `,"y":` + jsonNum16(y) + `}}}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")
	return srv.Close
}

func jsonNum(v uint32) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func jsonNum16(v int16) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// kiteUseRequest builds the actual wire payload for a category-508 (kite) cash
// item use: the common cashsb.ItemUse prefix (slot, itemId — no leading
// updateTime on GMS v83) followed by the type-18 sub-body, which per
// ItemUseKite's doc comment (libs/atlas-packet/cash/serverbound/item_use_kite.go)
// is exactly one length-prefixed ASCII string plus a trailing updateTime on
// GMS <= v84 (updateTimeFirst false). ItemUseKite's own Encode cannot be used
// here directly: its message field is private with no public constructor
// argument for it (NewItemUseKite only takes updateTimeFirst — see the task
// brief's note on this). So the sub-body bytes are built with the same
// response.Writer primitives ItemUseKite.Encode uses internally
// (WriteAsciiString then, since updateTimeFirst is false, a trailing
// WriteInt), which is the exact byte sequence the handler's real Decode call
// consumes -- this is not a hand-built struct, it is the real wire format.
func kiteUseRequest(l logrus.FieldLogger, source int16, itemId uint32, message string) *request.Reader {
	w := response.NewWriter(l)
	w.WriteAsciiString(message)
	w.WriteInt(0) // trailing updateTime, GMS v83 (updateTimeFirst == false)
	body := append(cashItemUsePrefix(source, itemId), w.Bytes()...)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

// The client sends no coordinates for a kite (the sub-body is the message
// alone) -- position and owner name must come from server-side character
// state.
func TestKiteUseEmitsCreateWithServerSidePosition(t *testing.T) {
	const characterId = uint32(42)
	const itemId = uint32(5080000)

	restoreEnv := newKiteCharacterServer(t, characterId, "Player", 320, -140)
	defer restoreEnv()

	restoreSlot := installCashItemInSlotSeam(t, cashRockSlot, itemId)
	defer restoreSlot()

	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	l := logrus.New()
	r := kiteUseRequest(l, cashRockSlot, itemId, "congrats!")

	handlerFunc := CharacterCashItemUseHandleFunc(l, ctx, nil)
	handlerFunc(s, r, map[string]interface{}{})

	msgs := (*captured)[string(kiteMsg.EnvCommandTopic)]
	if len(msgs) != 1 {
		t.Fatalf("emitted %d kite commands, want exactly 1", len(msgs))
	}
	var cmd kiteMsg.Command[kiteMsg.CreateCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != kiteMsg.CommandKiteCreate {
		t.Errorf("type = %s, want CREATE", cmd.Type)
	}
	if cmd.CharacterId != characterId {
		t.Errorf("characterId = %d, want %d", cmd.CharacterId, characterId)
	}
	if cmd.Body.X != 320 || cmd.Body.Y != -140 {
		t.Errorf("position = (%d,%d), want the character's (320,-140)", cmd.Body.X, cmd.Body.Y)
	}
	if cmd.Body.Name != "Player" {
		t.Errorf("name = %q, want the character's %q", cmd.Body.Name, "Player")
	}
	if cmd.Body.Message != "congrats!" {
		t.Errorf("message = %q, want %q", cmd.Body.Message, "congrats!")
	}
	if cmd.Body.TemplateId != itemId {
		t.Errorf("templateId = %d, want %d", cmd.Body.TemplateId, itemId)
	}
}

// FR-4.1: the kite item is deliberately NOT consumed. Placement is gated by
// the per-character cap in atlas-kites instead, so the arm must issue no saga
// and no inventory mutation -- no command may land on any topic other than the
// kite command topic.
func TestKiteUseDoesNotConsumeTheItem(t *testing.T) {
	const characterId = uint32(42)
	const itemId = uint32(5080000)

	restoreEnv := newKiteCharacterServer(t, characterId, "Player", 320, -140)
	defer restoreEnv()

	restoreSlot := installCashItemInSlotSeam(t, cashRockSlot, itemId)
	defer restoreSlot()

	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	l := logrus.New()
	r := kiteUseRequest(l, cashRockSlot, itemId, "congrats!")

	handlerFunc := CharacterCashItemUseHandleFunc(l, ctx, nil)
	handlerFunc(s, r, map[string]interface{}{})

	for topic := range *captured {
		if topic != string(kiteMsg.EnvCommandTopic) {
			t.Errorf("kite use emitted on unexpected topic %q — no saga or inventory command may be issued", topic)
		}
	}
}
