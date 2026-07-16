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
	CharacterInteractionModeInvite         CharacterInteractionMode = "INVITE"          // 2
	CharacterInteractionModeInviteResult   CharacterInteractionMode = "INVITE_RESULT"   // 3
	CharacterInteractionModeEnter          CharacterInteractionMode = "ENTER"           // 4
	CharacterInteractionModeEnterResult    CharacterInteractionMode = "ENTER_RESULT"    // 5
	CharacterInteractionModeChat           CharacterInteractionMode = "CHAT"            // 6
	CharacterInteractionModeChatThing      CharacterInteractionMode = "CHAT_THING"      // 8
	CharacterInteractionModeLeave          CharacterInteractionMode = "LEAVE"           // 10
	CharacterInteractionModeUpdateMerchant CharacterInteractionMode = "UPDATE_MERCHANT" // 25
	// CharacterInteractionModePersonalStoreItemSold is the per-sale sold-item
	// notification to the owner (CPersonalShopDlg::OnSoldItemResult). Its byte is
	// UPDATE_MERCHANT+1 in every version (v48 23, v61/72 24, v79 25, v83+ 26).
	CharacterInteractionModePersonalStoreItemSold CharacterInteractionMode = "PERSONAL_STORE_ITEM_SOLD"
	// The hired-merchant view responses echo the request mode byte: the client
	// decodes them under the same operation constants it sends
	// (CEntrustedShopDlg::OnPacket sub_51870D cases 0x2E/0x2F).
	CharacterInteractionModeMerchantViewVisitList CharacterInteractionMode = "MERCHANT_VIEW_VISIT_LIST" // 46
	CharacterInteractionModeMerchantViewBlackList CharacterInteractionMode = "MERCHANT_VIEW_BLACK_LIST" // 47

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

	// CharacterInteraction mini-game modes (CMiniRoomBaseDlg::OnPacketBase — Omok /
	// Match Cards, one enum shared by serverbound and clientbound). Verified
	// byte-identical on gms_v83 and gms_v95: docs/tasks/task-133-miniroom-minigames/ida-notes.md §G5.
	CharacterInteractionModeMemoryGameAskTie        CharacterInteractionMode = "MEMORY_GAME_ASK_TIE"        // 50
	CharacterInteractionModeMemoryGameTieAnswer     CharacterInteractionMode = "MEMORY_GAME_TIE_ANSWER"     // 51
	CharacterInteractionModeMemoryGameAskRetreat    CharacterInteractionMode = "MEMORY_GAME_ASK_RETREAT"    // 54
	CharacterInteractionModeMemoryGameRetreatAnswer CharacterInteractionMode = "MEMORY_GAME_RETREAT_ANSWER" // 55
	CharacterInteractionModeMemoryGameReady         CharacterInteractionMode = "MEMORY_GAME_READY"          // 58
	CharacterInteractionModeMemoryGameUnready       CharacterInteractionMode = "MEMORY_GAME_UNREADY"        // 59
	CharacterInteractionModeMemoryGameStart         CharacterInteractionMode = "MEMORY_GAME_START"          // 61
	CharacterInteractionModeMemoryGameResult        CharacterInteractionMode = "MEMORY_GAME_RESULT"         // 62
	CharacterInteractionModeMemoryGameSkip          CharacterInteractionMode = "MEMORY_GAME_SKIP"           // 63
	CharacterInteractionModeMemoryGameMoveStone     CharacterInteractionMode = "MEMORY_GAME_MOVE_STONE"     // 64
	CharacterInteractionModeMemoryGameFlipCard      CharacterInteractionMode = "MEMORY_GAME_FIP_CARD"       // 68 (typo is load-bearing)
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
// CMiniRoomBaseDlg::OnLeaveBase (v95 0x637510) reads only Decode1(slot); the trailing
// status byte is consumed by the subclass virtual OnLeave (e.g. CTradingRoomDlg::OnLeave),
// so the full mode-10 wire shape is mode + slot + status. status 0 = silent close
// (correct for a voluntary self-exit); use CharacterInteractionLeaveReasonBody to send
// a reason that shows a message.
func CharacterInteractionLeaveBody(slot byte, status byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeLeave, func(mode byte) packet.Encoder {
		return NewInteractionLeave(mode, slot, status)
	})
}

