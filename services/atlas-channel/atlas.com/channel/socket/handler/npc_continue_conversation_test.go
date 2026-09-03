package handler

import (
	"atlas-channel/npc"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// gms83MessageType mirrors the "messageType" table the NPCContinueConversation
// handler is configured with (template_gms_83_1.json). The handler reverse-maps
// the client's lastMessageType byte through this table, so the test drives the
// real config rather than hardcoded bytes.
var gms83MessageType = map[string]interface{}{
	"SAY":                    float64(0),
	"ASK_YES_NO":             float64(1),
	"ASK_TEXT":               float64(2),
	"ASK_NUMBER":             float64(3),
	"ASK_MENU":               float64(4),
	"ASK_QUIZ":               float64(5),
	"ASK_SPEED_QUIZ":         float64(6),
	"ASK_AVATAR":             float64(7),
	"ASK_MEMBER_SHOP_AVATAR": float64(8),
	"ASK_PET":                float64(9),
	"ASK_PET_ALL":            float64(10),
	"ASK_YES_NO_QUEST":       float64(12),
	"ASK_BOX_TEXT":           float64(13),
	"ASK_SLIDE_MENU":         float64(14),
}

// TestContinueConversationBodyKind pins the discriminator that decides which
// trailing body the serverbound continue-conversation packet carries. The
// version-specific byte numbering comes from tenant config; only the
// name→body-kind grouping is asserted here.
//
//	SAY / ASK_YES_NO / ASK_YES_NO_QUEST              → no trailing body
//	ASK_TEXT / ASK_BOX_TEXT                          → text reply
//	ASK_NUMBER / ASK_MENU / ASK_AVATAR / ASK_SLIDE_MENU → selection reply
func TestContinueConversationBodyKind(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	opts := map[string]interface{}{"messageType": gms83MessageType}

	cases := []struct {
		name    string
		msgType byte
		want    bodyKind
	}{
		{"SAY", 0, bodyNone},
		{"ASK_YES_NO", 1, bodyNone},
		{"ASK_YES_NO_QUEST", 12, bodyNone},
		{"ASK_TEXT", 2, bodyText},
		{"ASK_BOX_TEXT", 13, bodyText},
		{"ASK_NUMBER", 3, bodySelection},
		{"ASK_MENU", 4, bodySelection},
		{"ASK_AVATAR", 7, bodySelection},
		{"ASK_SLIDE_MENU", 14, bodySelection},
	}
	for _, c := range cases {
		if got := bodyKindFor(l, opts, c.msgType); got != c.want {
			t.Errorf("%s (byte %d): got %v, want %v", c.name, c.msgType, got, c.want)
		}
	}
}

// TestContinueConversationBodyKindUnconfigured guards the failure mode that
// caused the original regression: when the byte is not present in the
// messageType table (e.g. missing handler config), the handler must fall back
// to bodyNone rather than mis-parse the trailing bytes.
func TestContinueConversationBodyKindUnconfigured(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	// Empty options: nothing configured.
	if got := bodyKindFor(l, map[string]interface{}{}, 4); got != bodyNone {
		t.Errorf("missing messageType config: got %v, want bodyNone", got)
	}

	// Byte not assigned to any known name in the table.
	opts := map[string]interface{}{"messageType": gms83MessageType}
	if got := bodyKindFor(l, opts, 99); got != bodyNone {
		t.Errorf("unknown byte 99: got %v, want bodyNone", got)
	}
}

// recordingNpcProcessor is the recording npc.Processor test seam
// (npcProcessorFunc in npc_continue_conversation.go), used to capture what
// NPCContinueConversationHandleFunc forwards to ContinueConversation without
// a live Kafka broker.
type recordingNpcProcessor struct {
	continueCalls []recordedContinueCall
	disposeCalls  int
}

type recordedContinueCall struct {
	characterId     uint32
	action          byte
	lastMessageType byte
	selection       int32
	text            string
}

func (p *recordingNpcProcessor) StartConversation(_ field.Model, _ uint32, _ uint32, _ uint32) error {
	return nil
}

func (p *recordingNpcProcessor) ContinueConversation(characterId uint32, action byte, lastMessageType byte, selection int32, text string) error {
	p.continueCalls = append(p.continueCalls, recordedContinueCall{
		characterId:     characterId,
		action:          action,
		lastMessageType: lastMessageType,
		selection:       selection,
		text:            text,
	})
	return nil
}

func (p *recordingNpcProcessor) DisposeConversation(_ uint32) error {
	p.disposeCalls++
	return nil
}

var _ npc.Processor = (*recordingNpcProcessor)(nil)

// newContinueConversationTestSession builds a session.Model for characterId
// (idiom: newAutoAggroTestSession in auto_aggro_test.go).
func newContinueConversationTestSession(t *testing.T, characterId uint32) session.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)
	t.Cleanup(func() { session.ClearRegistryForTenant(ten.Id()) })

	ctx := tenant.WithContext(context.Background(), ten)
	sp := session.NewProcessor(logrus.New(), ctx)
	return sp.SetCharacterId(sessionId, characterId)
}

