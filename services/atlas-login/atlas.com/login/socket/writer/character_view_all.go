package writer

import (
	"atlas-login/character"
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
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

func CharacterViewAllCountBody(worldCount uint32, unk uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(CharacterViewAll, string(CharacterViewAllCodeCharacterCount), "codes", options))
			w.WriteInt(worldCount)
			w.WriteInt(unk)
			return w.Bytes()
		}
	}
}

func CharacterViewAllSearchFailedBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(CharacterViewAll, string(CharacterViewAllCodeSearchFailed), "codes", options))
			return w.Bytes()
		}
	}
}

func CharacterViewAllErrorBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(CharacterViewAll, string(CharacterViewAllCodeErrorViewAll), "codes", options))
			return w.Bytes()
		}
	}
}

func CharacterViewAllCharacterBody(worldId world.Id, characters []character.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(CharacterViewAll, string(CharacterViewAllCodeNormal), "codes", options))
			w.WriteByte(byte(worldId))
			w.WriteByte(byte(len(characters)))
			for _, c := range characters {
				WriteCharacter(l, ctx)(w, options)(c, true)
			}

			if t.Region() == "GMS" && t.MajorVersion() > 87 {
				w.WriteByte(1) // PIC handling
			}
			return w.Bytes()
		}
	}
}
