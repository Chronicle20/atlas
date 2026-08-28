package conversation

import (
	npcSender "atlas-npc-conversations/npc"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
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

// recordingKafkaWriter is a producer.Writer test double that records every
// message handed to it instead of dialing a broker. Installed once, process-
// wide, via producer.GetManager(producer.ConfigWriterFactory(...)) so that
// Continue's downstream ProcessState loop can walk into a real dialogue
// state (which calls npcSender.NewProcessor directly, bypassing the
// npcSenderProcessorFactory seam) without attempting a real Kafka write.
type recordingKafkaWriter struct {
	mu       sync.Mutex
	topic    string
	messages []kafka.Message
}

func (w *recordingKafkaWriter) Topic() string { return w.topic }

func (w *recordingKafkaWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messages = append(w.messages, msgs...)
	return nil
}

func (w *recordingKafkaWriter) Close() error { return nil }

func (w *recordingKafkaWriter) last() kafka.Message {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.messages[len(w.messages)-1]
}

var (
	kafkaWriterOnce sync.Once
	recordedWriter  = &recordingKafkaWriter{}
)

// installRecordingKafkaWriter installs recordedWriter as the process-wide
// producer.Manager's writer factory exactly once. Subsequent calls are no-ops
// (sync.Once), matching the manager's own singleton semantics.
func installRecordingKafkaWriter() *recordingKafkaWriter {
	kafkaWriterOnce.Do(func() {
		producer.GetManager(producer.ConfigWriterFactory(func(topic string) producer.Writer {
			recordedWriter.topic = topic
			return recordedWriter
		}))
	})
	return recordedWriter
}

// initAskTextRegistry wires a fresh miniredis-backed registry for the test.
func initAskTextRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
}

// buildAskTextMatchState builds the shared "askPassword" fixture state: an
// askText state parked awaiting free text, with the ordered match table from
// the task brief. matches, when non-nil, overrides the default three-entry
// table (used by the "empty matches" case).
func buildAskTextMatchState(t *testing.T, matches []AskTextMatchModel) StateModel {
	t.Helper()

	if matches == nil {
		literal, err := NewAskTextMatchBuilder().SetValue("Open Sesame").SetNextState("s-literal").Build()
		if err != nil {
			t.Fatalf("build literal match: %v", err)
		}
		fromContext, err := NewAskTextMatchBuilder().SetValueFromContext("{context.pw}").SetNextState("s-context").Build()
		if err != nil {
			t.Fatalf("build context match: %v", err)
		}
		duplicate, err := NewAskTextMatchBuilder().SetValue("Open Sesame").SetNextState("s-duplicate").Build()
		if err != nil {
			t.Fatalf("build duplicate match: %v", err)
		}
		matches = []AskTextMatchModel{*literal, *fromContext, *duplicate}
	}

	builder := NewAskTextBuilder().
		SetText("Enter the password.").
		SetMinLength(3).
		SetMaxLength(20).
		SetContextKey("answer").
		SetNextState("wrong-password")
	for _, m := range matches {
		builder = builder.AddMatch(m)
	}
	askText, err := builder.Build()
	if err != nil {
		t.Fatalf("build askText: %v", err)
	}

	state, err := NewStateBuilder().SetId("askPassword").SetAskText(askText).Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}
	return state
}

// buildAskTextParkState builds a trivial askText state that acts as a
// terminal target for Continue's downstream ProcessState loop: it re-parks
// on itself (via processAskTextState's "waiting for input" return) without
// re-entering the branch-evaluation logic under test.
func buildAskTextParkState(t *testing.T, id string) StateModel {
	t.Helper()
	askText, err := NewAskTextBuilder().
		SetText("(parked)").
		SetMinLength(0).
		SetMaxLength(999).
		SetContextKey("_park").
		SetNextState(id).
		Build()
	if err != nil {
		t.Fatalf("build park askText [%s]: %v", id, err)
	}
	state, err := NewStateBuilder().SetId(id).SetAskText(askText).Build()
	if err != nil {
		t.Fatalf("build park state [%s]: %v", id, err)
	}
	return state
}

