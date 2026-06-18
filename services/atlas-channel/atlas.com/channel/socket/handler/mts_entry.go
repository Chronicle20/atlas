package handler

import (
	"atlas-channel/cashshop/wallet"
	"atlas-channel/character"
	mtsholding "atlas-channel/mts/holding"
	mtslisting "atlas-channel/mts/listing"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
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
// channel announces, in order:
//   - the wallet (MTS_OPERATION2, prepaid + points; CITC::OnQueryCashResult);
//   - the initial browse page (MTS_OPERATION GET_ITC_LIST_DONE), page 0 of the
//     character's world;
//   - the character's own active listings (GET_USER_SALE_ITEM_DONE), filtered by
//     sellerId; and
//   - the character's take-home holdings (GET_USER_PURCHASE_ITEM_DONE).
//
// All four are produced from atlas-mts REST via the channel-side listing/holding
// read clients; the clientbound MtsResult* codecs live in
// libs/atlas-packet/field/clientbound/mts_operation*.go.
func EnterMtsHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := fieldsb.EnterMts{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById()(s.CharacterId())
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

		// Initial browse page (GET_ITC_LIST_DONE), page 0 of the character's world.
		// Reuses the synchronous browse arm's page writer (the same one the
		// GET_ITC_LIST request uses); an empty/failed page degrades to an empty list
		// inside writeBrowsePage rather than blocking entry.
		writeBrowsePage(l, ctx, wp, s, 0, 0, 0, 0, 0, mtslisting.BrowseFilter{})

		// The character's own active listings (GET_USER_SALE_ITEM_DONE), filtered by
		// sellerId so only this character's active sales are returned.
		announceUserSaleItems(l, ctx, wp, s)

		// The character's take-home holdings (GET_USER_PURCHASE_ITEM_DONE).
		announceUserPurchaseItems(l, ctx, wp, s)
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
	item := packetmodel.NewAsset(false, 0, m.TemplateId(), time.Time{}).SetStackableInfo(m.Quantity(), 0, 0)
	var dateExpired [8]byte
	return fieldpkt.MtsOperationNewItem(
		item,        // GW_ItemSlotBase blob
		m.ItcSn(),   // nITCSN = the holding serial (addresses take-home)
		0,           // nPrice
		0,           // nContractFee
		"",          // sContractFeeTxId
		"",          // sRollbackUsageID
		dateExpired, // ftITCDateExpired
		"",          // sUserID
		"",          // sGameID
		"",          // sComment
		0,           // nBidCount
		0,           // nBidRange
		0,           // nBidPrice
		0,           // nMinPrice
		0,           // nMaxPrice
		0,           // nUnitPrice
		0,           // nProcessStatus
	)
}
