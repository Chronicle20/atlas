package marriage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandSerialization(t *testing.T) {
	tests := []struct {
		name    string
		command interface{}
	}{
		{
			name: "ProposeCommand",
			command: Command[ProposeBody]{
				CharacterId: 12345,
				Type:        CommandMarriagePropose,
				Body: ProposeBody{
					TargetCharacterId: 67890,
				},
			},
		},
		{
			name: "AcceptCommand",
			command: Command[AcceptBody]{
				CharacterId: 67890,
				Type:        CommandMarriageAccept,
				Body: AcceptBody{
					ProposalId: 1,
				},
			},
		},
		{
			name: "DeclineCommand",
			command: Command[DeclineBody]{
				CharacterId: 67890,
				Type:        CommandMarriageDecline,
				Body: DeclineBody{
					ProposalId: 1,
				},
			},
		},
		{
			name: "DivorceCommand",
			command: Command[DivorceBody]{
				CharacterId: 12345,
				Type:        CommandMarriageDivorce,
				Body: DivorceBody{
					MarriageId: 1,
				},
			},
		},
		{
			name: "ScheduleCeremonyCommand",
			command: Command[ScheduleCeremonyBody]{
				CharacterId: 12345,
				Type:        CommandCeremonySchedule,
				Body: ScheduleCeremonyBody{
					MarriageId:  1,
					ScheduledAt: time.Now(),
					Invitees:    []uint32{111, 222, 333},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			data, err := json.Marshal(tt.command)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Test that the JSON contains expected fields
			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)
			
			assert.Contains(t, result, "characterId")
			assert.Contains(t, result, "type")
			assert.Contains(t, result, "body")
		})
	}
}

