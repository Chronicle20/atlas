package handler

import (
	"atlas-channel/character"
	"atlas-channel/guild"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterInfoRequestHandle = "CharacterInfoRequestHandle"

func CharacterInfoRequestHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	characterInfoFunc := session.Announce(l)(ctx)(wp)(writer.CharacterInfo)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		characterId := r.ReadUint32()
		petInfo := r.ReadBool()
		l.Debugf("Received info request for character [%d]. UpdateTime [%d]. PetInfo [%t].", characterId, updateTime, petInfo)

		c, err := character.GetById(l)(ctx)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] being requested.", characterId)
			return
		}
		g, _ := guild.GetByMemberId(l)(ctx)(characterId)

		err = characterInfoFunc(s, writer.CharacterInfoBody(t)(c, g))
		if err != nil {
			l.WithError(err).Errorf("Unable to write character information.")
		}
	}
}
