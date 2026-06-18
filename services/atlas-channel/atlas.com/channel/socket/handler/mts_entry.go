package handler

import (
	"atlas-channel/cashshop/wallet"
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

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
// configurable min level, then announce the initial MTS state. The wallet
// (MTS_OPERATION2, prepaid + points) is reachable now via the cash-shop wallet
// processor and is announced here. The initial browse page + the character's
// active listings + their holding are produced from atlas-mts REST by the
// browse/listing/holding arm tasks (design §5.1 / §5.3); that announce is left
// as a clearly-marked seam below rather than silently stubbed.
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

		// SEAM (sibling arm tasks — design §5.1/§5.3): announce the initial browse
		// page (MTS_OPERATION GET_ITC_LIST_DONE), the character's active listings
		// (GET_USER_SALE_ITEM_DONE), and their holding (GET_USER_PURCHASE_ITEM_DONE)
		// from atlas-mts REST. Those clientbound MtsResult* codecs already exist in
		// libs/atlas-packet/field/clientbound/mts_operation.go; what is missing is
		// the channel-side atlas-mts REST client + REST->MtsItem mapping, owned by
		// the browse/listing arm tasks. Until then only the wallet is announced.
		l.Debugf("Character [%d] entered MTS; wallet announced, browse/listings/holding announce pending arm tasks.", s.CharacterId())
	}
}
