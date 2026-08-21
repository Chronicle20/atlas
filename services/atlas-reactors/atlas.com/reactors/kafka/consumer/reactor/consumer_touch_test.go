package reactor

import (
	"atlas-reactors/reactor"
	"atlas-reactors/reactor/mock"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

func TestHandleTouch(t *testing.T) {
	type touchCall struct {
		reactorId   uint32
		characterId uint32
		touching    bool
	}

	tests := []struct {
		name        string
		commandType string
		touching    bool
		wantCalls   int
	}{
		{name: "routed", commandType: reactor.CommandTypeTouch, touching: true, wantCalls: 1},
		{name: "wrong type ignored", commandType: reactor.CommandTypeHit, touching: true, wantCalls: 0},
		{name: "leaving forwarded", commandType: reactor.CommandTypeTouch, touching: false, wantCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, _ := test.NewNullLogger()

			var calls []touchCall
			p := &mock.ProcessorMock{
				TouchFunc: func(reactorId uint32, characterId uint32, touching bool) error {
					calls = append(calls, touchCall{reactorId, characterId, touching})
					return nil
				},
			}

			c := reactor.Command[reactor.TouchCommandBody]{
				Type: tt.commandType,
				Body: reactor.TouchCommandBody{
					ReactorId:   42,
					CharacterId: 1000,
					Touching:    tt.touching,
				},
			}

			handleTouchFor(p, l, c)

			if len(calls) != tt.wantCalls {
				t.Fatalf("Touch calls = %d, want %d", len(calls), tt.wantCalls)
			}
			if tt.wantCalls == 1 {
				got := calls[0]
				if got.reactorId != 42 {
					t.Errorf("reactorId = %d, want 42", got.reactorId)
				}
				if got.characterId != 1000 {
					t.Errorf("characterId = %d, want 1000", got.characterId)
				}
				if got.touching != tt.touching {
					t.Errorf("touching = %v, want %v", got.touching, tt.touching)
				}
			}
		})
	}
}