func TestContinueAskText(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		preContext   map[string]string
		matches      []AskTextMatchModel // nil uses the shared three-entry table
		emptyMatches bool
		wantErr      bool
		wantState    string
		wantAnswer   string
	}{
		{
			name:       "first match wins",
			text:       "Open Sesame",
			preContext: map[string]string{},
			wantState:  "s-literal",
			wantAnswer: "Open Sesame",
		},
		{
			name:       "duplicate never reached",
			text:       "Open Sesame",
			preContext: map[string]string{},
			wantState:  "s-literal",
			wantAnswer: "Open Sesame",
		},
		{
			name:       "context match",
			text:       "hunter2xy",
			preContext: map[string]string{"pw": "hunter2xy"},
			wantState:  "s-context",
			wantAnswer: "hunter2xy",
		},
		{
			name:       "context match unresolved",
			text:       "hunter2xy",
			preContext: map[string]string{},
			wantState:  "wrong-password",
			wantAnswer: "hunter2xy",
		},
		{
			name:       "no match falls back",
			text:       "wrong answer",
			preContext: map[string]string{},
			wantState:  "wrong-password",
			wantAnswer: "wrong answer",
		},
		{
			name:         "empty matches",
			text:         "anything ok",
			preContext:   map[string]string{},
			emptyMatches: true,
			wantState:    "wrong-password",
			wantAnswer:   "anything ok",
		},
		{
			name:       "trimmed before match",
			text:       "  Open Sesame  ",
			preContext: map[string]string{},
			wantState:  "s-literal",
			wantAnswer: "Open Sesame",
		},
		{
			name:       "trimmed before length check",
			text:       "  ab  ",
			preContext: map[string]string{},
			wantErr:    true,
		},
		{
			name:       "case sensitive",
			text:       "open sesame",
			preContext: map[string]string{},
			wantState:  "wrong-password",
			wantAnswer: "open sesame",
		},
		{
			name:       "below minLength",
			text:       "ab",
			preContext: map[string]string{},
			wantErr:    true,
		},
		{
			name:       "at minLength",
			text:       "abc",
			preContext: map[string]string{},
			wantState:  "wrong-password",
			wantAnswer: "abc",
		},
		{
			name:       "at maxLength",
			text:       "abcdefghijklmnopqrst",
			preContext: map[string]string{},
			wantState:  "wrong-password",
			wantAnswer: "abcdefghijklmnopqrst",
		},
		{
			name:       "above maxLength",
			text:       "abcdefghijklmnopqrstu",
			preContext: map[string]string{},
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &recordingNpcSenderProcessor{}
			SetNpcSenderProcessorFactory(func(l logrus.FieldLogger, ctx context.Context) npcSender.Processor {
				return mock
			})
			defer SetNpcSenderProcessorFactory(nil)

			initAskTextRegistry(t)

			matches := tc.matches
			if tc.emptyMatches {
				matches = []AskTextMatchModel{}
			}
			askState := buildAskTextMatchState(t, matches)

			container := testStateContainer{
				start: "askPassword",
				states: []StateModel{
					askState,
					buildAskTextParkState(t, "s-literal"),
					buildAskTextParkState(t, "s-context"),
					buildAskTextParkState(t, "s-duplicate"),
					buildAskTextParkState(t, "wrong-password"),
				},
			}

			ctx := buildAskTextConversationContext(t, "askPassword", container, tc.preContext)

			l, hook := test.NewNullLogger()
			l.SetLevel(logrus.DebugLevel)
			var tm tenant.Model
			tctx := tenant.WithContext(context.Background(), tm)

			GetRegistry().SetContext(tctx, ctx.CharacterId(), ctx)

			p := &ProcessorImpl{l: l, ctx: tctx, t: tm}

			err := p.Continue(ctx.NpcId(), ctx.CharacterId(), 0, 0, 0, tc.text)

			got, getErr := GetRegistry().GetPreviousContext(tctx, ctx.CharacterId())
			if getErr != nil {
				t.Fatalf("GetPreviousContext: %v", getErr)
			}

			if tc.wantErr {
				if err == nil {
					t.Fatalf("Continue returned nil error, want error")
				}
				if got.CurrentState() != "askPassword" {
					t.Errorf("CurrentState = %q, want %q (parked on error)", got.CurrentState(), "askPassword")
				}
				foundErrorLog := false
				for _, entry := range hook.AllEntries() {
					if entry.Level.String() == "error" {
						foundErrorLog = true
					}
				}
				if !foundErrorLog {
					t.Errorf("expected an error-level log entry for the length rejection")
				}
				return
			}

			if err != nil {
				t.Fatalf("Continue returned error: %v", err)
			}
			if got.CurrentState() != tc.wantState {
				t.Errorf("CurrentState = %q, want %q", got.CurrentState(), tc.wantState)
			}
			if v := got.Context()["answer"]; v != tc.wantAnswer {
				t.Errorf("context[answer] = %q, want %q", v, tc.wantAnswer)
			}
		})
	}
}

