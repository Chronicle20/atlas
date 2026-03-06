package writer

import (
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type CharacterInteractionMode string

type CharacterInteractionEnterErrorMode string

const (
	// CharacterInteraction CMiniRoomBaseDlg::OnPacketBase
	CharacterInteraction                                          = "CharacterInteraction"
	CharacterInteractionModeInvite       CharacterInteractionMode = "INVITE"        // 2
	CharacterInteractionModeInviteResult CharacterInteractionMode = "INVITE_RESULT" // 3
	CharacterInteractionModeEnter        CharacterInteractionMode = "ENTER"         // 4
	CharacterInteractionModeEnterResult  CharacterInteractionMode = "ENTER_RESULT"  // 5

	CharacterInteractionEnterErrorModeRoomClosed                CharacterInteractionEnterErrorMode = "ROOM_CLOSED"                   // 1
	CharacterInteractionEnterErrorModeFull                      CharacterInteractionEnterErrorMode = "FULL"                          // 2
	CharacterInteractionEnterErrorModeOtherRequests             CharacterInteractionEnterErrorMode = "OTHER_REQUESTS"                // 3
	CharacterInteractionEnterErrorModeNotWhenDead               CharacterInteractionEnterErrorMode = "NOT_WHEN_DEAD"                 // 4
	CharacterInteractionEnterErrorModeNotInEvent                CharacterInteractionEnterErrorMode = "NOT_IN_EVENT"                  // 5
	CharacterInteractionEnterErrorModeUnable                    CharacterInteractionEnterErrorMode = "UNABLE"                        // 6
	CharacterInteractionEnterErrorModeTradeNotAllowed           CharacterInteractionEnterErrorMode = "TRADE_NOT_ALLOWED"             // 7
	CharacterInteractionEnterErrorModeNotSameMap                CharacterInteractionEnterErrorMode = "NOT_SAME_MAP"                  // 9
	CharacterInteractionEnterErrorModeCannotOpenStoreNearPortal CharacterInteractionEnterErrorMode = "CANNOT_OPEN_STORE_NEAR_PORTAL" // 10
	CharacterInteractionEnterErrorModeCannotStartGameHere       CharacterInteractionEnterErrorMode = "CANNOT_START_GAME_HERE"        // 11
	CharacterInteractionEnterErrorModeCannotOpenStoreInChannel  CharacterInteractionEnterErrorMode = "CANNOT_OPEN_STORE_IN_CHANNEL"  // 12
	CharacterInteractionEnterErrorModeCannotOpenMiniRoomHere    CharacterInteractionEnterErrorMode = "CANNOT_OPEN_MINI_ROOM_HERE"    // 13
	CharacterInteractionEnterErrorModeCannotStartGameHere2      CharacterInteractionEnterErrorMode = "CANNOT_START_GAME_HERE_2"      // 14
	CharacterInteractionEnterErrorModeMustBeInFreeMarket        CharacterInteractionEnterErrorMode = "MUST_BE_IN_FREE_MARKET"        // 15
	CharacterInteractionEnterErrorModeMustBeInRoom722           CharacterInteractionEnterErrorMode = "MUST_BE_IN_ROOM_722"           // 16
	CharacterInteractionEnterErrorModeCannotEnterStore          CharacterInteractionEnterErrorMode = "CANNOT_ENTER_STORE"            // 17
	CharacterInteractionEnterErrorModeUndergoingMaintenance     CharacterInteractionEnterErrorMode = "UNDERGOING_MAINTENANCE"        // 18
	CharacterInteractionEnterErrorModeCannotEnterTournamentRoom CharacterInteractionEnterErrorMode = "CANNOT_ENTER_TOURNAMENT_ROOM"  // 19
	CharacterInteractionEnterErrorModeTradeNotAllowed2          CharacterInteractionEnterErrorMode = "TRADE_NOT_ALLOWED_2"           // 20
	CharacterInteractionEnterErrorModeNotEnoughMesos            CharacterInteractionEnterErrorMode = "NOT_ENOUGH_MESOS"              // 21
	CharacterInteractionEnterErrorModeIncorrectPassword         CharacterInteractionEnterErrorMode = "INCORRECT_PASSWORD"            // 22
	CharacterInteractionEnterErrorModeItemExpired               CharacterInteractionEnterErrorMode = "ITEM_EXPIRED"                  // 24

)

func CharacterInteractionInviteBody(roomType model.MiniRoomType, name string, dwSN uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterInteractionMode(l)(options, CharacterInteractionModeInvite))
			w.WriteByte(byte(roomType))
			w.WriteAsciiString(name)
			w.WriteInt(dwSN)
			return w.Bytes()
		}
	}
}

func CharacterInteractionInviteResultBody(result byte, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			// 1, 2, 3, 4
			w.WriteByte(getCharacterInteractionMode(l)(options, CharacterInteractionModeInviteResult))
			w.WriteByte(result)
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func CharacterInteractionEnterBody(visitor model.MiniRoomVisitor) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterInteractionMode(l)(options, CharacterInteractionModeEnter))
			w.WriteByteArray(visitor.Enter()(l, ctx)(options))
			return w.Bytes()
		}
	}
}

func CharacterInteractionEnterResultSuccessBody(characterId uint32, mr model.MiniRoom) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterInteractionMode(l)(options, CharacterInteractionModeEnterResult))
			w.WriteByteArray(mr.Enter(characterId)(l, ctx)(options))
			return w.Bytes()
		}
	}
}

func CharacterInteractionEnterResultErrorBody(errorError CharacterInteractionEnterErrorMode) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterInteractionMode(l)(options, CharacterInteractionModeEnterResult))
			w.WriteByte(0)
			w.WriteByte(getCharacterInteractionEnterErrorMode(l)(options, errorError))
			return w.Bytes()
		}
	}
}

func getCharacterInteractionMode(l logrus.FieldLogger) func(options map[string]interface{}, key CharacterInteractionMode) byte {
	return func(options map[string]interface{}, key CharacterInteractionMode) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		op, ok := codes[string(key)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}

func getCharacterInteractionEnterErrorMode(l logrus.FieldLogger) func(options map[string]interface{}, key CharacterInteractionEnterErrorMode) byte {
	return func(options map[string]interface{}, key CharacterInteractionEnterErrorMode) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["enterError"]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		op, ok := codes[string(key)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}
