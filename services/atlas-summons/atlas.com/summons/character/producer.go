// Package character declares atlas-summons' local view of the atlas-character
// command contract (COMMAND_TOPIC_CHARACTER) and the provider used by the
// Beholder aura sweep to heal a summon's owner.
//
// Services in this monorepo never import one another, so the envelope and body
// shapes here are re-declared to match the atlas-character consumer at
// services/atlas-character/atlas.com/character/kafka/message/character/kafka.go
// (Command[E] at lines 54-60, ChangeHPBody at lines 160-163; handled by
// handleChangeHP in .../kafka/consumer/character/consumer.go:251-258). The JSON
// tags MUST stay byte-identical to that consumer or the heal silently fails.
package character

import (
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

const (
	// EnvCommandTopic names the env var holding the atlas-character command topic.
	EnvCommandTopic = "COMMAND_TOPIC_CHARACTER"

	// CommandChangeHP applies a signed HP delta to a character. Mirrors
	// atlas-character CommandChangeHP.
	CommandChangeHP = "CHANGE_HP"
)

// Command is the atlas-character command envelope. Tags match
// character/kafka/message/character/kafka.go:54-60.
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// ChangeHPBody mirrors character/kafka/message/character/kafka.go:160-163.
type ChangeHPBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}

// changeHPProvider applies amount HP to characterId. Used by the Beholder aura
// sweep to heal the owner (FR-5.x). The key is the character id so all of a
// character's commands stay ordered on one partition.
func changeHPProvider(worldId world.Id, channelId channel.Id, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &Command[ChangeHPBody]{
		TransactionId: uuid.New(),
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          CommandChangeHP,
		Body: ChangeHPBody{
			ChannelId: channelId,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ChangeHPProvider is the exported entry point for the Beholder aura sweep.
func ChangeHPProvider(worldId world.Id, channelId channel.Id, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	return changeHPProvider(worldId, channelId, characterId, amount)
}