// TestContinueAskTextDownstreamContextRead drives Continue into a real
// dialogue state that reads the stored answer back out of the context,
// proving the PRD acceptance criterion "the reply is readable from a later
// state". Because the dialogue path calls npcSender.NewProcessor directly
// (not through the overridable npcSenderProcessorFactory seam), a recording
// Kafka writer is installed on the shared producer manager so the resulting
// event can be inspected without a live broker.
func TestContinueAskTextDownstreamContextRead(t *testing.T) {
	writer := installRecordingKafkaWriter()

	initAskTextRegistry(t)

	askText, err := NewAskTextBuilder().
		SetText("Enter the password.").
		SetMinLength(3).
		SetMaxLength(20).
		SetContextKey("answer").
		SetNextState("echo").
		Build()
	if err != nil {
		t.Fatalf("build askText: %v", err)
	}
	askState, err := NewStateBuilder().SetId("askPassword").SetAskText(askText).Build()
	if err != nil {
		t.Fatalf("build askText state: %v", err)
	}

	choiceOk, err := NewChoiceBuilder().SetText("OK").SetNextState("").Build()
	if err != nil {
		t.Fatalf("build choice: %v", err)
	}
	dialogue, err := NewDialogueBuilder().
		SetDialogueType(SendOk).
		SetText("You said {context.answer}.").
		AddChoice(choiceOk).
		AddChoice(choiceOk).
		Build()
	if err != nil {
		t.Fatalf("build dialogue: %v", err)
	}
	echoState, err := NewStateBuilder().SetId("echo").SetDialogue(dialogue).Build()
	if err != nil {
		t.Fatalf("build echo state: %v", err)
	}

	container := testStateContainer{start: "askPassword", states: []StateModel{askState, echoState}}
	ctx := buildAskTextConversationContext(t, "askPassword", container, map[string]string{})

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	GetRegistry().SetContext(tctx, ctx.CharacterId(), ctx)

	p := &ProcessorImpl{l: l, ctx: tctx, t: tm}

	if err := p.Continue(ctx.NpcId(), ctx.CharacterId(), 0, 0, 0, "Open Sesame"); err != nil {
		t.Fatalf("Continue returned error: %v", err)
	}

	msg := writer.last()
	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(msg.Value, &body); err != nil {
		t.Fatalf("unmarshal recorded message: %v", err)
	}
	if strings.TrimSpace(body.Message) == "" {
		t.Fatalf("recorded message was empty")
	}
	if body.Message != "You said Open Sesame." {
		t.Errorf("rendered dialogue text = %q, want %q", body.Message, "You said Open Sesame.")
	}
}
