package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/expression"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	character2 "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// expressionItemOwnedFunc is a test seam for the extra-expression ownership
// check (package-var injection precedent: cashItemInSlotFunc in
// character_cash_item_use.go). Handler tests must not require a live character
// service to assert which branch a request reached. It returns only a bool
// because nothing downstream needs the asset itself.
var expressionItemOwnedFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, itemId item.Id) (bool, error) {
	cp := character.NewProcessor(l, ctx)
	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		return false, err
	}
	_, ok := c.Inventory().Cash().FindFirstByItemId(uint32(itemId))
	return ok, nil
}

func CharacterExpressionHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.ExpressionRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		emote := p.Emote()

		// CWvsContext::SendEmotionChange@0x9f9386 (GMS v95) refuses to send an
		// emotion above 0x17, and the receiving CAvatar::SetEmotion@0x466b00
		// drops one anyway, so a larger value cannot come from a stock client.
		if emote > item.MaxEmoteId {
			l.Warnf("Character [%d] requested out-of-range expression [%d]. Dropping.", s.CharacterId(), emote)
			return
		}

		// Extra expressions require the matching 516xxxx cash item. The stock
		// client applies the same rule itself: CUserLocal::UseFuncKeyMapped
		// case 3u@0x933884 gates on CWvsContext::IsExist(nItemID) before
		// sending. Fail closed on a lookup error — a broken character service
		// must not read as ownership.
		if itemId, ok := item.ExtraExpressionItemId(emote); ok {
			owns, err := expressionItemOwnedFunc(l, ctx, s.CharacterId(), itemId)
			if err != nil {
				l.WithError(err).Warnf("Unable to verify character [%d] owns item [%d] for expression [%d]. Dropping.", s.CharacterId(), itemId, emote)
				return
			}
			if !owns {
				l.Warnf("Character [%d] requested expression [%d] without owning item [%d]. Dropping.", s.CharacterId(), emote, itemId)
				return
			}
		}

		_ = expression.NewProcessor(l, ctx).Change(s.CharacterId(), s.Field(), emote, p.Duration(), p.ByItemOption())
	}
}
