package handler

import (
	"atlas-login/character/factory"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CreateCharacterHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.CreateCharacter{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		err := factory.NewProcessor(l, ctx).SeedCharacter(s.AccountId(), s.WorldId(), p.Name(), p.JobIndex(), p.SubJobIndex(), p.Face(), p.Hair(), p.HairColor(), p.SkinColor(), p.Gender(), p.TopTemplateId(), p.BottomTemplateId(), p.ShoesTemplateId(), p.WeaponTemplateId(), p.Strength(), p.Dexterity(), p.Intelligence(), p.Luck())
		if err != nil {
			l.WithError(err).Errorf("Error creating character from seed.")
			err = session.Announce(l)(ctx)(wp)(charcb.AddCharacterEntryWriter)(writer.AddCharacterErrorBody(writer.AddCharacterCodeUnknownError))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to show newly created character.")
			}
			return
		}
	}
}
