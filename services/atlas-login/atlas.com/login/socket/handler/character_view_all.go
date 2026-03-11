package handler

import (
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"

	world2 "github.com/Chronicle20/atlas-constants/world"
	charpkt "github.com/Chronicle20/atlas-packet/character"
	loginpkt "github.com/Chronicle20/atlas-packet/login"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterViewAllHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := loginpkt.AllCharacterListRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		ws, err := world.NewProcessor(l, ctx).GetAll()
		if err != nil {
			l.Debugf("Unable to retrieve available worlds.")
			err = session.Announce(l)(ctx)(wp)(charpkt.CharacterViewAllWriter)(writer.CharacterViewAllErrorBody())(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write view error.")
			}
			return
		}

		var wcs = make(map[world2.Id][]character.Model)
		var count int
		for _, w := range ws {
			var cs []character.Model
			cp := character.NewProcessor(l, ctx)
			cs, err = cp.GetForWorld(cp.InventoryDecorator())(s.AccountId(), w.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve characters for account [%d] in world [%d].", s.AccountId(), w.Id())
			}
			count += len(cs)
			wcs[w.Id()] = cs
		}

		l.Debugf("Located [%d] characters for account [%d].", count, s.AccountId())
		if count == 0 {
			err = session.Announce(l)(ctx)(wp)(charpkt.CharacterViewAllWriter)(writer.CharacterViewAllSearchFailedBody())(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write search failed.")
			}
			return
		}

		err = session.Announce(l)(ctx)(wp)(charpkt.CharacterViewAllWriter)(writer.CharacterViewAllCountBody(uint32(len(ws)), uint32(count)))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write count.")
			return
		}

		for w, cs := range wcs {
			err = session.Announce(l)(ctx)(wp)(charpkt.CharacterViewAllWriter)(writer.CharacterViewAllCharacterBody(w, cs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write search failed.")
			}
		}

		return
	}
}
