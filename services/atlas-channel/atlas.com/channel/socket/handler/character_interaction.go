package handler

import (
	"atlas-channel/character"
	"atlas-channel/merchant"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	interactioncb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound"
	interaction2 "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

type CharacterInteractionMode string

const (
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

func CharacterInteractionHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := interaction2.Operation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		mode := p.Mode()
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeCreate) {
			sp := &interaction2.OperationCreate{}
			sp.Decode(l, ctx)(r, readerOptions)
			roomType := model.MiniRoomType(sp.RoomType())
			if roomType == model.OmokMiniRoomType || roomType == model.MatchCardMiniRoomType {
				l.Debugf("Character [%d] has created a mini-room. roomType [%d], title [%s], private [%t], password [%s], nGameSpec [%d].", s.CharacterId(), roomType, sp.Title(), sp.Private(), sp.Password(), sp.NGameSpec())
				return
			}
			if roomType == model.TradeMiniRoomType {
				l.Debugf("Character [%d] has created a trade-room. roomType [%d], title [%s], private [%t].", s.CharacterId(), roomType, sp.Title(), sp.Private())
				return
			}
			if roomType == model.PersonalShopMiniRoomType || roomType == model.MerchantShopMiniRoomType {
				l.Debugf("Character [%d] has created a store. roomType [%d], title [%s], private [%t], position [%d], itemId [%d].", s.CharacterId(), roomType, sp.Title(), sp.Private(), sp.Slot(), sp.ItemId())
				rejectCreate := func(reason string) {
					l.Warnf("Character [%d] store create rejected: %s. roomType [%d], itemId [%d].", s.CharacterId(), reason, roomType, sp.ItemId())
					_ = session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionEnterResultErrorBody(interactioncb.CharacterInteractionEnterErrorModeUnable))(s)
				}

				// Permit validation: the claimed permit must be the right family for
				// the requested store kind (514 <-> personal store, 503 <-> hired
				// merchant) and actually present in the character's CASH inventory;
				// reject with mini-room error 6 otherwise. Permits are durable —
				// never consumed (owner decision, lifecycle audit Q1).
				permitClass := item.GetClassification(item.Id(sp.ItemId()))
				if roomType == model.PersonalShopMiniRoomType && permitClass != item.ClassificationStorePermit {
					rejectCreate("permit item is not a store permit (514 family)")
					return
				}
				if roomType == model.MerchantShopMiniRoomType && permitClass != item.ClassificationHiredMerchant {
					rejectCreate("permit item is not a hired-merchant permit (503 family)")
					return
				}

				cp := character.NewProcessor(l, ctx)
				c, err := cp.GetById(cp.InventoryDecorator)(s.CharacterId())
				if err != nil {
					l.WithError(err).Errorf("Unable to get character [%d] for shop placement.", s.CharacterId())
					return
				}
				if _, ok := c.Inventory().Cash().FindFirstByItemId(sp.ItemId()); !ok {
					rejectCreate("permit item not present in cash inventory")
					return
				}

				mp := merchant.NewProcessor(l, ctx)
				shopType := byte(1) // CharacterShop
				if roomType == model.MerchantShopMiniRoomType {
					shopType = byte(2) // HiredMerchant
				}
				_ = mp.PlaceShop(s.Field(), s.CharacterId(), shopType, sp.Title(), sp.ItemId(), c.X(), c.Y())
				return
			}
			if roomType == model.CashTradeMiniRoomType {
				l.Debugf("Character [%d] has created a store. roomType [%d], private [%t].", s.CharacterId(), roomType, sp.Private())
			}
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeInvite) {
			sp := &interaction2.OperationInvite{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] is sending character [%d] a trade invite.", s.CharacterId(), sp.TargetCharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeInviteDecline) {
			sp := &interaction2.OperationInviteDecline{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] is declining trade invite. serialNumber [%d], errorCode [%d].", s.CharacterId(), sp.SerialNumber(), sp.ErrorCode())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeVisit) {
			sp := &interaction2.OperationVisit{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] is visiting. serialNumber [%d], errorCode [%d], errorMessage [%s], something [%t], unk1 [%d], cashSerialNumber [%d].", s.CharacterId(), sp.SerialNumber(), sp.ErrorCode(), sp.ErrorMessage(), sp.Something(), sp.Unk1(), sp.CashSerialNumber())
			ownerCharacterId := sp.SerialNumber()
			mp := merchant.NewProcessor(l, ctx)
			shops, err := mp.GetByCharacterId(ownerCharacterId)
			if err != nil || len(shops) == 0 {
				l.WithError(err).Errorf("Unable to find shop for owner [%d].", ownerCharacterId)
				return
			}
			// The owner may have historical Closed rows (and up to one shop of
			// each type); visit the one that is actually enterable. Maintenance
			// is forwarded so the server replies with the faithful rejection.
			target, ok := pickShopByState(shops, merchant.StateOpen)
			if !ok {
				target, ok = pickShopByState(shops, merchant.StateMaintenance)
			}
			if !ok {
				l.Debugf("Character [%d] attempted to visit owner [%d] with no enterable shop.", s.CharacterId(), ownerCharacterId)
				return
			}
			visitorName := ""
			if vc, verr := character.NewProcessor(l, ctx).GetById()(s.CharacterId()); verr == nil {
				visitorName = vc.Name()
			}
			_ = mp.EnterShop(s.CharacterId(), target.Id(), visitorName)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeChat) {
			sp := &interaction2.OperationChat{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] is sending chat [%s].", s.CharacterId(), sp.Message())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.SendMessage(s.CharacterId(), visiting.Id(), sp.Message())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeExit) {
			l.Debugf("Character [%d] has stopped character interaction.", s.CharacterId())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Debugf("Character [%d] not visiting a shop, ignoring exit.", s.CharacterId())
				return
			}
			if visiting.CharacterId() == s.CharacterId() {
				// Owner closing the window: a hired merchant in Maintenance goes back
				// to running — a stocked merchant outlives the owner's management
				// view, while ExitMaintenance auto-closes an empty one. A personal
				// shop, or a merchant still in Draft setup, closes.
				if visiting.ShopType() == merchant.HiredMerchantShopType && visiting.State() == merchant.StateMaintenance {
					_ = mp.ExitMaintenance(s.CharacterId(), visiting.Id())
				} else {
					_ = mp.CloseShop(s.CharacterId(), visiting.Id())
				}
			} else {
				_ = mp.ExitShop(s.CharacterId(), visiting.Id())
			}
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeOpen) {
			sp := &interaction2.OperationOpen{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] has opened (something). success [%t].", s.CharacterId(), sp.Success())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.OpenShop(s.CharacterId(), visiting.Id())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeCashTradeOpen) {
			sp := &interaction2.OperationCashTradeOpen{}
			sp.Decode(l, ctx)(r, readerOptions)
			nProc := sp.NProc()
			roomType := model.MiniRoomType(sp.RoomType())
			if nProc == 0 && roomType == model.CashTradeMiniRoomType {
				l.Debugf("Character [%d] has opened cash trade. nProc [%d], roomType [%d], targetCharacterId [%d].", s.CharacterId(), nProc, roomType, sp.TargetCharacterId())
				return
			}
			if nProc == 4 && roomType == model.CashTradeMiniRoomType {
				l.Debugf("Character [%d] has opened cash trade. nProc [%d], roomType [%d], spw [%d], dwSN [%d], unk2 [%d].", s.CharacterId(), nProc, roomType, sp.Spw(), sp.DwSN(), sp.Unk2())
				return
			}
			if nProc == 4 && roomType == model.MerchantShopMiniRoomType {
				l.Debugf("Character [%d] entering merchant maintenance. nProc [%d], roomType [%d], spw [%d], shopId [%d], unk2 [%d], position [%d], serialNumber [%d].", s.CharacterId(), nProc, roomType, sp.Spw(), sp.ShopId(), sp.Unk2(), sp.Position(), sp.SerialNumber())
				mp := merchant.NewProcessor(l, ctx)
				shops, err := mp.GetByCharacterId(s.CharacterId())
				if err != nil || len(shops) == 0 {
					l.WithError(err).Errorf("Unable to find shop for character [%d].", s.CharacterId())
					return
				}
				// Maintenance entry only applies to the character's RUNNING hired
				// merchant — not a Closed history row or their personal shop.
				target, ok := pickMerchantByState(shops, merchant.StateOpen)
				if !ok {
					l.Debugf("Character [%d] requested merchant maintenance with no open hired merchant.", s.CharacterId())
					return
				}
				_ = mp.EnterMaintenance(s.CharacterId(), target.Id())
				return
			}
			if nProc == 11 && (roomType == model.PersonalShopMiniRoomType || roomType == model.MerchantShopMiniRoomType) {
				l.Debugf("Character [%d] has opened cash trade. nProc [%d], roomType [%d], birthday [%d].", s.CharacterId(), nProc, roomType, sp.Birthday())
				return
			}
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeTradePutItem) {
			sp := &interaction2.OperationTradePutItem{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] attempting to put [%d] item(s) from inventory compartment [%d] slot [%d] up for trade. target [%d].", s.CharacterId(), sp.Quantity(), sp.InventoryType(), sp.Slot(), sp.TargetSlot())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeTradeAddMeso) {
			sp := &interaction2.OperationTradeAddMeso{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] attempting to put [%d] meso up for trade.", s.CharacterId(), sp.Amount())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeTradeConfirm) {
			sp := &interaction2.OperationTradeConfirm{}
			sp.Decode(l, ctx)(r, readerOptions)
			for _, e := range sp.Entries() {
				l.Debugf("Character [%d] confirmed trade includes [%d]. crc [%d].", s.CharacterId(), e.Data(), e.Crc())
			}
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeTransaction) {
			sp := &interaction2.OperationTransaction{}
			sp.Decode(l, ctx)(r, readerOptions)
			for _, e := range sp.Entries() {
				l.Debugf("Character [%d] transaction includes [%d]. crc [%d].", s.CharacterId(), e.Data(), e.Crc())
			}
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStorePutItem) {
			sp := &interaction2.OperationPersonalStorePutItem{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] attempting to add [%d] item(s) from inventory compartment [%d] slot [%d] to store. set [%d], price [%d].", s.CharacterId(), sp.Quantity(), sp.InventoryType(), sp.Slot(), sp.Set(), sp.Price())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.AddListing(s.CharacterId(), visiting.Id(), sp.InventoryType(), sp.Slot(), sp.Quantity(), sp.Set(), sp.Price())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStoreBuy) {
			sp := &interaction2.OperationPersonalStoreBuy{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] attempting to purchase [%d] item(s) index [%d] from store.", s.CharacterId(), sp.Quantity(), sp.Index())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.PurchaseBundle(s.CharacterId(), visiting.Id(), uint16(sp.Index()), uint16(sp.Quantity()))
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStoreRemoveItem) {
			sp := &interaction2.OperationPersonalStoreRemoveItem{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] attempting to remove item index [%d] from store.", s.CharacterId(), sp.Index())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.RemoveListing(s.CharacterId(), visiting.Id(), uint16(sp.Index()))
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStoreAddToBlackList) {
			sp := &interaction2.OperationPersonalStoreAddToBlackList{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] is adding [%s] to field black list from slot [%d].", s.CharacterId(), sp.Name(), sp.Slot())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStoreSetVisitor) {
			sp := &interaction2.OperationPersonalStoreSetVisitor{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] has [%s] in their store at slot [%d]", s.CharacterId(), sp.Name(), sp.Slot())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModePersonalStoreSetBlackList) {
			sp := &interaction2.OperationPersonalStoreSetBlackList{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] has set store black list. size [%d]", s.CharacterId(), len(sp.Entries()))
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeFieldAddToBlackList) {
			sp := &interaction2.OperationFieldAddToBlackList{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] is adding [%s] to field black list.", s.CharacterId(), sp.Name())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeFieldRemoveFromBlackList) {
			sp := &interaction2.OperationFieldRemoveFromBlackList{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] is removing [%s] from field black list.", s.CharacterId(), sp.Name())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantPutItem) {
			sp := &interaction2.OperationMerchantPutItem{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] attempting to add [%d] item(s) from inventory compartment [%d] slot [%d] to merchant. set [%d], price [%d].", s.CharacterId(), sp.Quantity(), sp.InventoryType(), sp.Slot(), sp.Set(), sp.Price())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.AddListing(s.CharacterId(), visiting.Id(), sp.InventoryType(), sp.Slot(), sp.Quantity(), sp.Set(), sp.Price())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantBuy) {
			sp := &interaction2.OperationMerchantBuy{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] attempting to purchase [%d] item(s) index [%d] from merchant.", s.CharacterId(), sp.Quantity(), sp.Index())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.PurchaseBundle(s.CharacterId(), visiting.Id(), uint16(sp.Index()), uint16(sp.Quantity()))
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantRemoveItem) {
			sp := &interaction2.OperationMerchantRemoveItem{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] attempting to remove item index [%d] from merchant.", s.CharacterId(), sp.Index())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.RemoveListing(s.CharacterId(), visiting.Id(), uint16(sp.Index()))
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantMaintenanceOff) {
			l.Debugf("Character [%d] has stopped merchant maintenance.", s.CharacterId())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.ExitMaintenance(s.CharacterId(), visiting.Id())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantOrganize) {
			l.Debugf("Character [%d] has organized merchant inventory.", s.CharacterId())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.OrganizeListings(s.CharacterId(), visiting.Id())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantExit) {
			l.Debugf("Character [%d] has stopped merchant interaction.", s.CharacterId())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			// MERCHANT_EXIT (0x29) is the owner's explicit "close store" action:
			// it fully closes the shop (items back / Fredrick); from anyone else
			// it is a plain visitor exit.
			if visiting.CharacterId() == s.CharacterId() {
				_ = mp.CloseShop(s.CharacterId(), visiting.Id())
			} else {
				_ = mp.ExitShop(s.CharacterId(), visiting.Id())
			}
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantWithdrawMeso) {
			l.Debugf("Character [%d] has withdrew merchant meso.", s.CharacterId())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.WithdrawMeso(s.CharacterId(), visiting.Id())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantViewVisitList) {
			l.Debugf("Character [%d] has viewed merchant visit list.", s.CharacterId())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			visits, err := mp.GetVisits(visiting.Id().String())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visits for shop [%s].", visiting.Id())
				return
			}
			entries := make([]interactioncb.InteractionVisitListEntry, 0, len(visits))
			for _, v := range visits {
				entries = append(entries, interactioncb.InteractionVisitListEntry{Name: v.Name, Count: v.Count})
			}
			_ = session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionVisitListBody(entries))(s)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantViewBlackList) {
			l.Debugf("Character [%d] has viewed merchant black list.", s.CharacterId())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			names, err := mp.GetBlacklist(visiting.Id().String())
			if err != nil {
				l.WithError(err).Errorf("Unable to get blacklist for shop [%s].", visiting.Id())
				return
			}
			_ = session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionBlackListBody(names))(s)
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantAddToBlackList) {
			sp := &interaction2.OperationMerchantAddToBlackList{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] is adding [%s] to merchant black list.", s.CharacterId(), sp.Name())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.AddBlacklist(s.CharacterId(), visiting.Id(), sp.Name())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMerchantRemoveFromBlackList) {
			sp := &interaction2.OperationMerchantRemoveFromBlackList{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] is removing [%s] from merchant black list.", s.CharacterId(), sp.Name())
			mp := merchant.NewProcessor(l, ctx)
			visiting, err := mp.GetVisitingShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to get visiting shop for character [%d].", s.CharacterId())
				return
			}
			_ = mp.RemoveBlacklist(s.CharacterId(), visiting.Id(), sp.Name())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameAskTie) {
			l.Debugf("Character [%d] in memory game, is asking for a tie.", s.CharacterId())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameTieAnswer) {
			sp := &interaction2.OperationMemoryGameTieAnswer{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] in memory game, is answering tie request. response [%t].", s.CharacterId(), sp.Response())
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
			sp := &interaction2.OperationMemoryGameRetreatAnswer{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] in memory game, is answering retreat request. response [%t].", s.CharacterId(), sp.Response())
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
			sp := &interaction2.OperationMemoryGameMoveStone{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] in memory game, is moving stone. point [%d], color [%d].", s.CharacterId(), sp.Point(), sp.Color())
			return
		}
		if isCharacterInteraction(l)(readerOptions, mode, CharacterInteractionModeMemoryGameFlipCard) {
			sp := &interaction2.OperationMemoryGameFlipCard{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] in memory game, is flipping card [%d]. first [%t].", s.CharacterId(), sp.Index(), sp.First())
			return
		}
		l.Warnf("Character [%d] issued a unhandled character interaction [%d].", s.CharacterId(), mode)
	}
}

// pickShopByState returns the first shop in the given state.
func pickShopByState(shops []merchant.Model, state byte) (merchant.Model, bool) {
	for _, sh := range shops {
		if sh.State() == state {
			return sh, true
		}
	}
	return merchant.Model{}, false
}

// pickMerchantByState returns the first hired-merchant shop in the given state.
func pickMerchantByState(shops []merchant.Model, state byte) (merchant.Model, bool) {
	for _, sh := range shops {
		if sh.ShopType() == merchant.HiredMerchantShopType && sh.State() == state {
			return sh, true
		}
	}
	return merchant.Model{}, false
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
