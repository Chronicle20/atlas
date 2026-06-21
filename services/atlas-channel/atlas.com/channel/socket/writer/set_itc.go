package writer

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/maps/location"
	"context"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// SetItcBody builds the SET_ITC (MTS / ITC) scene-transition body. It mirrors
// CashShopOpenBody: the same CharacterData migrate-in block (built from the
// character + buddylist + resolved map id) plus the account name, followed by
// the Cosmic-faithful ITC config defaults and contract date. CStage::OnSetITC
// reads this block and pushes the CITC stage so the in-game MTS view opens.
func SetItcBody(a account.Model, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			cd := BuildCharacterData(c, bl, location.ResolveMapId(l, ctx, c.Id()))
			return fieldcb.NewSetItc(cd, a.Name()).Encode(l, ctx)(options)
		}
	}
}