// TestContinueConversationCarriesText pins the fix for the regression named
// in this task: the handler decoded the player's typed ASK_TEXT/ASK_BOX_TEXT
// reply and then discarded it, so the engine never received it. It now
// forwards the decoded text on the ContinueConversation call, leaving the
// selection/number path and the cancel-disposes path unaffected.
func TestContinueConversationCarriesText(t *testing.T) {
	opts := map[string]interface{}{"messageType": gms83MessageType}

	cases := []struct {
		name            string
		action          byte
		lastMessageType byte
		body            func(w *response.Writer)
		wantDispose     bool
		wantText        string
		wantSelection   int32
	}{
		{
			name:            "text reply carried",
			action:          1,
			lastMessageType: 2, // ASK_TEXT
			body:            func(w *response.Writer) { w.WriteAsciiString("Open Sesame") },
			wantText:        "Open Sesame",
			wantSelection:   -1,
		},
		{
			name:            "empty reply carried",
			action:          1,
			lastMessageType: 2, // ASK_TEXT
			body:            func(w *response.Writer) { w.WriteAsciiString("") },
			wantText:        "",
			wantSelection:   -1,
		},
		{
			name:            "box text reply carried",
			action:          1,
			lastMessageType: 13, // ASK_BOX_TEXT
			body:            func(w *response.Writer) { w.WriteAsciiString("multi line") },
			wantText:        "multi line",
			wantSelection:   -1,
		},
		{
			name:            "cancel disposes",
			action:          0,
			lastMessageType: 2, // ASK_TEXT
			body:            func(w *response.Writer) {},
			wantDispose:     true,
		},
		{
			name:            "selection path unaffected",
			action:          1,
			lastMessageType: 3, // ASK_NUMBER
			body:            func(w *response.Writer) { w.WriteInt32(7) },
			wantText:        "",
			wantSelection:   7,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, _ := testlog.NewNullLogger()
			s := newContinueConversationTestSession(t, 1001)

			w := response.NewWriter(l)
			w.WriteByte(c.lastMessageType)
			w.WriteByte(c.action)
			c.body(w)
			req := request.Request(w.Bytes())
			reader := request.NewRequestReader(&req, 0)

			rec := &recordingNpcProcessor{}
			prior := npcProcessorFunc
			npcProcessorFunc = func(_ logrus.FieldLogger, _ context.Context) npc.Processor { return rec }
			t.Cleanup(func() { npcProcessorFunc = prior })

			var wp writer.Producer
			NPCContinueConversationHandleFunc(l, context.Background(), wp)(s, &reader, opts)

			if c.wantDispose {
				if rec.disposeCalls != 1 {
					t.Fatalf("disposeCalls = %d, want 1", rec.disposeCalls)
				}
				if len(rec.continueCalls) != 0 {
					t.Fatalf("continueCalls = %d, want 0", len(rec.continueCalls))
				}
				return
			}

			if len(rec.continueCalls) != 1 {
				t.Fatalf("continueCalls = %d, want 1", len(rec.continueCalls))
			}
			got := rec.continueCalls[0]
			if got.text != c.wantText {
				t.Errorf("text = %q, want %q", got.text, c.wantText)
			}
			if got.selection != c.wantSelection {
				t.Errorf("selection = %d, want %d", got.selection, c.wantSelection)
			}
			if got.characterId != 1001 {
				t.Errorf("characterId = %d, want 1001", got.characterId)
			}
			if got.action != c.action {
				t.Errorf("action = %d, want %d", got.action, c.action)
			}
			if got.lastMessageType != c.lastMessageType {
				t.Errorf("lastMessageType = %d, want %d", got.lastMessageType, c.lastMessageType)
			}
		})
	}
}
