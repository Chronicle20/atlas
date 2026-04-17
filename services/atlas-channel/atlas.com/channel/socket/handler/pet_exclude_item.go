package handler

import (
	"atlas-channel/pet"
	"atlas-channel/pet/exclude"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	pet2 "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func PetItemExcludeHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := pet2.ExcludeItem{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		items := make([]exclude.Model, 0, len(p.ItemIds()))
		for i, itemId := range p.ItemIds() {
			items = append(items, exclude.NewModel(uint32(i), uint32(itemId)))
		}
		_ = pet.NewProcessor(l, ctx).SetExcludeItems(s.CharacterId(), uint32(p.PetId()), items)
	}
}
