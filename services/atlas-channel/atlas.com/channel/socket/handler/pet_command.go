package handler

import (
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	pet2 "github.com/Chronicle20/atlas-packet/pet/serverbound"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func PetCommandHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := pet2.Command{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = pet.NewProcessor(l, ctx).AttemptCommand(uint32(p.PetId()), p.Command(), p.ByName(), s.CharacterId())
	}
}
