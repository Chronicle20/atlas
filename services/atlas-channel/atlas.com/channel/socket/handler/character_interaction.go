package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

type CharacterInteractionMode string

const (
	CharacterInteractionHandle = "CharacterInteractionHandle"

	CharacterInteractionModeCreate                        CharacterInteractionMode = "CREATE"                             // 00 - 0
	CharacterInteractionModeInvite                        CharacterInteractionMode = "INVITE"                             // 02 - 2
	CharacterInteractionModeInviteDecline                 CharacterInteractionMode = "INVITE_DECLINE"                     // 03 - 3
	CharacterInteractionModeVisit                         CharacterInteractionMode = "VISIT"                              // 04 - 4
	CharacterInteractionModeChat                          CharacterInteractionMode = "CHAT"                               // 06 - 6
	CharacterInteractionModeExit                          CharacterInteractionMode = "EXIT"                               // 10 - A
	CharacterInteractionModeOpen                          CharacterInteractionMode = "OPEN"                               // 11 - B
	CharacterInteractionModeCashTradeOpen                 CharacterInteractionMode = "CASH_TRADE_OPEN"                    // 14 - E
	CharacterInteractionModeTradePutItem                  CharacterInteractionMode = "TRADE_PUT_ITEM"                     // 15 - F
	CharacterInteractionModeTradeAddMeso                  CharacterInteractionMode = "TRADE_ADD_MESO"                     // 16 - 10
	CharacterInteractionModeTradeConfirm                  CharacterInteractionMode = "TRADE_CONFIRM"                      // 17 - 11
	CharacterInteractionModeTransaction                   CharacterInteractionMode = "TRANSACTION"                        // 20 - 14
	CharacterInteractionModePersonalStorePutItem          CharacterInteractionMode = "PERSONAL_STORE_PUT_ITEM"            // 22 - 16
	CharacterInteractionModePersonalStoreBuy              CharacterInteractionMode = "PERSONAL_STORE_BUY"                 // 23 - 17
	CharacterInteractionModePersonalStoreRemoveItem       CharacterInteractionMode = "PERSONAL_STORE_REMOVE_ITEM"         // 27 - 1B
	CharacterInteractionModePersonalStoreAddToBlackList   CharacterInteractionMode = "PERSONAL_STORE_ADD_TO_BLACKLIST"    // 28 - 1C
	CharacterInteractionModePersonalStoreSetVisitor       CharacterInteractionMode = "PERSONAL_STORE_SET_VISITOR"         // 29 - 1D
	CharacterInteractionModePersonalStoreSetBlackList     CharacterInteractionMode = "PERSONAL_STORE_SET_BLACK_LIST"      // 30 - 1E
	CharacterInteractionModeFieldAddToBlackList           CharacterInteractionMode = "FIELD_ADD_TO_BLACK_LIST"            // 31 - 1F
	CharacterInteractionModeFieldRemoveFromBlackList      CharacterInteractionMode = "FIELD_REMOVE_FROM_BLACK_LIST"       // 32 - 20
	CharacterInteractionModeMerchantPutItem               CharacterInteractionMode = "MERCHANT_PUT_ITEM"                  // 33 - 21
	CharacterInteractionModeMerchantBuy                   CharacterInteractionMode = "MERCHANT_BUY"                       // 34 - 22
	CharacterInteractionModeMerchantRemoveItem            CharacterInteractionMode = "MERCHANT_REMOVE_ITEM"               // 38 - 26
	CharacterInteractionModeMerchantMaintenanceOff        CharacterInteractionMode = "MERCHANT_MERCHANT_OFF"              // 39 - 27
	CharacterInteractionModeMerchantOrganize              CharacterInteractionMode = "MERCHANT_ORGANIZE"                  // 40 - 28
	CharacterInteractionModeMerchantExit                  CharacterInteractionMode = "MERCHANT_EXIT"                      // 41 - 29
	CharacterInteractionModeMerchantWithdrawMeso          CharacterInteractionMode = "MERCHANT_WITHDRAW_MESO"             // 43 - 2B
	CharacterInteractionModeMerchantNameChange            CharacterInteractionMode = "MERCHANT_NAME_CHANGE"               // 45 - 2D
	CharacterInteractionModeMerchantViewVisitList         CharacterInteractionMode = "MERCHANT_VIEW_VISIT_LIST"           // 46 - 2E
	CharacterInteractionModeMerchantViewBlackList         CharacterInteractionMode = "MERCHANT_VIEW_BLACK_LIST"           // 47 - 2F
	CharacterInteractionModeMerchantAddToBlackList        CharacterInteractionMode = "MERCHANT_ADD_TO_BLACK_LIST"         // 48 - 30
	CharacterInteractionModeMerchantRemoveFromBlackList   CharacterInteractionMode = "MERCHANT_REMOVE_FROM_BLACK_LIST"    // 49 - 31
	CharacterInteractionModeMemoryGameAskTie              CharacterInteractionMode = "MEMORY_GAME_ASK_TIE"                // 50 - 32
	CharacterInteractionModeMemoryGameTieAnswer           CharacterInteractionMode = "MEMORY_GAME_TIE_ANSWER"             // 51 - 33
	CharacterInteractionModeMemoryGameForfeit             CharacterInteractionMode = "MEMORY_GAME_FORFEIT"                // 52 - 34
	CharacterInteractionModeMemoryGameAskRetreat          CharacterInteractionMode = "MEMORY_GAME_ASK_RETREAT"            // 54 - 36
	CharacterInteractionModeMemoryGameRetreatAnswer       CharacterInteractionMode = "MEMORY_GAME_RETREAT_ANSWER"         // 55 - 37
	CharacterInteractionModeMemoryGameExitAfterGame       CharacterInteractionMode = "MEMORY_GAME_EXIT_AFTER_GAME"        // 56 - 38
	CharacterInteractionModeMemoryGameCancelExitAfterGame CharacterInteractionMode = "MEMORY_GAME_CANCEL_EXIT_AFTER_GAME" // 57 - 39
	CharacterInteractionModeMemoryGameReady               CharacterInteractionMode = "MEMORY_GAME_READY"                  // 58 - 3A
	CharacterInteractionModeMemoryGameUnready             CharacterInteractionMode = "MEMORY_GAME_UNREADY"                // 59 - 3B
	CharacterInteractionModeMemoryGameExpel               CharacterInteractionMode = "MEMORY_GAME_EXPEL"                  // 60 - 3C
	CharacterInteractionModeMemoryGameStart               CharacterInteractionMode = "MEMORY_GAME_START"                  // 61 - 3D
	CharacterInteractionModeMemoryGameSkip                CharacterInteractionMode = "MEMORY_GAME_SKIP"                   // 63 - 3F
	CharacterInteractionModeMemoryGameMoveStone           CharacterInteractionMode = "MEMORY_GAME_MOVE_STONE"             // 64 - 40
	CharacterInteractionModeMemoryGameFlipCard            CharacterInteractionMode = "MEMORY_GAME_FIP_CARD"               // 68 - 44
)

