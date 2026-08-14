package saga

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// TestWorldTransferActionsAllResolveToHandlers guards against the class of
// bug that leaves a saga wedged indistinguishably from a slow downstream: an
// action with no case in GetHandler's switch. See task-227 Task 13.
func TestWorldTransferActionsAllResolveToHandlers(t *testing.T) {
	h := &HandlerImpl{}
	for _, a := range []Action{
		ValidateWorldTransfer, LeaveGuildForTransfer, LeavePartyForTransfer,
		SeverBuddiesForTransfer, ChangeCharacterWorld,
	} {
		if _, ok := h.GetHandler(a); !ok {
			t.Fatalf("action %s has no handler", a)
		}
	}
}

// TestWorldTransferStepsUnmarshalToConcretePayloads asserts payload unmarshal
// produces the concrete type, not map[string]interface{} — every handler
// type-asserts and returns "invalid payload" otherwise.
func TestWorldTransferStepsUnmarshalToConcretePayloads(t *testing.T) {
	raw := `{"action":"change_character_world","payload":{"characterId":1,"sourceWorldId":0,"destinationWorldId":1,"pendingChangeId":"` + uuid.New().String() + `"}}`
	var st Step[any]
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := st.Payload().(ChangeCharacterWorldPayload); !ok {
		t.Fatalf("payload type = %T, want ChangeCharacterWorldPayload", st.Payload())
	}
}

// TestWorldTransferAllFivePayloadsUnmarshalToConcreteTypes extends the brief's
// single-payload check across all five actions, since a switch-case typo
// (e.g. wrong payload type on the right action) would not be caught by
// TestWorldTransferStepsUnmarshalToConcretePayloads alone.
func TestWorldTransferAllFivePayloadsUnmarshalToConcreteTypes(t *testing.T) {
	pendingChangeId := uuid.New().String()
	cases := []struct {
		action  string
		payload string
		assert  func(t *testing.T, got any)
	}{
		{
			action:  "validate_world_transfer",
			payload: `{"characterId":1,"sourceWorldId":0,"destinationWorldId":1,"pendingChangeId":"` + pendingChangeId + `"}`,
			assert: func(t *testing.T, got any) {
				if _, ok := got.(ValidateWorldTransferPayload); !ok {
					t.Fatalf("payload type = %T, want ValidateWorldTransferPayload", got)
				}
			},
		},
		{
			action:  "leave_guild_for_transfer",
			payload: `{"characterId":1,"worldId":0,"guildId":5,"title":3}`,
			assert: func(t *testing.T, got any) {
				if _, ok := got.(LeaveGuildForTransferPayload); !ok {
					t.Fatalf("payload type = %T, want LeaveGuildForTransferPayload", got)
				}
			},
		},
		{
			action:  "leave_party_for_transfer",
			payload: `{"characterId":1,"worldId":0,"partyId":9}`,
			assert: func(t *testing.T, got any) {
				if _, ok := got.(LeavePartyForTransferPayload); !ok {
					t.Fatalf("payload type = %T, want LeavePartyForTransferPayload", got)
				}
			},
		},
		{
			action:  "sever_buddies_for_transfer",
			payload: `{"characterId":1,"worldId":0,"buddyIds":[2,3]}`,
			assert: func(t *testing.T, got any) {
				if _, ok := got.(SeverBuddiesForTransferPayload); !ok {
					t.Fatalf("payload type = %T, want SeverBuddiesForTransferPayload", got)
				}
			},
		},
		{
			action:  "change_character_world",
			payload: `{"characterId":1,"sourceWorldId":0,"destinationWorldId":1,"pendingChangeId":"` + pendingChangeId + `"}`,
			assert: func(t *testing.T, got any) {
				if _, ok := got.(ChangeCharacterWorldPayload); !ok {
					t.Fatalf("payload type = %T, want ChangeCharacterWorldPayload", got)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			raw := `{"action":"` + tc.action + `","payload":` + tc.payload + `}`
			var st Step[any]
			if err := json.Unmarshal([]byte(raw), &st); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			tc.assert(t, st.Payload())
		})
	}
}
