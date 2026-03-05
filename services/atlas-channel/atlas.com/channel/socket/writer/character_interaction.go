package writer

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
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

func CharacterInteractionInviteBody(l logrus.FieldLogger) func(roomType model.MiniRoomType, name string, dwSN uint32) BodyProducer {
	return func(roomType model.MiniRoomType, name string, dwSN uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterInteractionMode(l)(options, CharacterInteractionModeInvite))
			w.WriteByte(byte(roomType))
			w.WriteAsciiString(name)
			w.WriteInt(dwSN)
			return w.Bytes()
		}
	}
}

func CharacterInteractionInviteResultBody(l logrus.FieldLogger) func(result byte, message string) BodyProducer {
	return func(result byte, message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// 1, 2, 3, 4
			w.WriteByte(getCharacterInteractionMode(l)(options, CharacterInteractionModeInviteResult))
			w.WriteByte(result)
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func CharacterInteractionEnterBody(l logrus.FieldLogger, ctx context.Context) func(roomType model.MiniRoomType, slot byte, c character.Model) BodyProducer {
	t := tenant.MustFromContext(ctx)
	return func(roomType model.MiniRoomType, slot byte, c character.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterInteractionMode(l)(options, CharacterInteractionModeEnter))
			w.WriteByte(slot)
			WriteCharacterLook(t)(w, c, false)
			w.WriteAsciiString(c.Name())
			return w.Bytes()
		}
	}
}

func CharacterInteractionEnterMiniGameBody(l logrus.FieldLogger, ctx context.Context) func(roomType model.MiniRoomType, slot byte, c character.Model, mgr model.MiniGameRecord) BodyProducer {
	t := tenant.MustFromContext(ctx)
	return func(roomType model.MiniRoomType, slot byte, c character.Model, mgr model.MiniGameRecord) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterInteractionMode(l)(options, CharacterInteractionModeEnter))
			w.WriteByte(slot)
			WriteCharacterLook(t)(w, c, false)
			w.WriteAsciiString(c.Name())
			w.WriteShort(uint16(c.JobId()))
			mgr.Encode(l, t, options)(w)
			return w.Bytes()
		}
	}
}

func CharacterInteractionEnterResultSuccessBody(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, mr model.MiniRoom) BodyProducer {
	t := tenant.MustFromContext(ctx)
	return func(characterId uint32, mr model.MiniRoom) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterInteractionMode(l)(options, CharacterInteractionModeEnterResult))
			w.WriteByte(byte(mr.Type))
			w.WriteByte(mr.MaxUsers)
			if mr.IsMerchant() {
				w.WriteByte(0)
				w.WriteInt(uint32(mr.ItemId))
				w.WriteAsciiString("Hired Merchant") // TODO
			}
			for _, v := range mr.Visitors {
				w.WriteByte(v.Slot)
				WriteCharacterLook(t)(w, v.Character, false)
				w.WriteAsciiString(v.Character.Name())
			}
			w.WriteByte(-1)
			if mr.Type == model.OmokMiniRoom || mr.Type == model.MatchCardMiniRoom {
				// write omok standings
				nGameKind := byte(0)
				bTournament := false
				for _, v := range mr.Visitors {
					mgr := model.MiniGameRecord{}
					w.WriteByte(v.Slot)
					mgr.Encode(l, t, options)(w)
				}
				w.WriteByte(-1)
				w.WriteAsciiString("description")
				w.WriteByte(nGameKind)
				w.WriteBool(bTournament)
				if bTournament {
					nRound := byte(0)
					w.WriteByte(nRound)
				}
				return w.Bytes()
			}
			if mr.Type == model.PersonalShopMiniRoom {
				w.WriteAsciiString("description")
				w.WriteByte(16) // max item count
				w.WriteByte(0)  // item count
				for range 0 {
					w.WriteShort(0) // per bundle
					w.WriteShort(0) // quantity
					w.WriteInt(0)   // price
					_ = WriteAssetInfo(t)(true)(w)(asset.Model{})
				}
				return w.Bytes()
			}
			if mr.Type == model.MerchantShopMiniRoom {
				// only written to owner
				messages := uint16(0)
				w.WriteShort(messages)
				for range messages {
					w.WriteAsciiString("message")
					fromSlot := byte(0)
					w.WriteByte(fromSlot)
				}
				w.WriteAsciiString("owner name")
				w.WriteByte(16) // max item count
				w.WriteInt(0)   // characters meso
				w.WriteByte(0)  // item count
				for range 0 {
					w.WriteShort(0) // per bundle
					w.WriteShort(0) // quantity
					w.WriteInt(0)   // price
					_ = WriteAssetInfo(t)(true)(w)(asset.Model{})
				}
				return w.Bytes()
			}

			return w.Bytes()
		}
	}
}

func CharacterInteractionEnterResultErrorBody(l logrus.FieldLogger) func(errorError CharacterInteractionEnterErrorMode) BodyProducer {
	return func(errorError CharacterInteractionEnterErrorMode) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
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
