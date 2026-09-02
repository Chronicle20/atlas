package playernpc

import (
	msg "atlas-messages/kafka/message/playernpc"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func testRequester() *msg.Requester {
	return &msg.Requester{CharacterId: 7, WorldId: 1, ChannelId: 2, MapId: 102000004}
}

func TestHandleCommandSucceeded(t *testing.T) {
	logger, _ := test.NewNullLogger()

	t.Run("nil requester declined", func(t *testing.T) {
		var pinkCalls int
		pink := func(f field.Model, text string, recipients []uint32) error { pinkCalls++; return nil }

		e := msg.StatusEvent[msg.StatusCommandOutcomeBody]{
			Type: msg.EventTypeCommandSucceeded,
			Body: msg.StatusCommandOutcomeBody{
				TransactionId: uuid.New(),
				CharacterId:   42,
				CommandType:   msg.CommandTypeDeploy,
			},
		}
		handleCommandSucceededWithDeps(logger, e, pink)
		if pinkCalls != 0 {
			t.Fatalf("pink calls = %d, want 0", pinkCalls)
		}
	})

	t.Run("wrong type declined", func(t *testing.T) {
		var pinkCalls int
		pink := func(f field.Model, text string, recipients []uint32) error { pinkCalls++; return nil }

		e := msg.StatusEvent[msg.StatusCommandOutcomeBody]{
			Type: msg.EventTypeCommandFailed,
			Body: msg.StatusCommandOutcomeBody{
				CharacterId: 42,
				CommandType: msg.CommandTypeDeploy,
				Requester:   testRequester(),
			},
		}
		handleCommandSucceededWithDeps(logger, e, pink)
		if pinkCalls != 0 {
			t.Fatalf("pink calls = %d, want 0", pinkCalls)
		}
	})

	t.Run("succeeded produces accepted text addressed to requester", func(t *testing.T) {
		var texts []string
		var recipients [][]uint32
		var fields []field.Model
		pink := func(f field.Model, text string, r []uint32) error {
			texts = append(texts, text)
			recipients = append(recipients, r)
			fields = append(fields, f)
			return nil
		}

		requester := testRequester()
		e := msg.StatusEvent[msg.StatusCommandOutcomeBody]{
			Type: msg.EventTypeCommandSucceeded,
			Body: msg.StatusCommandOutcomeBody{
				CharacterId: 42,
				CommandType: msg.CommandTypeDeploy,
				Requester:   requester,
			},
		}
		handleCommandSucceededWithDeps(logger, e, pink)
		if len(texts) != 1 {
			t.Fatalf("pink calls = %d, want 1", len(texts))
		}
		if len(recipients) != 1 || len(recipients[0]) != 1 || recipients[0][0] != requester.CharacterId {
			t.Errorf("recipients = %v, want [%d]", recipients, requester.CharacterId)
		}
		wantField := field.NewBuilder(1, 2, 102000004).Build()
		if fields[0].WorldId() != wantField.WorldId() || fields[0].ChannelId() != wantField.ChannelId() || fields[0].MapId() != wantField.MapId() {
			t.Errorf("field = %+v, want %+v", fields[0], wantField)
		}
		if texts[0] == "" {
			t.Fatal("pink text is empty")
		}
	})
}

// TestFailureCodesProduceDistinctSentences is the plan.md Task 23c test
// table: each of the four design §8.3 codes, plus unresolvable, internal
// and an unrecognised default, must produce its own distinct sentence --
// not the bare code string.
func TestFailureCodesProduceDistinctSentences(t *testing.T) {
	logger, _ := test.NewNullLogger()
	requester := testRequester()

	codes := []string{"pool_exhausted", "map_full", "duplicate", "ineligible", "unresolvable", "internal", "some_unrecognised_code"}
	seen := make(map[string]bool)

	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			var texts []string
			pink := func(f field.Model, text string, r []uint32) error { texts = append(texts, text); return nil }

			e := msg.StatusEvent[msg.StatusCommandOutcomeBody]{
				Type: msg.EventTypeCommandFailed,
				Body: msg.StatusCommandOutcomeBody{
					CharacterId: 42,
					CommandType: msg.CommandTypeDeploy,
					Code:        code,
					Requester:   requester,
				},
			}
			handleCommandFailedWithDeps(logger, e, pink)
			if len(texts) != 1 {
				t.Fatalf("pink calls = %d, want 1", len(texts))
			}
			if texts[0] == code {
				t.Errorf("pink text = %q, want a human-readable sentence, not the bare code", texts[0])
			}
			if seen[texts[0]] {
				t.Errorf("pink text %q reused across codes, want a distinct sentence per code", texts[0])
			}
			seen[texts[0]] = true
		})
	}
}

func TestHandleCommandFailed_NilRequesterDeclined(t *testing.T) {
	logger, _ := test.NewNullLogger()
	var pinkCalls int
	pink := func(f field.Model, text string, r []uint32) error { pinkCalls++; return nil }

	e := msg.StatusEvent[msg.StatusCommandOutcomeBody]{
		Type: msg.EventTypeCommandFailed,
		Body: msg.StatusCommandOutcomeBody{
			CharacterId: 42,
			CommandType: msg.CommandTypeDeploy,
			Code:        "pool_exhausted",
		},
	}
	handleCommandFailedWithDeps(logger, e, pink)
	if pinkCalls != 0 {
		t.Fatalf("pink calls = %d, want 0", pinkCalls)
	}
}
