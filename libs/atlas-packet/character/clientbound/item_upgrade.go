package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterItemUpgradeWriter = "CharacterItemUpgrade"

// ItemUpgrade represents the SHOW_SCROLL_EFFECT packet (CUser::ShowItemUpgradeEffect).
//
// Wire layout (IDA CUser::ShowItemUpgradeEffect, outer dispatcher reads characterId first):
//
//	Decode4  characterId        — read by CUserPool::OnUserCommonPacket (case 186 = 0xBA)
//	Decode1  success            — bSuccess
//	Decode1  cursed             — v4 (cursed/failed flag)
//	Decode1  legendarySpirit    — bEnchantSkill (true = Vega/enchant scroll category)
//	Decode4  enchantCategory    — nEnchantCategory (enchant type; 0 for normal scrolls)
//	Decode1  whiteScroll        — v5 (lucky/white-scroll display flag)
//	Decode1  enchantResultFlag  — v6 (passed to CUIEnchantDlg; 0 for normal scrolls)
type ItemUpgrade struct {
	characterId      uint32
	success          bool
	cursed           bool
	legendarySpirit  bool
	enchantCategory  int32
	whiteScroll      bool
	enchantResultFlag byte
}

// NewItemUpgrade constructs an ItemUpgrade for a normal (non-enchant) scroll result.
// enchantCategory and enchantResultFlag default to 0 (correct for all non-Vega scrolls).
func NewItemUpgrade(characterId uint32, success bool, cursed bool, legendarySpirit bool, whiteScroll bool) ItemUpgrade {
	return ItemUpgrade{
		characterId:     characterId,
		success:         success,
		cursed:          cursed,
		legendarySpirit: legendarySpirit,
		whiteScroll:     whiteScroll,
	}
}

// NewItemUpgradeEnchant constructs an ItemUpgrade for an enchant/Vega scroll result.
func NewItemUpgradeEnchant(characterId uint32, success bool, cursed bool, legendarySpirit bool, enchantCategory int32, whiteScroll bool, enchantResultFlag byte) ItemUpgrade {
	return ItemUpgrade{
		characterId:       characterId,
		success:           success,
		cursed:            cursed,
		legendarySpirit:   legendarySpirit,
		enchantCategory:   enchantCategory,
		whiteScroll:       whiteScroll,
		enchantResultFlag: enchantResultFlag,
	}
}

func (m ItemUpgrade) CharacterId() uint32       { return m.characterId }
func (m ItemUpgrade) Success() bool             { return m.success }
func (m ItemUpgrade) Cursed() bool              { return m.cursed }
func (m ItemUpgrade) LegendarySpirit() bool     { return m.legendarySpirit }
func (m ItemUpgrade) EnchantCategory() int32    { return m.enchantCategory }
func (m ItemUpgrade) WhiteScroll() bool         { return m.whiteScroll }
func (m ItemUpgrade) EnchantResultFlag() byte   { return m.enchantResultFlag }
func (m ItemUpgrade) Operation() string         { return CharacterItemUpgradeWriter }
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
		w.WriteInt32(m.enchantCategory)
		w.WriteBool(m.whiteScroll)
		w.WriteByte(m.enchantResultFlag)
		return w.Bytes()
	}
}

func (m *ItemUpgrade) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.success = r.ReadBool()
		m.cursed = r.ReadBool()
		m.legendarySpirit = r.ReadBool()
		m.enchantCategory = r.ReadInt32()
		m.whiteScroll = r.ReadBool()
		m.enchantResultFlag = r.ReadByte()
	}
}