// Leave-reason keys resolved via the "leaveReason" tenant writer table. The
// status byte is client-interpreted (CPersonalShopDlg::OnLeave switch, v95
// @0x699c40): values not in the switch — including 0 — render an empty Notice
// dialog, so an ejected visitor must receive a mapped reason (DOM-25).
const (
	CharacterInteractionLeaveReasonShopClosed = "SHOP_CLOSED"  // 3  "The shop is closed."
	CharacterInteractionLeaveReasonUserBanned = "USER_BANNED"  // 5  "The user has been banned."
	CharacterInteractionLeaveReasonOutOfStock = "OUT_OF_STOCK" // 14 "The items are out of stock."
)

// CharacterInteractionLeaveReasonBody sends a LEAVE whose status byte is resolved
// from the tenant "leaveReason" table, so the ejected visitor sees the correct
// message instead of an empty dialog. reason is one of the
// CharacterInteractionLeaveReason* keys.
func CharacterInteractionLeaveReasonBody(slot byte, reason string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CharacterInteractionModeLeave)
			status := atlas_packet.ResolveCode(l, options, "leaveReason", reason)
			return NewInteractionLeave(mode, slot, status).Encode(l, ctx)(options)
		}
	}
}

func CharacterInteractionUpdateMerchantBody(meso uint32, items []interaction.RoomShopItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeUpdateMerchant, func(mode byte) packet.Encoder {
		return NewInteractionUpdateMerchant(mode, meso, items)
	})
}

// CharacterInteractionMiniGameRoomBody is the EnterResult SUCCESS body for an
// Omok / Match Cards room (the game analogue of
// CharacterInteractionEnterResultSuccessBody; same ENTER_RESULT mode key,
// discrete game-shaped struct — see InteractionMiniGameRoom for the
// IDA-derived two-list layout). yourSlot is the recipient's slot (0 owner /
// 1 visitor); gameKind is the piece/sub-type byte (Cosmic `piece`).
func CharacterInteractionMiniGameRoomBody(roomType interaction.RoomType, capacity byte, yourSlot byte, players []MiniGameRoomPlayer, title string, gameKind byte, tournament bool, round byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeEnterResult, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameRoom(mode, roomType, capacity, yourSlot, players, title, gameKind, tournament, round)
	})
}

// CharacterInteractionMiniGameEnterBody notifies the room owner that a visitor
// joined a game room (the game analogue of CharacterInteractionEnterBody; same
// ENTER mode key, discrete game-shaped struct carrying the trailing 20-byte
// record — see InteractionMiniGameEnter).
func CharacterInteractionMiniGameEnterBody(player MiniGameRoomPlayer) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeEnter, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameEnter(mode, player)
	})
}

func CharacterInteractionMiniGameReadyBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameReady, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameReady(mode)
	})
}

func CharacterInteractionMiniGameUnreadyBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameUnready, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameUnready(mode)
	})
}

func CharacterInteractionMiniGameRequestTieBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameAskTie, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameRequestTie(mode)
	})
}

// CharacterInteractionMiniGameAnswerTieBody covers the deny path only — the accept
// path emits RESULT (mode 62) instead, per the brief and ida-notes.md §G5 RESULT.
func CharacterInteractionMiniGameAnswerTieBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameTieAnswer, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameAnswerTie(mode)
	})
}

// CharacterInteractionMiniGameRetreatRequestBody sends the bodyless
// ASK_RETREAT mode — ida-notes.md §G2.
func CharacterInteractionMiniGameRetreatRequestBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameAskRetreat, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameRetreatRequest(mode)
	})
}

