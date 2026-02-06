package handler

import (
	"atlas-channel/pet"
	"atlas-channel/pet/exclude"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const PetItemExcludeHandle = "PetItemExcludeHandle"

func PetItemExcludeHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		petId := r.ReadUint64()
		items := make([]exclude.Model, 0)
		count := r.ReadByte()
		for i := range count {
			itemId := r.ReadInt32()
			items = append(items, exclude.NewModel(uint32(i), uint32(itemId)))
		}
		_ = pet.NewProcessor(l, ctx).SetExcludeItems(s.CharacterId(), uint32(petId), items)
	}
}
