package rps

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestCommandSelectRoundTrip(t *testing.T) {
	cmd := Command[SelectCommandBody]{
		CharacterId: 12345,
		WorldId:     world.Id(0),
		ChannelId:   channel.Id(1),
		Type:        CommandTypeSelect,
		Body: SelectCommandBody{
			Throw: 2,
		},
	}

	b, err := json.Marshal(cmd)
	assert.NoError(t, err)

	var out Command[SelectCommandBody]
	err = json.Unmarshal(b, &out)
	assert.NoError(t, err)

	assert.Equal(t, CommandTypeSelect, out.Type)
	assert.Equal(t, uint32(12345), out.CharacterId)
	assert.Equal(t, world.Id(0), out.WorldId)
	assert.Equal(t, channel.Id(1), out.ChannelId)
	assert.Equal(t, byte(2), out.Body.Throw)
}

func TestCommandTypeConstants(t *testing.T) {
	assert.Equal(t, "SELECT", CommandTypeSelect)
	assert.Equal(t, "CONTINUE", CommandTypeContinue)
	assert.Equal(t, "COLLECT", CommandTypeCollect)
	assert.Equal(t, "QUIT", CommandTypeQuit)
}

func TestEventGameOpenedRoundTrip(t *testing.T) {
	evt := Event[GameOpenedEventBody]{
		CharacterId: 55,
		WorldId:     world.Id(0),
		ChannelId:   channel.Id(2),
		Type:        EventTypeGameOpened,
		Body: GameOpenedEventBody{
			NpcId: 9010000,
			Ante:  1000,
		},
	}

	b, err := json.Marshal(evt)
	assert.NoError(t, err)

	var out Event[GameOpenedEventBody]
	err = json.Unmarshal(b, &out)
	assert.NoError(t, err)

	assert.Equal(t, EventTypeGameOpened, out.Type)
	assert.Equal(t, uint32(55), out.CharacterId)
	assert.Equal(t, uint32(9010000), out.Body.NpcId)
	assert.Equal(t, uint32(1000), out.Body.Ante)
}

func TestEventRoundResultRoundTrip(t *testing.T) {
	evt := Event[RoundResultEventBody]{
		CharacterId: 55,
		WorldId:     world.Id(0),
		ChannelId:   channel.Id(2),
		Type:        EventTypeRoundResult,
		Body: RoundResultEventBody{
			OpponentThrow: 1,
			Outcome:       2,
			Rung:          3,
			Prize: Prize{
				ItemId:   item.Id(4031059),
				Quantity: 1,
				Meso:     0,
			},
		},
	}

	b, err := json.Marshal(evt)
	assert.NoError(t, err)

	var out Event[RoundResultEventBody]
	err = json.Unmarshal(b, &out)
	assert.NoError(t, err)

	assert.Equal(t, EventTypeRoundResult, out.Type)
	assert.Equal(t, byte(1), out.Body.OpponentThrow)
	assert.Equal(t, 2, out.Body.Outcome)
	assert.Equal(t, 3, out.Body.Rung)
	assert.Equal(t, item.Id(4031059), out.Body.Prize.ItemId)
	assert.Equal(t, uint32(1), out.Body.Prize.Quantity)
	assert.Equal(t, uint32(0), out.Body.Prize.Meso)
}

func TestEventGameEndedRoundTrip_WithPrize(t *testing.T) {
	evt := Event[GameEndedEventBody]{
		CharacterId: 55,
		WorldId:     world.Id(0),
		ChannelId:   channel.Id(2),
		Type:        EventTypeGameEnded,
		Body: GameEndedEventBody{
			Reason: ReasonCollected,
			GrantedPrize: &Prize{
				ItemId:   item.Id(4031059),
				Quantity: 1,
				Meso:     100,
			},
		},
	}

	b, err := json.Marshal(evt)
	assert.NoError(t, err)

	var out Event[GameEndedEventBody]
	err = json.Unmarshal(b, &out)
	assert.NoError(t, err)

	assert.Equal(t, EventTypeGameEnded, out.Type)
	assert.Equal(t, ReasonCollected, out.Body.Reason)
	if assert.NotNil(t, out.Body.GrantedPrize) {
		assert.Equal(t, item.Id(4031059), out.Body.GrantedPrize.ItemId)
		assert.Equal(t, uint32(1), out.Body.GrantedPrize.Quantity)
		assert.Equal(t, uint32(100), out.Body.GrantedPrize.Meso)
	}
}

func TestEventGameEndedRoundTrip_WithoutPrize(t *testing.T) {
	evt := Event[GameEndedEventBody]{
		CharacterId: 55,
		WorldId:     world.Id(0),
		ChannelId:   channel.Id(2),
		Type:        EventTypeGameEnded,
		Body: GameEndedEventBody{
			Reason: ReasonQuit,
		},
	}

	b, err := json.Marshal(evt)
	assert.NoError(t, err)

	var out Event[GameEndedEventBody]
	err = json.Unmarshal(b, &out)
	assert.NoError(t, err)

	assert.Equal(t, EventTypeGameEnded, out.Type)
	assert.Equal(t, ReasonQuit, out.Body.Reason)
	assert.Nil(t, out.Body.GrantedPrize)
}

func TestEnvTopicConstants(t *testing.T) {
	assert.Equal(t, "COMMAND_TOPIC_RPS", EnvCommandTopic)
	assert.Equal(t, "EVENT_TOPIC_RPS", EnvEventTopic)
}
