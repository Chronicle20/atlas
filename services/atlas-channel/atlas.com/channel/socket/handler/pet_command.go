package handler

import (
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const PetCommandHandle = "PetCommandHandle"

func PetCommandHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		petId := r.ReadUint64()
		byName := r.ReadBool()
		command := r.ReadByte()
		_ = pet.NewProcessor(l, ctx).AttemptCommand(uint32(petId), command, byName, s.CharacterId())
	}
}
