package conversation

import (
	npcSender "atlas-npc-conversations/npc"
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// recordingNpcSenderProcessor is a test double for npcSender.Processor that
// records the arguments passed to SendText. All other methods are no-ops;
// processAskTextState calls only SendText.
type recordingNpcSenderProcessor struct {
	sendTextCalled bool
	message        string
	defaultValue   string
	minLength      uint16
	maxLength      uint16
}

func (r *recordingNpcSenderProcessor) Dispose(ch channel.Model, characterId uint32) {}
func (r *recordingNpcSenderProcessor) SendSimple(ch channel.Model, characterId uint32, npcId uint32) npcSender.TalkFunc {
	return nil
}

func (r *recordingNpcSenderProcessor) SendNext(ch channel.Model, characterId uint32, npcId uint32) npcSender.TalkFunc {
	return nil
}

func (r *recordingNpcSenderProcessor) SendNextPrevious(ch channel.Model, characterId uint32, npcId uint32) npcSender.TalkFunc {
	return nil
}

func (r *recordingNpcSenderProcessor) SendPrevious(ch channel.Model, characterId uint32, npcId uint32) npcSender.TalkFunc {
	return nil
}

func (r *recordingNpcSenderProcessor) SendOk(ch channel.Model, characterId uint32, npcId uint32) npcSender.TalkFunc {
	return nil
}

func (r *recordingNpcSenderProcessor) SendYesNo(ch channel.Model, characterId uint32, npcId uint32) npcSender.TalkFunc {
	return nil
}

func (r *recordingNpcSenderProcessor) SendAcceptDecline(ch channel.Model, characterId uint32, npcId uint32) npcSender.TalkFunc {
	return nil
}

func (r *recordingNpcSenderProcessor) SendNumber(ch channel.Model, characterId uint32, npcId uint32, message string, def uint32, min uint32, max uint32) error {
	return nil
}

func (r *recordingNpcSenderProcessor) SendText(ch channel.Model, characterId uint32, npcId uint32, message string, def string, min uint16, max uint16) error {
	r.sendTextCalled = true
	r.message = message
	r.defaultValue = def
	r.minLength = min
	r.maxLength = max
	return nil
}

func (r *recordingNpcSenderProcessor) SendStyle(ch channel.Model, characterId uint32, npcId uint32, message string, styles []uint32) error {
	return nil
}

func (r *recordingNpcSenderProcessor) SendSlideMenu(ch channel.Model, characterId uint32, npcId uint32, message string, menuType uint32) error {
	return nil
}

func (r *recordingNpcSenderProcessor) SendNPCTalk(ch channel.Model, characterId uint32, npcId uint32, config *npcSender.TalkConfig) func(message string, configurations ...npcSender.TalkConfigurator) {
	return func(message string, configurations ...npcSender.TalkConfigurator) {}
}

var _ npcSender.Processor = (*recordingNpcSenderProcessor)(nil)

// newAskTextTestProcessor builds a bare ProcessorImpl suitable for
// processAskTextState, which touches only p.l and p.ctx (via the npc sender
// factory) and never the registry or database.
func newAskTextTestProcessor(t *testing.T) *ProcessorImpl {
	t.Helper()
	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	return &ProcessorImpl{
		l:   l,
		ctx: tctx,
		t:   tm,
	}
}

func buildAskTextConversationContext(t *testing.T, stateId string, container StateContainer, ctxValues map[string]string) ConversationContext {
	t.Helper()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	b := NewConversationContextBuilder().
		SetField(f).
		SetCharacterId(42).
		SetNpcId(9000019).
		SetCurrentState(stateId).
		SetConversation(container)
	if ctxValues != nil {
		b = b.SetContext(ctxValues)
	}
	return b.Build()
}

func TestProcessAskTextState(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		defaultText string
		minLength   uint16
		maxLength   uint16
		context     map[string]string
		wantMessage string
		wantDefault string
		wantMinLen  uint16
		wantMaxLen  uint16
	}{
		{
			name:        "plain prompt",
			text:        "Password!",
			defaultText: "",
			minLength:   1,
			maxLength:   32,
			context:     nil,
			wantMessage: "Password!",
			wantDefault: "",
			wantMinLen:  1,
			wantMaxLen:  32,
		},
		{
			name:        "context placeholder resolved",
			text:        "Enter #b{context.hint}#k!",
			defaultText: "",
			minLength:   1,
			maxLength:   32,
			context:     map[string]string{"hint": "the password"},
			wantMessage: "Enter #bthe password#k!",
			wantDefault: "",
			wantMinLen:  1,
			wantMaxLen:  32,
		},
		{
			name:        "default value carried",
			text:        "Password!",
			defaultText: "prefill",
			minLength:   1,
			maxLength:   32,
			context:     nil,
			wantMessage: "Password!",
			wantDefault: "prefill",
			wantMinLen:  1,
			wantMaxLen:  32,
		},
		{
			name:        "zero minLength",
			text:        "Password!",
			defaultText: "",
			minLength:   0,
			maxLength:   8,
			context:     nil,
			wantMessage: "Password!",
			wantDefault: "",
			wantMinLen:  0,
			wantMaxLen:  8,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &recordingNpcSenderProcessor{}
			SetNpcSenderProcessorFactory(func(l logrus.FieldLogger, ctx context.Context) npcSender.Processor {
				return mock
			})
			defer SetNpcSenderProcessorFactory(nil)

			askText, err := NewAskTextBuilder().
				SetText(tc.text).
				SetDefaultText(tc.defaultText).
				SetMinLength(tc.minLength).
				SetMaxLength(tc.maxLength).
				SetNextState("done").
				Build()
			if err != nil {
				t.Fatalf("build askText: %v", err)
			}

			state, err := NewStateBuilder().SetId("askPassword").SetAskText(askText).Build()
			if err != nil {
				t.Fatalf("build state: %v", err)
			}

			container := testStateContainer{start: "askPassword", states: []StateModel{state}}
			ctx := buildAskTextConversationContext(t, "askPassword", container, tc.context)

			p := newAskTextTestProcessor(t)
			nextStateId, err := p.processAskTextState(ctx, state)
			if err != nil {
				t.Fatalf("processAskTextState returned error: %v", err)
			}
			if nextStateId != state.Id() {
				t.Errorf("nextStateId = %q, want %q (parked awaiting input)", nextStateId, state.Id())
			}

			if !mock.sendTextCalled {
				t.Fatalf("SendText was not called")
			}
			if mock.message != tc.wantMessage {
				t.Errorf("Message = %q, want %q", mock.message, tc.wantMessage)
			}
			if mock.defaultValue != tc.wantDefault {
				t.Errorf("DefaultValue = %q, want %q", mock.defaultValue, tc.wantDefault)
			}
			if mock.minLength != tc.wantMinLen {
				t.Errorf("MinLength = %d, want %d", mock.minLength, tc.wantMinLen)
			}
			if mock.maxLength != tc.wantMaxLen {
				t.Errorf("MaxLength = %d, want %d", mock.maxLength, tc.wantMaxLen)
			}
		})
	}
}