// CharacterInteractionMiniGameRetreatAnswerBody: accept selects the shape —
// on decline (accept == false) stoneCount/turnSlot are ignored and not
// serialized; on accept they are the N stones the client pops from the tail
// of the move history and the slot whose turn follows, per ida-notes.md §G2
// (no Cosmic reference; the sole layout authority, verified gms_v83/gms_v95).
func CharacterInteractionMiniGameRetreatAnswerBody(accept bool, stoneCount byte, turnSlot byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameRetreatAnswer, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameRetreatAnswer(mode, accept, stoneCount, turnSlot)
	})
}

func CharacterInteractionMiniGameSkipBody(who byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameSkip, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameSkip(mode, who)
	})
}

// CharacterInteractionMiniGameStartOmokBody and
// CharacterInteractionMiniGameStartMatchCardsBody are the two discrete START
// arms (mode 61, ida-notes.md §G1/§G5); both resolve the same mode key.
func CharacterInteractionMiniGameStartOmokBody(firstMover byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameStart, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameStartOmok(mode, firstMover)
	})
}

func CharacterInteractionMiniGameStartMatchCardsBody(firstMover byte, deck []uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameStart, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameStartMatchCards(mode, firstMover, deck)
	})
}

func CharacterInteractionMiniGameMoveStoneBody(x uint32, y uint32, stoneType byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameMoveStone, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameMoveStone(mode, x, y, stoneType)
	})
}

// CharacterInteractionMiniGameCardSelectFirstBody and
// CharacterInteractionMiniGameCardSelectSecondBody are the two discrete
// SELECT_CARD/FLIP_CARD arms (mode 68, ida-notes.md §G5); both resolve the
// same mode key.
func CharacterInteractionMiniGameCardSelectFirstBody(slot byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameFlipCard, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameCardSelectFirst(mode, slot)
	})
}

func CharacterInteractionMiniGameCardSelectSecondBody(slot byte, firstSlot byte, resultType byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameFlipCard, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameCardSelectSecond(mode, slot, firstSlot, resultType)
	})
}

// CharacterInteractionMiniGameResultBody: resultType 0 = normal win, 1 = tie,
// 2 = forfeit win. visitorWon is only meaningful (and only serialized) for
// resultType != 1 — see InteractionMiniGameResult / ida-notes.md §G5 RESULT.
func CharacterInteractionMiniGameResultBody(resultType byte, visitorWon bool, ownerRecord interaction.GameRecord, visitorRecord interaction.GameRecord) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameResult, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameResult(mode, resultType, visitorWon, ownerRecord, visitorRecord)
	})
}

// CharacterInteractionUpdatePersonalShopBody is the mode-25 refresh for a
// personal shop (item 514): same shape as the merchant refresh but WITHOUT the
// leading meso field, which only the hired-merchant client reads.
func CharacterInteractionUpdatePersonalShopBody(items []interaction.RoomShopItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeUpdateMerchant, func(mode byte) packet.Encoder {
		return NewInteractionUpdatePersonalShop(mode, items)
	})
}

// CharacterInteractionPersonalStoreItemSoldBody sends the owner the sold-item
// notification for one sale (item index in the shop's listing display, bundles
// purchased, and the buyer's name). The client appends it to the sold ledger
// and advances the running totals.
func CharacterInteractionPersonalStoreItemSoldBody(itemIndex byte, bundleCount uint16, buyerName string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModePersonalStoreItemSold, func(mode byte) packet.Encoder {
		return NewInteractionPersonalShopItemSold(mode, itemIndex, bundleCount, buyerName)
	})
}

func CharacterInteractionVisitListBody(entries []InteractionVisitListEntry) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMerchantViewVisitList, func(mode byte) packet.Encoder {
		conv := make([]VisitListEntry, 0, len(entries))
		for _, e := range entries {
			conv = append(conv, VisitListEntry{Name: e.Name, Count: e.Count})
		}
		return NewInteractionVisitList(mode, conv)
	})
}

// InteractionVisitListEntry is the caller-facing entry shape (avoids exporting
// the codec's internal type through the body signature).
type InteractionVisitListEntry struct {
	Name  string
	Count uint32
}

func CharacterInteractionBlackListBody(names []string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMerchantViewBlackList, func(mode byte) packet.Encoder {
		return NewInteractionBlackList(mode, names)
	})
}
