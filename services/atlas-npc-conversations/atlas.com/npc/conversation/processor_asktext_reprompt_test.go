package conversation

import (
	npcSender "atlas-npc-conversations/npc"
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestContinueAskText_RepromptOnLengthViolation drives Continue with text
// that violates MinLength/MaxLength and asserts the engine re-prompts on the
// same state instead of silently discarding the input (the bug from task
// 21's review). It reuses buildAskTextMatchState/buildAskTextParkState/
// buildAskTextConversationContext/initAskTextRegistry/
// recordingNpcSenderProcessor from processor_asktext_test.go.
func TestContinueAskText_RepromptOnLengthViolation(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "below minLength", text: "ab"},                    // MinLength is 3
		{name: "above maxLength", text: "abcdefghijklmnopqrstu"}, // MaxLength is 20
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &recordingNpcSenderProcessor{}
			SetNpcSenderProcessorFactory(func(l logrus.FieldLogger, ctx context.Context) npcSender.Processor {
				return mock
			})
			defer SetNpcSenderProcessorFactory(nil)

			initAskTextRegistry(t)

			askState := buildAskTextMatchState(t, nil)
			askText := askState.AskText()

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

			ctx := buildAskTextConversationContext(t, "askPassword", container, map[string]string{})

			l, _ := test.NewNullLogger()
			l.SetLevel(logrus.DebugLevel)
			var tm tenant.Model
			tctx := tenant.WithContext(context.Background(), tm)

			GetRegistry().SetContext(tctx, ctx.CharacterId(), ctx)

			p := &ProcessorImpl{l: l, ctx: tctx, t: tm}

			err := p.Continue(ctx.NpcId(), ctx.CharacterId(), 0, 0, 0, tc.text)
			if err != nil {
				t.Fatalf("Continue returned error, want nil (re-prompt, not error): %v", err)
			}

			// SendText must be called again for the SAME state, with the
			// same prompt/default/min/max the state declares.
			if !mock.sendTextCalled {
				t.Fatalf("SendText was not called; expected a re-prompt")
			}
			if mock.message != askText.Text() {
				t.Errorf("re-prompt message = %q, want %q", mock.message, askText.Text())
			}
			if mock.defaultValue != askText.DefaultText() {
				t.Errorf("re-prompt default = %q, want %q", mock.defaultValue, askText.DefaultText())
			}
			if mock.minLength != askText.MinLength() {
				t.Errorf("re-prompt minLength = %d, want %d", mock.minLength, askText.MinLength())
			}
			if mock.maxLength != askText.MaxLength() {
				t.Errorf("re-prompt maxLength = %d, want %d", mock.maxLength, askText.MaxLength())
			}

			// The rejected input must NOT be written to the conversation
			// context.
			got, getErr := GetRegistry().GetPreviousContext(tctx, ctx.CharacterId())
			if getErr != nil {
				t.Fatalf("GetPreviousContext: %v", getErr)
			}
			if v, exists := got.Context()["answer"]; exists {
				t.Errorf("context[answer] = %q, want absent (rejected input must not be stored)", v)
			}

			// The conversation must stay on the same state.
			if got.CurrentState() != "askPassword" {
				t.Errorf("CurrentState = %q, want %q (re-prompt stays on same state)", got.CurrentState(), "askPassword")
			}
		})
	}
}

// TestContinueAskText_RepromptPreservesPriorContextValue proves the rejected
// input does not clobber a previously-stored value under the same context
// key: after re-prompting, the key holds its prior value, never the invalid
// string.
func TestContinueAskText_RepromptPreservesPriorContextValue(t *testing.T) {
	mock := &recordingNpcSenderProcessor{}
	SetNpcSenderProcessorFactory(func(l logrus.FieldLogger, ctx context.Context) npcSender.Processor {
		return mock
	})
	defer SetNpcSenderProcessorFactory(nil)

	initAskTextRegistry(t)

	askState := buildAskTextMatchState(t, nil)
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

	ctx := buildAskTextConversationContext(t, "askPassword", container, map[string]string{"answer": "priorvalue"})

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	GetRegistry().SetContext(tctx, ctx.CharacterId(), ctx)

	p := &ProcessorImpl{l: l, ctx: tctx, t: tm}

	if err := p.Continue(ctx.NpcId(), ctx.CharacterId(), 0, 0, 0, "ab"); err != nil {
		t.Fatalf("Continue returned error, want nil: %v", err)
	}

	got, getErr := GetRegistry().GetPreviousContext(tctx, ctx.CharacterId())
	if getErr != nil {
		t.Fatalf("GetPreviousContext: %v", getErr)
	}
	if v := got.Context()["answer"]; v != "priorvalue" {
		t.Errorf("context[answer] = %q, want %q (prior value must survive the rejected input)", v, "priorvalue")
	}
	if got.CurrentState() != "askPassword" {
		t.Errorf("CurrentState = %q, want %q", got.CurrentState(), "askPassword")
	}
}

// TestContinueAskText_ValidInputAfterReprompt proves a re-prompt leaves the
// conversation in a usable state: a subsequent valid input on the same state
// advances normally, storing the trimmed value and honoring first-match-wins
// over matches (regression coverage for the happy path this fix must not
// disturb).
func TestContinueAskText_ValidInputAfterReprompt(t *testing.T) {
	mock := &recordingNpcSenderProcessor{}
	SetNpcSenderProcessorFactory(func(l logrus.FieldLogger, ctx context.Context) npcSender.Processor {
		return mock
	})
	defer SetNpcSenderProcessorFactory(nil)

	initAskTextRegistry(t)

	askState := buildAskTextMatchState(t, nil)
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

	ctx := buildAskTextConversationContext(t, "askPassword", container, map[string]string{})

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	GetRegistry().SetContext(tctx, ctx.CharacterId(), ctx)

	p := &ProcessorImpl{l: l, ctx: tctx, t: tm}

	// First, a rejected input that must re-prompt without wedging the
	// conversation.
	if err := p.Continue(ctx.NpcId(), ctx.CharacterId(), 0, 0, 0, "ab"); err != nil {
		t.Fatalf("Continue (invalid) returned error, want nil: %v", err)
	}

	// Then a valid input on the same state must advance normally, honoring
	// first-match-wins over the ordered match table.
	if err := p.Continue(ctx.NpcId(), ctx.CharacterId(), 0, 0, 0, "Open Sesame"); err != nil {
		t.Fatalf("Continue (valid) returned error: %v", err)
	}

	got, getErr := GetRegistry().GetPreviousContext(tctx, ctx.CharacterId())
	if getErr != nil {
		t.Fatalf("GetPreviousContext: %v", getErr)
	}
	if got.CurrentState() != "s-literal" {
		t.Errorf("CurrentState = %q, want %q", got.CurrentState(), "s-literal")
	}
	if v := got.Context()["answer"]; v != "Open Sesame" {
		t.Errorf("context[answer] = %q, want %q", v, "Open Sesame")
	}
}