func TestEventSerialization(t *testing.T) {
	tests := []struct {
		name  string
		event interface{}
	}{
		{
			name: "ProposalCreatedEvent",
			event: Event[ProposalCreatedBody]{
				CharacterId: 12345,
				Type:        EventProposalCreated,
				Body: ProposalCreatedBody{
					ProposalId:        1,
					ProposerId:        12345,
					TargetCharacterId: 67890,
					ProposedAt:        time.Now(),
					ExpiresAt:         time.Now().Add(24 * time.Hour),
				},
			},
		},
		{
			name: "ProposalAcceptedEvent",
			event: Event[ProposalAcceptedBody]{
				CharacterId: 67890,
				Type:        EventProposalAccepted,
				Body: ProposalAcceptedBody{
					ProposalId:        1,
					ProposerId:        12345,
					TargetCharacterId: 67890,
					AcceptedAt:        time.Now(),
				},
			},
		},
		{
			name: "MarriageCreatedEvent",
			event: Event[MarriageCreatedBody]{
				CharacterId: 12345,
				Type:        EventMarriageCreated,
				Body: MarriageCreatedBody{
					MarriageId:   1,
					CharacterId1: 12345,
					CharacterId2: 67890,
					MarriedAt:    time.Now(),
				},
			},
		},
		{
			name: "MarriageDivorcedEvent",
			event: Event[MarriageDivorcedBody]{
				CharacterId: 12345,
				Type:        EventMarriageDivorced,
				Body: MarriageDivorcedBody{
					MarriageId:   1,
					CharacterId1: 12345,
					CharacterId2: 67890,
					DivorcedAt:   time.Now(),
					InitiatedBy:  12345,
				},
			},
		},
		{
			name: "CeremonyScheduledEvent",
			event: Event[CeremonyScheduledBody]{
				CharacterId: 12345,
				Type:        EventCeremonyScheduled,
				Body: CeremonyScheduledBody{
					CeremonyId:   1,
					MarriageId:   1,
					CharacterId1: 12345,
					CharacterId2: 67890,
					ScheduledAt:  time.Now(),
					Invitees:     []uint32{111, 222, 333},
				},
			},
		},
		{
			name: "MarriageErrorEvent",
			event: Event[MarriageErrorBody]{
				CharacterId: 12345,
				Type:        EventMarriageError,
				Body: MarriageErrorBody{
					ErrorType:   ErrorTypeProposal,
					ErrorCode:   ErrorCodeAlreadyMarried,
					Message:     "Character is already married",
					CharacterId: 12345,
					Context:     "propose",
					Timestamp:   time.Now(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			data, err := json.Marshal(tt.event)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Test that the JSON contains expected fields
			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)
			
			assert.Contains(t, result, "characterId")
			assert.Contains(t, result, "type")
			assert.Contains(t, result, "body")
		})
	}
}

func TestCommandTypes(t *testing.T) {
	expectedCommands := []string{
		CommandMarriagePropose,
		CommandMarriageAccept,
		CommandMarriageDecline,
		CommandMarriageCancel,
		CommandMarriageDivorce,
		CommandCeremonySchedule,
		CommandCeremonyStart,
		CommandCeremonyComplete,
		CommandCeremonyCancel,
		CommandCeremonyPostpone,
		CommandCeremonyReschedule,
		CommandCeremonyAddInvitee,
		CommandCeremonyRemoveInvitee,
		CommandCeremonyAdvanceState,
	}

	for _, cmd := range expectedCommands {
		assert.NotEmpty(t, cmd, "Command type should not be empty")
	}
}

func TestEventTypes(t *testing.T) {
	expectedEvents := []string{
		EventProposalCreated,
		EventProposalAccepted,
		EventProposalDeclined,
		EventProposalExpired,
		EventProposalCancelled,
		EventMarriageCreated,
		EventMarriageDivorced,
		EventMarriageDeleted,
		EventCeremonyScheduled,
		EventCeremonyStarted,
		EventCeremonyCompleted,
		EventCeremonyPostponed,
		EventCeremonyCancelled,
		EventCeremonyRescheduled,
		EventMarriageError,
	}

	for _, event := range expectedEvents {
		assert.NotEmpty(t, event, "Event type should not be empty")
	}
}

func TestErrorTypes(t *testing.T) {
	expectedErrorTypes := []string{
		ErrorTypeProposal,
		ErrorTypeMarriage,
		ErrorTypeCeremony,
		ErrorTypeValidation,
		ErrorTypeCooldown,
		ErrorTypeEligibility,
		ErrorTypeNotFound,
		ErrorTypeAlreadyExists,
		ErrorTypeStateTransition,
		ErrorTypeInviteeLimit,
		ErrorTypeDisconnectionTimeout,
	}

	for _, errorType := range expectedErrorTypes {
		assert.NotEmpty(t, errorType, "Error type should not be empty")
	}
}

func TestErrorCodes(t *testing.T) {
	expectedErrorCodes := []string{
		ErrorCodeAlreadyMarried,
		ErrorCodeAlreadyEngaged,
		ErrorCodeInsufficientLevel,
		ErrorCodeSelfProposal,
		ErrorCodeGlobalCooldown,
		ErrorCodeTargetCooldown,
		ErrorCodeProposalExpired,
		ErrorCodeProposalNotFound,
		ErrorCodeMarriageNotFound,
		ErrorCodeCeremonyNotFound,
		ErrorCodeInvalidState,
		ErrorCodeInviteeLimitExceeded,
		ErrorCodeInviteeAlreadyInvited,
		ErrorCodeInviteeNotFound,
		ErrorCodePartnerDisconnected,
		ErrorCodeCeremonyTimeout,
		ErrorCodeConcurrentProposal,
		ErrorCodeTenantMismatch,
	}

	for _, errorCode := range expectedErrorCodes {
		assert.NotEmpty(t, errorCode, "Error code should not be empty")
	}
}

func TestTopicEnvironmentVariables(t *testing.T) {
	assert.Equal(t, "COMMAND_TOPIC_MARRIAGE", EnvCommandTopic)
	assert.Equal(t, "EVENT_TOPIC_MARRIAGE_STATUS", EnvEventTopicStatus)
}

func TestSpecificCommandDeserialization(t *testing.T) {
	// Test that a command can be deserialized correctly
	proposeCmd := Command[ProposeBody]{
		CharacterId: 12345,
		Type:        CommandMarriagePropose,
		Body: ProposeBody{
			TargetCharacterId: 67890,
		},
	}

	// Serialize
	data, err := json.Marshal(proposeCmd)
	require.NoError(t, err)

	// Deserialize
	var deserializedCmd Command[ProposeBody]
	err = json.Unmarshal(data, &deserializedCmd)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, proposeCmd.CharacterId, deserializedCmd.CharacterId)
	assert.Equal(t, proposeCmd.Type, deserializedCmd.Type)
	assert.Equal(t, proposeCmd.Body.TargetCharacterId, deserializedCmd.Body.TargetCharacterId)
}

func TestSpecificEventDeserialization(t *testing.T) {
	// Test that an event can be deserialized correctly
	createdEvent := Event[ProposalCreatedBody]{
		CharacterId: 12345,
		Type:        EventProposalCreated,
		Body: ProposalCreatedBody{
			ProposalId:        1,
			ProposerId:        12345,
			TargetCharacterId: 67890,
			ProposedAt:        time.Now().UTC().Truncate(time.Second),
			ExpiresAt:         time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second),
		},
	}

	// Serialize
	data, err := json.Marshal(createdEvent)
	require.NoError(t, err)

	// Deserialize
	var deserializedEvent Event[ProposalCreatedBody]
	err = json.Unmarshal(data, &deserializedEvent)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, createdEvent.CharacterId, deserializedEvent.CharacterId)
	assert.Equal(t, createdEvent.Type, deserializedEvent.Type)
	assert.Equal(t, createdEvent.Body.ProposalId, deserializedEvent.Body.ProposalId)
	assert.Equal(t, createdEvent.Body.ProposerId, deserializedEvent.Body.ProposerId)
	assert.Equal(t, createdEvent.Body.TargetCharacterId, deserializedEvent.Body.TargetCharacterId)
	assert.Equal(t, createdEvent.Body.ProposedAt, deserializedEvent.Body.ProposedAt)
	assert.Equal(t, createdEvent.Body.ExpiresAt, deserializedEvent.Body.ExpiresAt)
}