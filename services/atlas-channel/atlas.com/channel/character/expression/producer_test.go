package expression

import (
	"encoding/json"
	"testing"

	expression2 "atlas-channel/kafka/message/expression"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestSetCommandProviderCarriesDurationAndByItemOption verifies that
// SetCommandProvider carries duration and byItemOption through to the
// produced Kafka message envelope without clamping or altering them.
func TestSetCommandProviderCarriesDurationAndByItemOption(t *testing.T) {
	tests := []struct {
		name         string
		expression   uint32
		duration     int32
		byItemOption bool
	}{
		{"v95 extra expression", 8, -1, false},
		{"item option set", 12, 3000, true},
		{"pre-v95 zero values", 5, 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

			msgs, err := SetCommandProvider(1000, f, tc.expression, tc.duration, tc.byItemOption)()
			if err != nil {
				t.Fatalf("SetCommandProvider error: %v", err)
			}
			if len(msgs) != 1 {
				t.Fatalf("expected 1 message, got %d", len(msgs))
			}

			var cmd expression2.Command
			if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if cmd.CharacterId != 1000 {
				t.Errorf("characterId: got %d, want 1000", cmd.CharacterId)
			}
			if cmd.Expression != tc.expression {
				t.Errorf("expression: got %d, want %d", cmd.Expression, tc.expression)
			}
			if cmd.MapId != _map.Id(100000000) {
				t.Errorf("mapId: got %d, want 100000000", cmd.MapId)
			}
			if cmd.Duration != tc.duration {
				t.Errorf("duration: got %d, want %d", cmd.Duration, tc.duration)
			}
			if cmd.ByItemOption != tc.byItemOption {
				t.Errorf("byItemOption: got %v, want %v", cmd.ByItemOption, tc.byItemOption)
			}
		})
	}
}

// TestSetCommandProviderSetsTransactionId verifies that SetCommandProvider
// assigns a fresh, non-nil transaction id to every command it emits.
func TestSetCommandProviderSetsTransactionId(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	msgs1, err := SetCommandProvider(1000, f, 8, -1, false)()
	if err != nil {
		t.Fatalf("SetCommandProvider error: %v", err)
	}
	var cmd1 expression2.Command
	if err := json.Unmarshal(msgs1[0].Value, &cmd1); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	msgs2, err := SetCommandProvider(1000, f, 8, -1, false)()
	if err != nil {
		t.Fatalf("SetCommandProvider error: %v", err)
	}
	var cmd2 expression2.Command
	if err := json.Unmarshal(msgs2[0].Value, &cmd2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if cmd1.TransactionId == uuid.Nil {
		t.Errorf("cmd1.TransactionId: got nil uuid")
	}
	if cmd2.TransactionId == uuid.Nil {
		t.Errorf("cmd2.TransactionId: got nil uuid")
	}
	if cmd1.TransactionId == cmd2.TransactionId {
		t.Errorf("expected distinct transaction ids, got the same value for both commands: %s", cmd1.TransactionId)
	}
}
