package writer

import (
	"atlas-channel/socket/model"
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	interactionpkt "github.com/Chronicle20/atlas-packet/interaction"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type CharacterInteractionMode string

type CharacterInteractionEnterErrorMode string

const (
	// CharacterInteraction CMiniRoomBaseDlg::OnPacketBase
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
		return func(options map[string]interface{}) []byte {
			mode := getCharacterInteractionMode(l)(options, CharacterInteractionModeInvite)
			return interactionpkt.NewInteractionInvite(mode, byte(roomType), name, dwSN).Encode(l, ctx)(options)
		}
	}
}

func CharacterInteractionInviteResultBody(result byte, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			// 1, 2, 3, 4
			mode := getCharacterInteractionMode(l)(options, CharacterInteractionModeInviteResult)
			return interactionpkt.NewInteractionInviteResult(mode, result, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterInteractionEnterBody(visitor model.MiniRoomVisitor) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterInteractionMode(l)(options, CharacterInteractionModeEnter)
			v := visitor.ToPacketVisitor(l, ctx, options)
			return interactionpkt.NewInteractionEnter(mode, v).Encode(l, ctx)(options)
		}
	}
}

func CharacterInteractionEnterResultSuccessBody(characterId uint32, mr model.MiniRoom) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterInteractionMode(l)(options, CharacterInteractionModeEnterResult)
			room := mr.ToPacketRoom(l, ctx, options, characterId)
			return interactionpkt.NewInteractionEnterResultSuccess(mode, room).Encode(l, ctx)(options)
		}
	}
}

func CharacterInteractionEnterResultErrorBody(errorError CharacterInteractionEnterErrorMode) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterInteractionMode(l)(options, CharacterInteractionModeEnterResult)
			errorCode := getCharacterInteractionEnterErrorMode(l)(options, errorError)
			return interactionpkt.NewInteractionEnterResultError(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func getCharacterInteractionMode(l logrus.FieldLogger) func(options map[string]interface{}, key CharacterInteractionMode) byte {
	return func(options map[string]interface{}, key CharacterInteractionMode) byte {
		return atlas_packet.ResolveCode(l, options, "operations", string(key))
	}
}

func getCharacterInteractionEnterErrorMode(l logrus.FieldLogger) func(options map[string]interface{}, key CharacterInteractionEnterErrorMode) byte {
	return func(options map[string]interface{}, key CharacterInteractionEnterErrorMode) byte {
		return atlas_packet.ResolveCode(l, options, "enterError", string(key))
	}
}
