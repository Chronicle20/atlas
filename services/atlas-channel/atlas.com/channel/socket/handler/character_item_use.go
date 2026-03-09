package handler

import (
	"atlas-channel/consumable"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-constants/item"
	inventory2 "github.com/Chronicle20/atlas-packet/inventory"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := inventory2.NewItemUse(inventory2.CharacterItemUseHandle)
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character.Id(s.CharacterId()), item.Id(p.ItemId()), slot.Position(p.Source()), p.UpdateTime())
	}
}

func CharacterItemUseTownScrollHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := inventory2.NewItemUse(inventory2.CharacterItemUseTownScrollHandle)
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character.Id(s.CharacterId()), item.Id(p.ItemId()), slot.Position(p.Source()), p.UpdateTime())
	}
}

func CharacterItemUseScrollHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := inventory2.ScrollUse{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = consumable.NewProcessor(l, ctx).RequestScrollUse(s.Field(), character.Id(s.CharacterId()), slot.Position(p.ScrollSlot()), slot.Position(p.EquipSlot()), p.WhiteScroll(), p.LegendarySpirit(), p.UpdateTime())
	}
}

func CharacterItemUseSummonBagHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := inventory2.NewItemUse(inventory2.CharacterItemUseSummonBagHandle)
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character.Id(s.CharacterId()), item.Id(p.ItemId()), slot.Position(p.Source()), p.UpdateTime())
	}
}
