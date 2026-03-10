package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterItemUpgradeWriter = "CharacterItemUpgrade"

type ItemUpgrade struct {
	characterId     uint32
	success         bool
	cursed          bool
	legendarySpirit bool
	whiteScroll     bool
}

func NewItemUpgrade(characterId uint32, success bool, cursed bool, legendarySpirit bool, whiteScroll bool) ItemUpgrade {
	return ItemUpgrade{characterId: characterId, success: success, cursed: cursed, legendarySpirit: legendarySpirit, whiteScroll: whiteScroll}
}

func (m ItemUpgrade) CharacterId() uint32     { return m.characterId }
func (m ItemUpgrade) Success() bool           { return m.success }
func (m ItemUpgrade) Cursed() bool            { return m.cursed }
func (m ItemUpgrade) LegendarySpirit() bool   { return m.legendarySpirit }
func (m ItemUpgrade) WhiteScroll() bool       { return m.whiteScroll }
func (m ItemUpgrade) Operation() string       { return CharacterItemUpgradeWriter }
func (m ItemUpgrade) String() string {
	return fmt.Sprintf("item upgrade characterId [%d] success [%v]", m.characterId, m.success)
}

func (m ItemUpgrade) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteBool(m.success)
		w.WriteBool(m.cursed)
		w.WriteBool(m.legendarySpirit)
		w.WriteBool(m.whiteScroll)
		return w.Bytes()
	}
}

func (m *ItemUpgrade) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.success = r.ReadBool()
		m.cursed = r.ReadBool()
		m.legendarySpirit = r.ReadBool()
		m.whiteScroll = r.ReadBool()
	}
}
