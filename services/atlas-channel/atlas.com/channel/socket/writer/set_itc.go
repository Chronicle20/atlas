package writer

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/maps/location"
	"context"
	"time"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// SetItcBody builds the SET_ITC (MTS / ITC) scene-transition body. It mirrors
// CashShopOpenBody: the same CharacterData migrate-in block (built from the
// character + buddylist + resolved map id) plus the account name, followed by
// the Cosmic-faithful ITC config defaults and the current server time.
// CStage::OnSetITC reads this block and pushes the CITC stage so the in-game MTS
// view opens. The trailing server-now FILETIME seeds the client's ITC clock
// (m_ftRel) so auction countdowns are correct — see the SetItcWriter doc.
func SetItcBody(a account.Model, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			cd := BuildCharacterData(c, bl, location.ResolveMapId(l, ctx, c.Id()))
			return fieldcb.NewSetItc(cd, a.Name(), packetmodel.MsTimeBytes(time.Now())).Encode(l, ctx)(options)
		}
	}
}
