package handler

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/cashshop"
	"atlas-channel/cashshop/wallet"
	"atlas-channel/character"
	mtsholding "atlas-channel/mts/holding"
	mtslisting "atlas-channel/mts/listing"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// mtsMinLevel is the configurable-min-level gate for MTS entry (design §5.1
// default 10). The client send (CWvsContext::SendMigrateToITCRequest) already
// performs guest/lie-detector/map-flag guards; the server re-checks level as the
// authoritative gate. When the per-world MTS config becomes reachable channel-
// side this default is replaced by the configured floor.
const mtsMinLevel = byte(10)

// EnterMtsHandleFunc handles the bodiless ENTER_MTS
// (CWvsContext::SendMigrateToITCRequest) — the MTS entry/migration request. It
// mirrors CashShopEntryHandleFunc: load the account + character, gate on the
// configurable min level, then announce the initial MTS state. On entry the
// channel announces, in order (mirroring Cosmic's EnterMTSHandler entry sequence):
//   - the wanted-listing-over summary (MTS_OPERATION mode 0x3D, (0,0));
//   - the wallet (MTS_OPERATION2, prepaid + points; CITC::OnQueryCashResult);
//   - the initial browse page (MTS_OPERATION GET_ITC_LIST_DONE) on tab/category 1
//     (MTS tabs are 1-indexed; category 0 is an invalid tab that crashes the
//     canvas render), with sortType=1, sortColumn=1, trailing requestSent=1;
//   - the character's take-home holdings (GET_USER_PURCHASE_ITEM_DONE); and
//   - the character's own active listings (GET_USER_SALE_ITEM_DONE), filtered by
//     sellerId. (Cosmic sends purchase before sale.)
//
// The lists are produced from atlas-mts REST via the channel-side listing/holding
// read clients; the clientbound MtsResult* codecs live in
// libs/atlas-packet/field/clientbound/mts_operation*.go.
func EnterMtsHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := fieldsb.EnterMts{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// Load the character with the same decorators CashShopEntryHandleFunc uses
		// so the SET_ITC CharacterData migrate-in block is complete (inventory,
		// pets, skills, quests).
		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator, cp.SkillModelDecorator, cp.QuestModelDecorator)(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d] attempting to enter MTS.", s.CharacterId())
			return
		}

		// Authoritative level gate (design §5.1, default 10). The cash-shop
		// map/event eligibility is enforced upstream by the client send guards
		// (guest/lie-detector/map-flag); the level floor is the server check.
		if c.Level() < mtsMinLevel {
			l.Debugf("Character [%d] level [%d] below MTS minimum [%d]; entry denied.", s.CharacterId(), c.Level(), mtsMinLevel)
			return
		}

		// SET_ITC scene transition (CStage::OnSetITC) FIRST — this pushes the
		// client's CITC stage so the in-game MTS view opens. It mirrors
		// CashShopEntryHandleFunc's CashShopOpen send: the same CharacterData
		// migrate-in block (built from account + decorated character + buddylist)
		// plus the account name and the ITC config (Cosmic-faithful defaults
		// 5000/7/500/24/168; replaced by per-tenant MTS config when reachable
		// channel-side). Without this the client never enters the ITC scene, so
		// the wallet/browse/listing announces below have no scene to render in.
		a, err := account.NewProcessor(l, ctx).GetById(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate account [%d] attempting to enter MTS.", s.AccountId())
			return
		}
		bl, err := buddylist.NewProcessor(l, ctx).GetById(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate buddylist [%d] attempting to enter MTS.", s.CharacterId())
			return
		}
		err = session.Announce(l)(ctx)(wp)(fieldcb.SetItcWriter)(writer.SetItcBody(a, c, bl))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce SET_ITC scene transition to character [%d] on MTS entry.", s.CharacterId())
			return
		}

		// Leave-field / mark-entered migration. The ITC is rendered inside the
		// cash-shop stage family (SET_ITC pushes the same CStage the cash shop
		// uses), so the migration is identical to CashShopEntryHandleFunc's
		// cashshop.Enter: emit the CharacterEnter status event so the player
		// leaves the field and is marked as in the cash-shop/ITC stage.
		err = cashshop.NewProcessor(l, ctx).Enter(s.CharacterId(), s.Field())
		if err != nil {
			l.WithError(err).Errorf("Unable to announce [%d] has entered the MTS (cash-shop stage).", s.CharacterId())
		}

		// MTSWantedListingOver (CITC::OnNormalItemResult case 61 / mode 0x3D,
		// v83 sub_5A523E): the "expired wanted listings" summary. The sub-handler
		// reads two int32s (expired NX, expired item count) and shows a StringPool
		// notice ONLY when both are > 0; with (0,0) it reads both ints cleanly and
		// shows nothing. Cosmic's EnterMTSHandler sends this before the wallet on
		// entry. Atlas's MtsOperationNotifyCancelWishResultBody is exactly this
		// 2-int body resolved at mode 0x3D, so it is reused here (no new writer).
		// IDA-verified (v83 0x5a523e): Decode4(nx), Decode4(items), gate nx>=0 && items>0.
		err = session.Announce(l)(ctx)(wp)(fieldcb.MtsOperationWriter)(fieldpkt.MtsOperationNotifyCancelWishResultBody(0, 0))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce MTS wanted-listing-over summary to character [%d] on entry.", s.CharacterId())
			return
		}

		// Initial state: wallet (reachable now). prepaid -> cash bucket, points ->
		// MaplePoints bucket (CITC::OnQueryCashResult two-bucket shape).
		w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve MTS wallet for account [%d] on entry.", s.AccountId())
			w = wallet.Model{}
		}
		err = session.Announce(l)(ctx)(wp)(fieldcb.MtsOperation2Writer)(fieldcb.NewMtsOperation2(w.Prepaid(), w.Points()).Encode)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce MTS wallet to character [%d] on entry.", s.CharacterId())
			return
		}

		// Initial browse page (GET_ITC_LIST_DONE). The ENTRY default selects the
		// first tab — category=1 (MTS tabs are 1-indexed; category 0 is not a valid
		// tab and leaves CITCWnd_List selecting a non-existent category, which
		// crashes on the next canvas render). sortType=1, sortColumn=1, and the
		// trailing requestSent=1 mirror Cosmic's sendMTS(items, tab=1, type=0,
		// page=0, ...) entry packet (which writes sortType/sortColumn bytes 1,1 and
		// the trailing 1). Reuses the synchronous browse-page writer; an empty/failed
		// page degrades to an empty list rather than blocking entry. IDA-verified
		// read order: CITC::OnGetITCListDone (v83 0x5a48af).
		writeBrowsePage(l, ctx, wp, s, 1, 0, 0, 1, 1, 1, mtslisting.BrowseFilter{})

		// The character's take-home holdings (GET_USER_PURCHASE_ITEM_DONE). Cosmic's
		// EnterMTSHandler sends transferInventory (purchase) BEFORE notYetSoldInv
		// (sale), so the purchase list is announced first.
		announceUserPurchaseItems(l, ctx, wp, s)

		// The character's own active listings (GET_USER_SALE_ITEM_DONE), filtered by
		// sellerId so only this character's active sales are returned.
		announceUserSaleItems(l, ctx, wp, s)
	}
}

