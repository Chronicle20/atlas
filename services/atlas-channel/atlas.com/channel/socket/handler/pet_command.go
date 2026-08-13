package handler

import (
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	pet2 "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func PetCommandHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := pet2.Command{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		// p.PetId() is the client's pet serial, not the Atlas pet id.
		pm, err := pet.NewProcessor(l, ctx).GetBySerialNumber(s.CharacterId(), p.PetId())
		if err != nil {
			l.WithError(err).Debugf("Unable to resolve pet [%d] for character [%d].", p.PetId(), s.CharacterId())
			return
		}
		_ = pet.NewProcessor(l, ctx).AttemptCommand(pm.Id(), p.Command(), p.ByName(), s.CharacterId())
	}
}
