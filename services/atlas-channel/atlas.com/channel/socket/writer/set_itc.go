package writer

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/maps/location"
	configuration "atlas-channel/mts/configuration"
	"context"
	"time"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// SetItcBody builds the SET_ITC (MTS / ITC) scene-transition body. It mirrors
// CashShopOpenBody: the same CharacterData migrate-in block (built from the
// character + buddylist + resolved map id) plus the account name, followed by
// the ITC config values and the current server time. CStage::OnSetITC reads this
// block and pushes the CITC stage so the in-game MTS view opens. The ITC config
// values (listing fee, commission rate/base, auction min/max hours) are read
// from the tenant's mts-configs configuration, falling back to defaults on a
// fetch miss. The trailing server-now FILETIME seeds the client's ITC clock
// (m_ftRel) so auction countdowns are correct — see the SetItcWriter doc.
func SetItcBody(a account.Model, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			cd := BuildCharacterData(c, bl, location.ResolveMapId(l, ctx, c.Id()))
			t := tenant.MustFromContext(ctx)
			cfg := configuration.GetRegistry().GetTenantConfig(l, ctx, t.Id())
			return fieldcb.NewSetItcWithConfig(cd, a.Name(),
				cfg.ListingFee(),
				uint32(cfg.CommissionRate()*100),
				cfg.CommissionBase(),
				uint32(cfg.AuctionMinHours()),
				uint32(cfg.AuctionMaxHours()),
				packetmodel.MsTimeBytes(time.Now()),
			).Encode(l, ctx)(options)
		}
	}
}
