package saga

import (
	sagaMsg "atlas-saga-orchestrator/kafka/message/saga"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompletedStatusEventProviderNoteSendResults verifies that a completed
// note_send saga's COMPLETED status event carries the sending character's id
// in Body.Results["characterId"], so atlas-channel can announce MEMO_RESULT
// SEND_SUCCESS to the sender's session (the completed body has no
// CharacterId field of its own; Results is the existing mechanism — see
// extractCharacterCreationResults).
func TestCompletedStatusEventProviderNoteSendResults(t *testing.T) {
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(NoteSend).
		SetInitiatedBy("test").
		AddStep("consume_note_item", Completed, DestroyAsset, DestroyAssetPayload{CharacterId: 100, TemplateId: 5090000, Quantity: 1}).
		AddStep("create_note", Completed, CreateNote, CreateNotePayload{SenderId: 100, ReceiverId: 200, Message: "hi", Flag: 1}).
		Build()
	require.NoError(t, err)

	msgs, err := CompletedStatusEventProvider(s)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var e sagaMsg.StatusEvent[sagaMsg.StatusEventCompletedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &e))
	assert.Equal(t, "note_send", e.Body.SagaType)
	require.NotNil(t, e.Body.Results)
	assert.Equal(t, float64(100), e.Body.Results["characterId"], "sender characterId must ride in Results")
}
