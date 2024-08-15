package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const CharacterInfoRequestHandle = "CharacterInfoRequestHandle"

func CharacterInfoRequestHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	characterInfoFunc := session.Announce(l)(wp)(writer.CharacterInfo)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		characterId := r.ReadUint32()
		petInfo := r.ReadBool()
		l.Debugf("Received info request for character [%d]. UpdateTime [%d]. PetInfo [%t].", characterId, updateTime, petInfo)

		c, err := character.GetById(l, span, s.Tenant())(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] being requested.", characterId)
			return
		}
		err = characterInfoFunc(s, writer.CharacterInfoBody(s.Tenant())(c))
		if err != nil {
			l.WithError(err).Errorf("Unable to write character information.")
		}
	}
}
