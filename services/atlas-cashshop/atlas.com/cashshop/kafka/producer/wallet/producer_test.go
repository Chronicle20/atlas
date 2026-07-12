package wallet

import (
	"encoding/json"
	"testing"

	"atlas-cashshop/kafka/message/wallet"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestErrorStatusEventProvider proves the wallet ERROR ack is well-formed: type
// ERROR, keyed body echoing the transaction id and reason, so the orchestrator can
// fail the waiting saga step fast on a failed transactional adjust (task-102).
func TestErrorStatusEventProvider(t *testing.T) {
	tx := uuid.New()
	msgs, err := ErrorStatusEventProvider(20, tx, "record not found")()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var e wallet.StatusEvent[wallet.StatusEventErrorBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &e))
	require.Equal(t, wallet.StatusEventTypeError, e.Type)
	require.Equal(t, uint32(20), e.AccountId)
	require.Equal(t, tx, e.Body.TransactionId)
	require.Equal(t, "record not found", e.Body.Reason)
}
