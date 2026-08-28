package handler

import (
	expression2 "atlas-channel/kafka/message/expression"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// installExpressionItemOwnedSeam swaps expressionItemOwnedFunc for the test and
// returns the call log (the itemId requested on each call) plus a restore func
// (shape copied from installCashItemInSlotSeam in character_cash_item_use_test.go).
func installExpressionItemOwnedSeam(t *testing.T, owns bool, err error) (*[]item.Id, func()) {
	t.Helper()
	orig := expressionItemOwnedFunc
	calls := make([]item.Id, 0)
	expressionItemOwnedFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, itemId item.Id) (bool, error) {
		calls = append(calls, itemId)
		return owns, err
	}
	return &calls, func() {
		expressionItemOwnedFunc = orig
	}
}

// expressionRequestBytes builds the wire payload for a GMS v83 expression
// request: a bare Encode4(emote) body, matching ExpressionRequest.Decode for
// that version (character_expression.go's Decode gate, IDA v83
// CWvsContext::SendEmotionChange@0xa24470 - shape copied from kiteUseRequest
// in character_cash_item_use_kite_test.go).
func expressionRequestBytes(l logrus.FieldLogger, emote uint32) *request.Reader {
	w := response.NewWriter(l)
	w.WriteInt(emote)
	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

func TestCharacterExpressionHandleFunc_Gate(t *testing.T) {
	tests := []struct {
		name           string
		emote          uint32
		owns           bool
		seamErr        error
		expectCalls    int
		expectItemId   item.Id
		expectCommands int
	}{
		{
			name:           "base emote skips the lookup",
			emote:          5,
			expectCalls:    0,
			expectCommands: 1,
		},
		{
			name:           "base emote upper bound skips the lookup",
			emote:          7,
			expectCalls:    0,
			expectCommands: 1,
		},
		{
			name:           "owned extra expression is forwarded",
			emote:          8,
			owns:           true,
			expectCalls:    1,
			expectItemId:   item.Id(5160000),
			expectCommands: 1,
		},
		{
			name:           "unowned extra expression is dropped",
			emote:          8,
			owns:           false,
			expectCalls:    1,
			expectItemId:   item.Id(5160000),
			expectCommands: 0,
		},
		{
			name:           "lookup error fails closed",
			emote:          8,
			owns:           false,
			seamErr:        errors.New("boom"),
			expectCalls:    1,
			expectItemId:   item.Id(5160000),
			expectCommands: 0,
		},
		{
			name:           "gated upper bound is dropped when unowned",
			emote:          23,
			owns:           false,
			expectCalls:    1,
			expectItemId:   item.Id(5160015),
			expectCommands: 0,
		},
		{
			name:           "out of range never reaches the lookup",
			emote:          24,
			expectCalls:    0,
			expectCommands: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			captured, restoreProducer := installCapturingProducer()
			defer restoreProducer()

			calls, restoreSeam := installExpressionItemOwnedSeam(t, tc.owns, tc.seamErr)
			defer restoreSeam()

			s, ctx, cleanup := newCashItemUseTestSession(t, 555)
			defer cleanup()

			l := logrus.New()
			CharacterExpressionHandleFunc(l, ctx, nil)(s, expressionRequestBytes(l, tc.emote), map[string]interface{}{})

			if len(*calls) != tc.expectCalls {
				t.Fatalf("seam calls: got %d, want %d", len(*calls), tc.expectCalls)
			}
			if tc.expectCalls > 0 && (*calls)[0] != tc.expectItemId {
				t.Errorf("seam itemId: got %d, want %d", (*calls)[0], tc.expectItemId)
			}

			msgs := (*captured)[string(expression2.EnvExpressionCommand)]
			if len(msgs) != tc.expectCommands {
				t.Fatalf("commands emitted: got %d, want %d", len(msgs), tc.expectCommands)
			}

			if tc.expectCommands > 0 {
				var cmd expression2.Command
				if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
					t.Fatalf("unmarshal command: %v", err)
				}
				if cmd.Expression != tc.emote {
					t.Errorf("command.Expression: got %d, want %d", cmd.Expression, tc.emote)
				}
				if cmd.CharacterId != 555 {
					t.Errorf("command.CharacterId: got %d, want %d", cmd.CharacterId, 555)
				}
			}
		})
	}
}

func TestCharacterExpressionHandleFunc_ForwardsDurationAndByItemOption(t *testing.T) {
	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	_, restoreSeam := installExpressionItemOwnedSeam(t, true, nil)
	defer restoreSeam()

	s, ctx, cleanup := newCashItemUseTestSessionForVersion(t, 555, "GMS", 95)
	defer cleanup()

	l := logrus.New()
	w := response.NewWriter(l)
	w.WriteInt(8)
	w.WriteInt32(-1)
	w.WriteBool(false)
	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)

	CharacterExpressionHandleFunc(l, ctx, nil)(s, &reader, map[string]interface{}{})

	msgs := (*captured)[string(expression2.EnvExpressionCommand)]
	if len(msgs) != 1 {
		t.Fatalf("commands emitted: got %d, want 1", len(msgs))
	}
	var cmd expression2.Command
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal command: %v", err)
	}
	if cmd.Duration != int32(-1) {
		t.Errorf("command.Duration: got %d, want -1", cmd.Duration)
	}
	if cmd.ByItemOption != false {
		t.Errorf("command.ByItemOption: got %v, want false", cmd.ByItemOption)
	}
}
