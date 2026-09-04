package ops

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

func TestSendMessage(t *testing.T) {
	target := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()

	tests := []struct {
		name        string
		params      map[string]string
		wantErr     string
		wantPayload saga.SendMessagePayload
	}{
		{
			name:    "missing message",
			params:  map[string]string{},
			wantErr: `send_message: parameter "message" is required`,
		},
		{
			name:   "default type",
			params: map[string]string{"message": "hi"},
			wantPayload: saga.SendMessagePayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MessageType: "PINK_TEXT",
				Message:     "hi",
			},
		},
		{
			name:   "messageType key",
			params: map[string]string{"message": "hi", "messageType": "NOTICE"},
			wantPayload: saga.SendMessagePayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MessageType: "NOTICE",
				Message:     "hi",
			},
		},
		{
			name:   "type key",
			params: map[string]string{"message": "hi", "type": "NOTICE"},
			wantPayload: saga.SendMessagePayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MessageType: "NOTICE",
				Message:     "hi",
			},
		},
		{
			name:   "messageType wins",
			params: map[string]string{"message": "hi", "messageType": "NOTICE", "type": "POP_UP"},
			wantPayload: saga.SendMessagePayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MessageType: "NOTICE",
				Message:     "hi",
			},
		},
		{
			name:   "numeric 5 via type",
			params: map[string]string{"message": "hi", "type": "5"},
			wantPayload: saga.SendMessagePayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MessageType: "PINK_TEXT",
				Message:     "hi",
			},
		},
		{
			name:   "numeric 6 via type",
			params: map[string]string{"message": "hi", "type": "6"},
			wantPayload: saga.SendMessagePayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MessageType: "BLUE_TEXT",
				Message:     "hi",
			},
		},
		{
			name:   "numeric 5 via messageType",
			params: map[string]string{"message": "hi", "messageType": "5"},
			wantPayload: saga.SendMessagePayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MessageType: "PINK_TEXT",
				Message:     "hi",
			},
		},
		{
			name:   "numeric 6 via messageType",
			params: map[string]string{"message": "hi", "messageType": "6"},
			wantPayload: saga.SendMessagePayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MessageType: "BLUE_TEXT",
				Message:     "hi",
			},
		},
		{
			name:   "unknown numeric passes through",
			params: map[string]string{"message": "hi", "type": "9"},
			wantPayload: saga.SendMessagePayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MessageType: "9",
				Message:     "hi",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := SendMessage(tt.params, DirectResolver{}, target, 7)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("got error %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if step.Action() != saga.SendMessage {
				t.Fatalf("got action %v, want %v", step.Action(), saga.SendMessage)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.SendMessagePayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload != tt.wantPayload {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}

func TestSendMessageResolvesThroughResolver(t *testing.T) {
	target := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()
	rec := &recordingResolver{}

	_, err := SendMessage(map[string]string{"message": "hi", "messageType": "NOTICE"}, rec, target, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rec.params) != 2 {
		t.Fatalf("got %d resolver calls, want 2: params=%v rawVals=%v", len(rec.params), rec.params, rec.rawVals)
	}
	if rec.params[0] != "message" || rec.rawVals[0] != "hi" {
		t.Fatalf("call 0 = (%q, %q), want (message, hi)", rec.params[0], rec.rawVals[0])
	}
	if rec.params[1] != "messageType" || rec.rawVals[1] != "NOTICE" {
		t.Fatalf("call 1 = (%q, %q), want (messageType, NOTICE)", rec.params[1], rec.rawVals[1])
	}
}
