package handler

import (
	"atlas-channel/cashshop/wishlist"
	"atlas-channel/character"
	"atlas-channel/guild"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	character2 "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterInfoRequestHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.InfoRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		cp := character.NewProcessor(l, ctx)
		decorators := make([]model.Decorator[character.Model], 0)
		if p.PetInfo() {
			decorators = append(decorators, cp.PetAssetEnrichmentDecorator)
		}
		c, err := cp.GetById(decorators...)(p.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] being requested.", p.CharacterId())
			return
		}
		g, _ := guild.NewProcessor(l, ctx).GetByMemberId(p.CharacterId())

		var wl []wishlist.Model
		wl, err = wishlist.NewProcessor(l, ctx).GetByCharacterId(p.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve wishlist for character [%d].", p.CharacterId())
			wl = make([]wishlist.Model, 0)
		}

		if p.CharacterId() != s.CharacterId() {
			var ps []pet.Model
			ps, err = pet.NewProcessor(l, ctx).GetByOwner(p.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve pet [%d] being requested.", p.CharacterId())
			}

			for _, pe := range ps {
				_ = session.Announce(l)(ctx)(wp)(writer.PetExcludeResponse)(writer.PetExcludeResponseBody(pe))(s)
			}
		}

		err = session.Announce(l)(ctx)(wp)(writer.CharacterInfo)(writer.CharacterInfoBody(c, g, wl))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write character information.")
		}
	}
}
