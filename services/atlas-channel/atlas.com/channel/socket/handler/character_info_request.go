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
	"github.com/Chronicle20/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterInfoRequestHandle = "CharacterInfoRequestHandle"

func CharacterInfoRequestHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		characterId := r.ReadUint32()
		petInfo := r.ReadBool()
		l.Debugf("Received info request for character [%d]. UpdateTime [%d]. PetInfo [%t].", characterId, updateTime, petInfo)

		cp := character.NewProcessor(l, ctx)
		decorators := make([]model.Decorator[character.Model], 0)
		if petInfo {
			decorators = append(decorators, cp.PetAssetEnrichmentDecorator)
		}
		c, err := cp.GetById(decorators...)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] being requested.", characterId)
			return
		}
		g, _ := guild.NewProcessor(l, ctx).GetByMemberId(characterId)

		var wl []wishlist.Model
		wl, err = wishlist.NewProcessor(l, ctx).GetByCharacterId(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve wishlist for character [%d].", characterId)
			wl = make([]wishlist.Model, 0)
		}

		if characterId != s.CharacterId() {
			var ps []pet.Model
			ps, err = pet.NewProcessor(l, ctx).GetByOwner(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve pet [%d] being requested.", characterId)
			}

			for _, p := range ps {
				_ = session.Announce(l)(ctx)(wp)(writer.PetExcludeResponse)(writer.PetExcludeResponseBody(p))(s)
			}
		}

		err = session.Announce(l)(ctx)(wp)(writer.CharacterInfo)(writer.CharacterInfoBody(t)(c, g, wl))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write character information.")
		}
	}
}
