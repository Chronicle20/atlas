package writer

import (
	"atlas-login/character"
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	charpkt "github.com/Chronicle20/atlas-packet/character"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
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
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(CharacterViewAll, string(CharacterViewAllCodeCharacterCount), "codes", options)
			return charpkt.NewCharacterViewAllCount(resolved, worldCount, unk).Encode(l, ctx)(options)
		}
	}
}

func CharacterViewAllSearchFailedBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(CharacterViewAll, string(CharacterViewAllCodeSearchFailed), "codes", options)
			return charpkt.NewCharacterViewAllSearchFailed(resolved).Encode(l, ctx)(options)
		}
	}
}

func CharacterViewAllErrorBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(CharacterViewAll, string(CharacterViewAllCodeErrorViewAll), "codes", options)
			return charpkt.NewCharacterViewAllError(resolved).Encode(l, ctx)(options)
		}
	}
}

func CharacterViewAllCharacterBody(worldId world.Id, characters []character.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(CharacterViewAll, string(CharacterViewAllCodeNormal), "codes", options)
			entries := make([]packetmodel.CharacterListEntry, len(characters))
			for i, c := range characters {
				entries[i] = toCharacterListEntry(c)
			}
			return charpkt.NewCharacterViewAllCharacters(resolved, worldId, entries).Encode(l, ctx)(options)
		}
	}
}
