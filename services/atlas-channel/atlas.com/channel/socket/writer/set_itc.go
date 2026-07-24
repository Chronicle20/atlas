package writer

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/character/teleportrock"
	"atlas-channel/maps/location"
	configuration "atlas-channel/mts/configuration"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// SetItcBody builds the SET_ITC (MTS / ITC) scene-transition body. It mirrors
// CashShopOpenBody: the same CharacterData migrate-in block (built from the
// character + buddylist + resolved map id) plus the account name, followed by
// the ITC config values and the current server time. CStage::OnSetITC reads this
// block and pushes the CITC stage so the in-game MTS view opens. The listing fee
// and auction min/max hours are read from the tenant's mts-configs configuration
// (defaults on a fetch miss). The trailing server-now FILETIME seeds the client's
// ITC clock (m_ftRel) so auction countdowns are correct — see the SetItcWriter doc.
//
// The client's commission rate/base (m_nCommissionRate / m_nCommissionBase) are
// sent as the REAL tenant knobs: atlas-mts stores the seller's BASE price, and the
// client applies commission itself — both for the register-dialog fee preview and
// the bid dialog's "Your Bid" line (CITCBidAuctionDlg::GetPriceWithCommision =
// commissionBase + (commissionRate+100)*bid/100, IDA-verified v95 0x58b5e0). The
// browse/detail contract fee (atlas-mts's withContractFee) independently shows the
// all-in price for listings the client did not just register/bid on.
func SetItcBody(a account.Model, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			trm, err := teleportrock.NewProcessor(l, ctx).GetByCharacterId(c.Id())
			if err != nil {
				// Fail-open: a missing list must never block MTS/ITC entry (design §4.4).
				l.WithError(err).Warnf("Unable to fetch teleport-rock maps for character [%d]; sending empty lists.", c.Id())
				trm = teleportrock.Model{}
			}
			cd := BuildCharacterData(c, bl, location.ResolveMapId(l, ctx, c.Id()), trm)
			t := tenant.MustFromContext(ctx)
			cfg := configuration.GetRegistry().GetTenantConfig(l, ctx, t.Id())
			return fieldcb.NewSetItcWithConfig(cd, a.Name(),
				cfg.ListingFee(),
				uint32(cfg.CommissionRate()*100), // m_nCommissionRate: percent, client applies it itself
				cfg.CommissionBase(),             // m_nCommissionBase: same
				uint32(cfg.AuctionMinHours()),
				uint32(cfg.AuctionMaxHours()),
				packetmodel.MsTimeBytes(time.Now()),
			).Encode(l, ctx)(options)
		}
	}
}
