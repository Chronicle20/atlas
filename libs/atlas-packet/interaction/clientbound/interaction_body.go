package clientbound

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type CharacterInteractionMode = string

type CharacterInteractionEnterErrorMode = string

const (
	// CharacterInteraction CMiniRoomBaseDlg::OnPacketBase
	CharacterInteractionModeInvite          CharacterInteractionMode = "INVITE"           // 2
	CharacterInteractionModeInviteResult    CharacterInteractionMode = "INVITE_RESULT"    // 3
	CharacterInteractionModeEnter           CharacterInteractionMode = "ENTER"            // 4
	CharacterInteractionModeEnterResult     CharacterInteractionMode = "ENTER_RESULT"     // 5
	CharacterInteractionModeChat            CharacterInteractionMode = "CHAT"             // 6
	CharacterInteractionModeChatThing       CharacterInteractionMode = "CHAT_THING"       // 8
	CharacterInteractionModeLeave           CharacterInteractionMode = "LEAVE"            // 10
	CharacterInteractionModeUpdateMerchant  CharacterInteractionMode = "UPDATE_MERCHANT"  // 25

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

func CharacterInteractionInviteBody(roomType byte, name string, dwSN uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeInvite, func(mode byte) packet.Encoder {
		return NewInteractionInvite(mode, roomType, name, dwSN)
	})
}

func CharacterInteractionInviteResultBody(result byte, message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeInviteResult, func(mode byte) packet.Encoder {
		return NewInteractionInviteResult(mode, result, message)
	})
}

func CharacterInteractionChatBody(slot byte, message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CharacterInteractionModeChat)
			chatType := atlas_packet.ResolveCode(l, options, "operations", CharacterInteractionModeChatThing)
			return NewInteractionChat(mode, chatType, slot, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterInteractionEnterResultErrorBody(errorError CharacterInteractionEnterErrorMode) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CharacterInteractionModeEnterResult)
			errorCode := atlas_packet.ResolveCode(l, options, "enterError", errorError)
			return NewInteractionEnterResultError(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CharacterInteractionEnterResultSuccessBody(room interaction.Room) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeEnterResult, func(mode byte) packet.Encoder {
		return NewInteractionEnterResultSuccess(mode, room)
	})
}

func CharacterInteractionEnterBody(visitor interaction.Visitor) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeEnter, func(mode byte) packet.Encoder {
		return NewInteractionEnter(mode, visitor)
	})
}

// CharacterInteractionLeaveBody notifies a client that a visitor has left the room.
// TODO: verify status byte values with client testing.
func CharacterInteractionLeaveBody(slot byte, status byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeLeave, func(mode byte) packet.Encoder {
		return NewInteractionLeave(mode, slot, status)
	})
}

func CharacterInteractionUpdateMerchantBody(meso uint32, items []interaction.RoomShopItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeUpdateMerchant, func(mode byte) packet.Encoder {
		return NewInteractionUpdateMerchant(mode, meso, items)
	})
}
