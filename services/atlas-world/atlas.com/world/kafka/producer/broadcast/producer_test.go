package broadcast

import (
	"encoding/json"
	"testing"

	message "atlas-world/kafka/message/broadcast"

	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// TestQueuedStatusEventProvider proves the QUEUED status event carries the
// worldId-derived key (pinning the single-partition-ordering-per-world
// property) and round-trips every field.
func TestQueuedStatusEventProvider(t *testing.T) {
	worldId := world.Id(7)
	msgs, err := QueuedStatusEventProvider(worldId, "TV", 12345, 42)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, kproducer.CreateKey(int(worldId)), msgs[0].Key)

	var e message.StatusEvent
	require.NoError(t, json.Unmarshal(msgs[0].Value, &e))
	require.Equal(t, message.StatusTypeQueued, e.Type)
	require.Equal(t, "TV", e.Family)
	require.Equal(t, byte(worldId), e.WorldId)
	require.Equal(t, uint32(12345), e.CharacterId)
	require.Equal(t, uint32(42), e.WaitSeconds)
}

// TestEndedStatusEventProvider proves the ENDED status event carries the
// worldId-derived key and round-trips Family/WorldId/CharacterId.
func TestEndedStatusEventProvider(t *testing.T) {
	worldId := world.Id(3)
	msgs, err := EndedStatusEventProvider(worldId, "AVATAR", 999)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, kproducer.CreateKey(int(worldId)), msgs[0].Key)

	var e message.StatusEvent
	require.NoError(t, json.Unmarshal(msgs[0].Value, &e))
	require.Equal(t, message.StatusTypeEnded, e.Type)
	require.Equal(t, "AVATAR", e.Family)
	require.Equal(t, byte(worldId), e.WorldId)
	require.Equal(t, uint32(999), e.CharacterId)
}

// TestStartedStatusEventProvider proves the STARTED status event carries the
// worldId-derived key and every StartedPayload field lands in its matching
// StatusEvent slot — a fully-populated fixture (every field distinct and
// non-zero, including the pointer-valued ReceiverLook) so a future field
// swap (e.g. SenderName/SenderMedal, both string-typed) fails this test
// even though go build/go vet would stay green.
func TestStartedStatusEventProvider(t *testing.T) {
	worldId := world.Id(11)
	p := message.StartedPayload{
		CharacterId:     555,
		DurationSeconds: 30,
		ChannelId:       2,
		SenderName:      "sender-name",
		SenderMedal:     "sender-medal",
		Messages:        []string{"line-one", "line-two"},
		WhispersOn:      true,
		ItemId:          5390000,
		TvMessageType:   "HEART",
		SenderLook: sharedsaga.AvatarSnapshot{
			Gender:       0,
			SkinColor:    3,
			Face:         20000,
			Hair:         30000,
			Equips:       map[int16]uint32{-1: 1002140},
			MaskedEquips: map[int16]uint32{-5: 1040002},
			Pets:         map[int8]uint32{0: 5000001},
		},
		ReceiverName: "receiver-name",
		ReceiverLook: &sharedsaga.AvatarSnapshot{
			Gender:       1,
			SkinColor:    4,
			Face:         21000,
			Hair:         31000,
			Equips:       map[int16]uint32{-1: 1002141},
			MaskedEquips: map[int16]uint32{-101: 1002999},
			Pets:         map[int8]uint32{1: 5000002},
		},
	}

	msgs, err := StartedStatusEventProvider(worldId, "TV", p)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, kproducer.CreateKey(int(worldId)), msgs[0].Key)

	var e message.StatusEvent
	require.NoError(t, json.Unmarshal(msgs[0].Value, &e))
	require.Equal(t, message.StatusTypeStarted, e.Type)
	require.Equal(t, "TV", e.Family)
	require.Equal(t, byte(worldId), e.WorldId)
	require.Equal(t, p.CharacterId, e.CharacterId)
	require.Equal(t, p.DurationSeconds, e.TotalWaitSeconds)
	require.Equal(t, p.ChannelId, e.ChannelId)
	require.Equal(t, p.SenderName, e.SenderName)
	require.Equal(t, p.SenderMedal, e.SenderMedal)
	require.Equal(t, p.Messages, e.Messages)
	require.Equal(t, p.WhispersOn, e.WhispersOn)
	require.Equal(t, p.ItemId, e.ItemId)
	require.Equal(t, p.TvMessageType, e.TvMessageType)
	require.Equal(t, p.SenderLook, e.SenderLook)
	require.Equal(t, p.ReceiverName, e.ReceiverName)
	require.NotNil(t, e.ReceiverLook)
	require.Equal(t, *p.ReceiverLook, *e.ReceiverLook)
}
