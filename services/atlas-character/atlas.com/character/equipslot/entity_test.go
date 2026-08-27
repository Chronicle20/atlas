package equipslot

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMakeToEntityRoundTrip proves Make (entity.go, DOM-03) and Model.ToEntity
// (entity.go, DOM-02) are inverses for every field the domain Model owns.
// TransactionId is deliberately excluded -- it is a write-path idempotency
// key that never leaves the persistence layer, so ToEntity always emits the
// zero UUID for it regardless of what the source Entity carried.
func TestMakeToEntityRoundTrip(t *testing.T) {
	e := Entity{
		Id:            uuid.New(),
		TenantId:      uuid.New(),
		CharacterId:   42,
		SlotIndex:     7,
		ExpiresAt:     time.Now().Add(30 * 24 * time.Hour).UTC().Truncate(time.Second),
		TransactionId: uuid.New(),
	}

	m, err := Make(e)
	require.NoError(t, err)
	assert.Equal(t, e.Id, m.Id())
	assert.Equal(t, e.CharacterId, m.CharacterId())
	assert.Equal(t, e.SlotIndex, m.SlotIndex())
	assert.Equal(t, e.ExpiresAt, m.ExpiresAt())

	back := m.ToEntity()
	assert.Equal(t, e.Id, back.Id)
	assert.Equal(t, e.TenantId, back.TenantId)
	assert.Equal(t, e.CharacterId, back.CharacterId)
	assert.Equal(t, e.SlotIndex, back.SlotIndex)
	assert.Equal(t, e.ExpiresAt, back.ExpiresAt)
	assert.Equal(t, uuid.Nil, back.TransactionId, "ToEntity must not resurrect the write-path dedupe key")
}
