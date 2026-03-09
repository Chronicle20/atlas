package handler

import (
	"atlas-channel/character/key"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	character2 "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterKeyMapChangeHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.KeyMapChange{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if p.Mode() == 0 {
			for _, e := range p.Entries() {
				err := key.NewProcessor(l, ctx).Update(s.CharacterId(), e.KeyId, e.TheType, e.Action)
				if err != nil {
					l.WithError(err).Errorf("Unable to update key map for character [%d].", s.CharacterId())
				}
			}
			return
		}
		if p.Mode() == 1 {
			l.Debugf("Character [%d] attempting to Auto HP potion to [%d].", s.CharacterId(), p.ItemId())
			err := key.NewProcessor(l, ctx).Update(s.CharacterId(), 91, 7, int32(p.ItemId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to update key map for character [%d].", s.CharacterId())
			}
			return
		}
		if p.Mode() == 2 {
			l.Debugf("Character [%d] attempting to Auto MP potion to [%d].", s.CharacterId(), p.ItemId())
			err := key.NewProcessor(l, ctx).Update(s.CharacterId(), 92, 7, int32(p.ItemId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to update key map for character [%d].", s.CharacterId())
			}
			return
		}
	}
}
