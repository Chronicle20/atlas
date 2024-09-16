package writer

import (
	"atlas-login/character"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterViewAll = "CharacterViewAll"

type CharacterViewAllCode string

const (
	CharacterViewAllCodeNormal         CharacterViewAllCode = "NORMAL"
	CharacterViewAllCodeCharacterCount CharacterViewAllCode = "CHARACTER_COUNT"
	CharacterViewAllCodeErrorViewAll   CharacterViewAllCode = "ERROR_VIEW_ALL"
	CharacterViewAllCodeSearchFailed   CharacterViewAllCode = "SEARCH_FAILED"
	CharacterViewAllCodeSearchFailed2  CharacterViewAllCode = "SEARCH_FAILED_2"
	CharacterViewAllCodeErrorViewAll2  CharacterViewAllCode = "ERROR_VIEW_ALL_2"
)

func CharacterViewAllCountBody(l logrus.FieldLogger) func(worldCount uint32, unk uint32) BodyProducer {
	return func(worldCount uint32, unk uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(CharacterViewAll, string(CharacterViewAllCodeCharacterCount), "codes", options))
			w.WriteInt(worldCount)
			w.WriteInt(unk)
			return w.Bytes()
		}
	}
}

func CharacterViewAllSearchFailedBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(CharacterViewAll, string(CharacterViewAllCodeSearchFailed), "codes", options))
			return w.Bytes()
		}
	}
}

func CharacterViewAllErrorBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(CharacterViewAll, string(CharacterViewAllCodeErrorViewAll), "codes", options))
			return w.Bytes()
		}
	}
}

func CharacterViewAllCharacterBody(l logrus.FieldLogger, tenant tenant.Model) func(worldId byte, characters []character.Model) BodyProducer {
	return func(worldId byte, characters []character.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(CharacterViewAll, string(CharacterViewAllCodeNormal), "codes", options))
			w.WriteByte(worldId)
			w.WriteByte(byte(len(characters)))
			for _, c := range characters {
				WriteCharacter(tenant)(w, c, true)
			}

			if tenant.Region() == "GMS" && tenant.MajorVersion() > 87 {
				w.WriteByte(1) // PIC handling
			}
			return w.Bytes()
		}
	}
}
