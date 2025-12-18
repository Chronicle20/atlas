package handler

import (
	"atlas-channel/character/key"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterKeyMapChangeHandle = "CharacterKeyMapChangeHandle"

func CharacterKeyMapChangeHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		mode := r.ReadUint32()
		if mode == 0 {
			changes := r.ReadUint32()
			for range changes {
				keyId := r.ReadInt32()
				theType := r.ReadInt8()
				action := r.ReadInt32()
				l.Debugf("Character [%d] attempting to change key [%d] to type [%d] action [%d].", s.CharacterId(), keyId, theType, action)
				err := key.NewProcessor(l, ctx).Update(s.CharacterId(), keyId, theType, action)
				if err != nil {
					l.WithError(err).Errorf("Unable to update key map for character [%d].", s.CharacterId())
				}
			}
			return
		}
		if mode == 1 {
			itemId := r.ReadUint32()
			l.Debugf("Character [%d] attempting to Auto HP potion to [%d].", s.CharacterId(), itemId)
			err := key.NewProcessor(l, ctx).Update(s.CharacterId(), 91, 7, int32(itemId))
			if err != nil {
				l.WithError(err).Errorf("Unable to update key map for character [%d].", s.CharacterId())
			}
			return
		}
		if mode == 2 {
			itemId := r.ReadUint32()
			l.Debugf("Character [%d] attempting to Auto MP potion to [%d].", s.CharacterId(), itemId)
			err := key.NewProcessor(l, ctx).Update(s.CharacterId(), 92, 7, int32(itemId))
			if err != nil {
				l.WithError(err).Errorf("Unable to update key map for character [%d].", s.CharacterId())
			}
			return
		}
	}
}
