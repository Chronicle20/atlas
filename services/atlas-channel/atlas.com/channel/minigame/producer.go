package minigame

import (
	minigame2 "atlas-channel/kafka/message/minigame"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func CreateCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32, roomType byte, title string, private bool, password string, pieceType byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &minigame2.Command[minigame2.CreateCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   characterId,
		Type:          minigame2.CommandTypeCreate,
		Body: minigame2.CreateCommandBody{
			RoomType:  roomType,
			Title:     title,
			Private:   private,
			Password:  password,
			PieceType: pieceType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func VisitCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32, roomId uint32, password string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &minigame2.Command[minigame2.VisitCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   characterId,
		Type:          minigame2.CommandTypeVisit,
		Body: minigame2.VisitCommandBody{
			RoomId:   roomId,
			Password: password,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func LeaveCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &minigame2.Command[minigame2.EmptyCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   characterId,
		Type:          minigame2.CommandTypeLeave,
		Body:          minigame2.EmptyCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChatCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &minigame2.Command[minigame2.ChatCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   characterId,
		Type:          minigame2.CommandTypeChat,
		Body: minigame2.ChatCommandBody{
			Message: message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ReadyCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	return emptyBodyCommandProvider(transactionId, f, characterId, minigame2.CommandTypeReady)
}

func UnreadyCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	return emptyBodyCommandProvider(transactionId, f, characterId, minigame2.CommandTypeUnready)
}

func StartCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	return emptyBodyCommandProvider(transactionId, f, characterId, minigame2.CommandTypeStart)
}

func GiveUpCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	return emptyBodyCommandProvider(transactionId, f, characterId, minigame2.CommandTypeGiveUp)
}

func RequestTieCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	return emptyBodyCommandProvider(transactionId, f, characterId, minigame2.CommandTypeRequestTie)
}

func RequestRetreatCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	return emptyBodyCommandProvider(transactionId, f, characterId, minigame2.CommandTypeRequestRetreat)
}

func ExpelCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	return emptyBodyCommandProvider(transactionId, f, characterId, minigame2.CommandTypeExpel)
}

func SkipCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	return emptyBodyCommandProvider(transactionId, f, characterId, minigame2.CommandTypeSkip)
}

func MoveStoneCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32, x uint32, y uint32, stoneType byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &minigame2.Command[minigame2.MoveStoneCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   characterId,
		Type:          minigame2.CommandTypeMoveStone,
		Body: minigame2.MoveStoneCommandBody{
			X:         x,
			Y:         y,
			StoneType: stoneType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func FlipCardCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32, first bool, cardIndex byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &minigame2.Command[minigame2.FlipCardCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   characterId,
		Type:          minigame2.CommandTypeFlipCard,
		Body: minigame2.FlipCardCommandBody{
			First:     first,
			CardIndex: cardIndex,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AnswerTieCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32, accept bool) model.Provider[[]kafka.Message] {
	return answerCommandProvider(transactionId, f, characterId, minigame2.CommandTypeAnswerTie, accept)
}

func AnswerRetreatCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32, accept bool) model.Provider[[]kafka.Message] {
	return answerCommandProvider(transactionId, f, characterId, minigame2.CommandTypeAnswerRetreat, accept)
}

func ExitAfterGameCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32, cancel bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	commandType := minigame2.CommandTypeExitAfterGame
	if cancel {
		commandType = minigame2.CommandTypeCancelExitAfterGame
	}
	value := &minigame2.Command[minigame2.EmptyCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   characterId,
		Type:          commandType,
		Body:          minigame2.EmptyCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func emptyBodyCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32, commandType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &minigame2.Command[minigame2.EmptyCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   characterId,
		Type:          commandType,
		Body:          minigame2.EmptyCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func answerCommandProvider(transactionId uuid.UUID, f field.Model, characterId uint32, commandType string, accept bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &minigame2.Command[minigame2.AnswerCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   characterId,
		Type:          commandType,
		Body: minigame2.AnswerCommandBody{
			Accept: accept,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