func CharacterInteractionHandleFunc(l logrus.FieldLogger, _ context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		mode := r.ReadByte()
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeCreate) {
			// CMiniRoomBaseDlg::OnCheckSSN2Static
			// 1 - Omok, 2 - Match Card, 3 - Trade, 4 - Shop, 5 - Merchant, 6 Cash Shop?
			roomType := model.MiniRoomType(r.ReadByte())
			title := ""
			private := false
			if roomType == model.OmokMiniRoom || roomType == model.MatchCardMiniRoom {
				// CWvsContext::SendCreateMiniGameRequest
				title = r.ReadAsciiString()
				private = r.ReadBool()
				password := ""
				if private {
					password = r.ReadAsciiString()
				}
				nGameSpec := r.ReadByte()
				l.Debugf("Character [%d] has created a mini-room. roomType [%d], title [%s], private [%t], password [%s], nGameSpec [%d].", s.CharacterId(), roomType, title, private, password, nGameSpec)
				return
			}
			if roomType == model.TradeMiniRoom {
				// CField::SendInviteTradingRoomMsg
				private = r.ReadBool()
				l.Debugf("Character [%d] has created a trade-room. roomType [%d], title [%s], private [%t].", s.CharacterId(), roomType, title, private)
				return
			}
			if roomType == model.PersonalShopMiniRoom || roomType == model.MerchantShopMiniRoom {
				// CWvsContext::SendOpenShopRequest
				title = r.ReadAsciiString()
				private = r.ReadBool()
				slot := r.ReadInt16()
				itemId := r.ReadUint32()
				l.Debugf("Character [%d] has created a store. roomType [%d], title [%s], private [%t], position [%d], itemId [%d].", s.CharacterId(), roomType, title, private, slot, itemId)
				return
			}
			if roomType == model.CashTradeMiniRoom {
				// CMiniRoomBaseDlg::OnCheckSSN2Static
				private = r.ReadBool()
				l.Debugf("Character [%d] has created a store. roomType [%d], private [%t].", s.CharacterId(), roomType, private)
			}
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeInvite) {
			targetCharacterId := r.ReadUint32()
			l.Debugf("Character [%d] is sending character [%d] a trade invite.", s.CharacterId(), targetCharacterId)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeInviteDecline) {
			serialNumber := r.ReadUint32()
			errorCode := r.ReadByte() // 3 - birthday failed
			l.Debugf("Character [%d] is declining trade invite. serialNumber [%d], errorCode [%d].", s.CharacterId(), serialNumber, errorCode)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeVisit) {
			serialNumber := r.ReadUint32()
			errorCode := r.ReadByte() // should be 0
			errorMessage := ""
			if errorCode != 0 {
				errorMessage = r.ReadAsciiString()
			}
			something := r.ReadBool()
			unk1 := int16(0)
			cashSerialNumber := uint64(0)
			if something {
				unk1 = r.ReadInt16() // position?
				cashSerialNumber = r.ReadUint64()
			}
			l.Debugf("Character [%d] is accepting a trade invite. serialNumber [%d], errorCode [%d], errorMessage [%s], something [%t], unk1 [%d], cashSerialNumber [%d].", s.CharacterId(), serialNumber, errorCode, errorMessage, something, unk1, cashSerialNumber)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeChat) {
			message := r.ReadAsciiString()
			l.Debugf("Character [%d] is sending chat [%s].", s.CharacterId(), message)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeExit) {
			l.Debugf("Character [%d] has stopped character interaction.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeOpen) {
			success := r.ReadBool()
			l.Debugf("Character [%d] has opened (something). success [%t].", s.CharacterId(), success)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeCashTradeOpen) {
			// CMiniRoomBaseDlg::OnCheckSSN2Static
			nProc := r.ReadByte() // can be greater than 0
			roomType := model.MiniRoomType(r.ReadByte())
			if nProc == 0 && roomType == model.CashTradeMiniRoom {
				// CField::SendInviteTradingRoomMsg
				targetCharacterId := r.ReadUint32()
				l.Debugf("Character [%d] has opened cash trade. nProc [%d], roomType [%d], targetCharacterId [%d].", s.CharacterId(), nProc, roomType, targetCharacterId)
				return
			}
			if nProc == 4 && roomType == model.CashTradeMiniRoom {
				// CMiniRoomBaseDlg::SendCashInviteResult
				spw := r.ReadUint32()
				dwSN := r.ReadUint32()
				unk2 := r.ReadByte()
				l.Debugf("Character [%d] has opened cash trade. nProc [%d], roomType [%d], spw [%d], dwSN [%d], unk2 [%d].", s.CharacterId(), nProc, roomType, spw, dwSN, unk2)
				return
			}
			if nProc == 4 && roomType == model.MerchantShopMiniRoom {
				// CWvsContext::OnEntrustedShopCheckResult
				// TODO This immediately triggered from a hired_merchant_operation ConfirmManage
				spw := r.ReadUint32()
				shopId := r.ReadUint32()
				unk2 := r.ReadByte()
				position := r.ReadUint16()
				serialNumber := r.ReadUint64()
				l.Debugf("Character [%d] has opened cash trade. nProc [%d], roomType [%d], spw [%d], shopId [%d], unk2 [%d], position [%d], serialNumber [%d].", s.CharacterId(), nProc, roomType, spw, shopId, unk2, position, serialNumber)
				return
			}
			if nProc == 11 && (roomType == model.PersonalShopMiniRoom || roomType == model.MerchantShopMiniRoom) {
				// CPersonalShopDlg::CheckCashItemInList
				birthday := r.ReadUint32()
				l.Debugf("Character [%d] has opened cash trade. nProc [%d], roomType [%d], birthday [%d].", s.CharacterId(), nProc, roomType, birthday)
				return
			}
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeTradePutItem) {
			it := r.ReadByte()
			slot := r.ReadInt16()
			quantity := r.ReadUint16()
			targetSlot := r.ReadByte()
			l.Debugf("Character [%d] attempting to put [%d] item(s) from inventory compartment [%d] slot [%d] up for trade. target [%d].", s.CharacterId(), quantity, it, slot, targetSlot)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeTradeAddMeso) {
			amount := r.ReadInt32()
			l.Debugf("Character [%d] attempting to put [%d] meso up for trade.", s.CharacterId(), amount)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeTradeConfirm) {
			size := r.ReadByte()
			for range size {
				data := r.ReadUint32()
				crc := r.ReadUint32()
				l.Debugf("Character [%d] confirmed trade includes [%d]. crc [%d].", s.CharacterId(), data, crc)
			}
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeTransaction) {
			size := r.ReadByte()
			for range size {
				data := r.ReadUint32()
				crc := r.ReadUint32()
				l.Debugf("Character [%d] transaction includes [%d]. crc [%d].", s.CharacterId(), data, crc)
			}
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStorePutItem) {
			it := r.ReadByte()
			slot := r.ReadInt16()
			quantity := r.ReadUint16()
			set := r.ReadUint16()
			price := r.ReadUint32()
			l.Debugf("Character [%d] attempting to add [%d] item(s) from inventory compartment [%d] slot [%d] to store. set [%d], price [%d].", s.CharacterId(), quantity, it, slot, set, price)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStoreBuy) {
			index := r.ReadByte()
			quantity := r.ReadUint16()
			l.Debugf("Character [%d] attempting to purchase [%d] item(s) index [%d] from store.", s.CharacterId(), quantity, index)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStoreRemoveItem) {
			index := r.ReadUint16()
			l.Debugf("Character [%d] attempting to remove item index [%d] from store.", s.CharacterId(), index)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStoreAddToBlackList) {
			slot := r.ReadByte()
			name := r.ReadAsciiString()
			l.Debugf("Character [%d] is adding [%s] to field black list from slot [%d].", s.CharacterId(), name, slot)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStoreSetVisitor) {
			slot := r.ReadByte()
			name := r.ReadAsciiString()
			l.Debugf("Character [%d] has [%s] in their store at slot [%d]", s.CharacterId(), name, slot)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStoreSetBlackList) {
			names := make([]string, 0)
			size := r.ReadUint16()
			for range size {
				names = append(names, string(r.ReadByte()))
			}
			l.Debugf("Character [%d] has set store black list. size [%d]", s.CharacterId(), size)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeFieldAddToBlackList) {
			name := r.ReadAsciiString()
			l.Debugf("Character [%d] is adding [%s] to field black list.", s.CharacterId(), name)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeFieldRemoveFromBlackList) {
			name := r.ReadAsciiString()
			l.Debugf("Character [%d] is removing [%s] from field black list.", s.CharacterId(), name)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantPutItem) {
			it := r.ReadByte()
			slot := r.ReadInt16()
			quantity := r.ReadUint16()
			set := r.ReadUint16()
			price := r.ReadUint32()
			l.Debugf("Character [%d] attempting to add [%d] item(s) from inventory compartment [%d] slot [%d] to merchant. set [%d], price [%d].", s.CharacterId(), quantity, it, slot, set, price)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantBuy) {
			index := r.ReadByte()
			quantity := r.ReadUint16()
			l.Debugf("Character [%d] attempting to purchase [%d] item(s) index [%d] from merchant.", s.CharacterId(), quantity, index)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantRemoveItem) {
			index := r.ReadUint16()
			l.Debugf("Character [%d] attempting to remove item index [%d] from merchant.", s.CharacterId(), index)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantMaintenanceOff) {
			l.Debugf("Character [%d] has stopped merchant maintenance.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantOrganize) {
			l.Debugf("Character [%d] has organized merchant inventory.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantExit) {
			l.Debugf("Character [%d] has stopped merchant interaction.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantWithdrawMeso) {
			l.Debugf("Character [%d] has withdrew merchant meso.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantNameChange) {
			unk1 := r.ReadUint32()
			l.Debugf("Character [%d] wants to change their merchant shop name. unk1 [%d].", s.CharacterId(), unk1)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantViewVisitList) {
			l.Debugf("Character [%d] has viewed merchant visit list.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantViewBlackList) {
			l.Debugf("Character [%d] has viewed merchant black list.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantAddToBlackList) {
			name := r.ReadAsciiString()
			l.Debugf("Character [%d] is adding [%s] to merchant black list.", s.CharacterId(), name)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantRemoveFromBlackList) {
			name := r.ReadAsciiString()
			l.Debugf("Character [%d] is removing [%s] from merchant black list.", s.CharacterId(), name)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameAskTie) {
			l.Debugf("Character [%d] in memory game, is asking for a tie.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameTieAnswer) {
			response := r.ReadBool()
			l.Debugf("Character [%d] in memory game, is answering tie request. response [%t].", s.CharacterId(), response)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameForfeit) {
			l.Debugf("Character [%d] in memory game, is forfeiting.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameAskRetreat) {
			l.Debugf("Character [%d] in memory game, is asking to retreat.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameRetreatAnswer) {
			response := r.ReadBool()
			l.Debugf("Character [%d] in memory game, is answering retreat request. response [%t].", s.CharacterId(), response)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameExitAfterGame) {
			l.Debugf("Character [%d] in memory game, wants to exit after it is completed.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameCancelExitAfterGame) {
			l.Debugf("Character [%d] in memory game, does not want to exit after it is completed.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameReady) {
			l.Debugf("Character [%d] is ready for a memory game.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameUnready) {
			l.Debugf("Character [%d] is not ready for a memory game.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameExpel) {
			l.Debugf("Character [%d] has expelled visitor from the memory game.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameStart) {
			l.Debugf("Character [%d] is starting a memory game.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameSkip) {
			l.Debugf("Character [%d] in memory game, is being skipped.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameMoveStone) {
			point := r.ReadInt64()
			color := r.ReadByte()
			l.Debugf("Character [%d] in memory game, is moving stone. point [%d], color [%d].", s.CharacterId(), point, color)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameFlipCard) {
			first := r.ReadBool()
			index := r.ReadByte()
			l.Debugf("Character [%d] in memory game, is flipping card [%d]. first [%t].", s.CharacterId(), index, first)
			return
		}
		l.Warnf("Character [%d] issued a unhandled character interaction [%d].", s.CharacterId(), mode)
	}
}

func isCharacterInteraction(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key CharacterInteractionMode) bool {
	return func(options map[string]interface{}, op byte, key CharacterInteractionMode) bool {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		res, ok := codes[string(key)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}
		return byte(res) == op
	}
}
