package shop

import (
	"testing"
	"time"

	msg "atlas-merchant/message"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The shop detail payload must carry the persisted shop messages so the
// channel can replay them into the owner's management view (audit F10).
func TestTransformWithMessages(t *testing.T) {
	shopId := uuid.New()
	m, err := NewBuilder().
		SetId(shopId).
		SetCharacterId(1000).
		SetShopType(HiredMerchant).
		SetState(Open).
		SetTitle("Shop").
		SetMapId(910000001).
		SetPermitItemId(5030000).
		Build()
	require.NoError(t, err)

	sent := time.Now()
	message, err := msg.NewBuilder().SetId(uuid.New()).SetShopId(shopId).SetCharacterId(2000).SetContent("hello").SetSentAt(sent).Build()
	require.NoError(t, err)
	messages := []msg.Model{message}

	rm, err := TransformWithMessages(messages)(m)
	require.NoError(t, err)
	require.Len(t, rm.Messages, 1)
	assert.Equal(t, uint32(2000), rm.Messages[0].CharacterId)
	assert.Equal(t, "hello", rm.Messages[0].Content)
	assert.WithinDuration(t, sent, rm.Messages[0].SentAt, time.Second)
}