// announceUserSaleItems queries the character's own active listings over REST
// (sellerId filter) and announces them as the GET_USER_SALE_ITEM_DONE result. On
// a REST error an empty list is announced so the client's "my sales" tab is not
// left hanging.
func announceUserSaleItems(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) {
	ms, err := mtslisting.NewProcessor(l, ctx).Browse(s.WorldId(), mtslisting.BrowseFilter{SellerId: s.CharacterId()})
	if err != nil {
		l.WithError(err).Errorf("Unable to load active listings for seller [%d] on entry; announcing empty sale list.", s.CharacterId())
		ms = nil
	}

	items := make([]fieldcb.MtsItem, 0, len(ms))
	for _, m := range ms {
		items = append(items, mtsItemFromListing(m))
	}

	body := fieldpkt.MtsOperationGetUserSaleItemDoneBody(items)
	if err := session.Announce(l)(ctx)(wp)(fieldcb.MtsOperationWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to announce MTS sale items to character [%d].", s.CharacterId())
	}
}

// announceUserPurchaseItems queries the character's take-home holdings over REST
// and announces them as the GET_USER_PURCHASE_ITEM_DONE result. limitedCount is 0
// (no per-account purchase cap is surfaced channel-side) and requestSent is 0. On
// a REST error an empty list is announced so the client's holding tab is not left
// hanging.
func announceUserPurchaseItems(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) {
	hs, err := mtsholding.NewProcessor(l, ctx).GetByCharacter(s.CharacterId())
	if err != nil {
		l.WithError(err).Errorf("Unable to load holdings for character [%d] on entry; announcing empty purchase list.", s.CharacterId())
		hs = nil
	}

	items := make([]fieldcb.MtsItem, 0, len(hs))
	for _, h := range hs {
		items = append(items, mtsItemFromHolding(h))
	}

	body := fieldpkt.MtsOperationGetUserPurchaseItemDoneBody(items, 0, 0)
	if err := session.Announce(l)(ctx)(wp)(fieldcb.MtsOperationWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to announce MTS purchase items to character [%d].", s.CharacterId())
	}
}

// mtsItemFromHolding maps one channel-side holding.Model to a clientbound MtsItem
// (ITCITEM) for the GET_USER_PURCHASE_ITEM_DONE list. The item-slot blob carries
// the template id and quantity; the MTS trailer carries itcSn (= the holding's
// serial, which addresses the take-home arm). A holding has no price/bid metadata,
// so the remaining trailer fields are zeroed.
func mtsItemFromHolding(m mtsholding.Model) fieldcb.MtsItem {
	// Delegates to the shared mtsholding.ToMtsItem so the entry push and the
	// consumer's post-take-home re-push produce identical wire bytes (including the
	// zeroPosition=true bare item blob — see ToMtsItem).
	return mtsholding.ToMtsItem(m)
}
