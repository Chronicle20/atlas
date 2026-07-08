package game

import (
	"atlas-rps/kafka/message/rps"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// gameOpenedEventProvider builds the GameOpened event emitted when a new RPS
// session is opened for a character at an NPC.
func gameOpenedEventProvider(characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &rps.Event[rps.GameOpenedEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		ChannelId:   channelId,
		Type:        rps.EventTypeGameOpened,
		Body: rps.GameOpenedEventBody{
			NpcId: npcId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// roundResultEventProvider builds the RoundResult event emitted after a
// round is adjudicated, carrying the opponent's throw, the outcome, the
// resulting rung, and any prize resolved at that rung.
func roundResultEventProvider(characterId uint32, worldId world.Id, channelId channel.Id, opponentThrow Throw, outcome Outcome, rung int, prize rps.Prize) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &rps.Event[rps.RoundResultEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		ChannelId:   channelId,
		Type:        rps.EventTypeRoundResult,
		Body: rps.RoundResultEventBody{
			OpponentThrow: byte(opponentThrow),
			Outcome:       int(outcome),
			Rung:          rung,
			Prize:         prize,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// gameEndedEventProvider builds the GameEnded event emitted when a session
// terminates. grantedPrize should be non-nil only when reason is
// rps.ReasonCollected.
func gameEndedEventProvider(characterId uint32, worldId world.Id, channelId channel.Id, reason string, grantedPrize *rps.Prize) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &rps.Event[rps.GameEndedEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		ChannelId:   channelId,
		Type:        rps.EventTypeGameEnded,
		Body: rps.GameEndedEventBody{
			Reason:       reason,
			GrantedPrize: grantedPrize,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
