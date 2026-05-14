package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterItemUpgradeWriter = "CharacterItemUpgrade"

// ItemUpgrade represents the SHOW_SCROLL_EFFECT packet (CUser::ShowItemUpgradeEffect).
//
// Wire layout — version-gated (IDA v83@0x93354d, v87@0x9adb79, v95@0x8e7b00;
// IDA JMS v185 CUser::ShowItemUpgradeEffect@0x9f1a92):
//
//	Decode4  characterId        — read by CUserPool::OnUserCommonPacket (case 0xBA) before dispatch
//	Decode1  success            — bSuccess
//	Decode1  cursed             — v4 (cursed/failed flag)
//	Decode1  legendarySpirit    — bEnchantSkill (Vega/enchant scroll category)
//	Decode4  enchantCategory    — nEnchantCategory [GMS>87 only; added after v87; absent in JMS v185]
//	Decode1  whiteScroll        — v5 (lucky/white-scroll display flag)
//	Decode1  enchantResultFlag  — v6 (passed to CUIEnchantDlg) [GMS>87 and JMS v185]
//
// IDA v83 CUser::ShowItemUpgradeEffect@0x93354d: reads Decode1×4 only (no whiteScroll, no enchantResultFlag).
// IDA v87 CUser::ShowItemUpgradeEffect@0x9adb79: reads Decode1×4 only (same as v83 — no enchantCategory, no enchantResultFlag).
// IDA v95 CUser::ShowItemUpgradeEffect@0x8e7b00: reads all 7 fields (enchantCategory+enchantResultFlag added in v95).
// IDA JMS v185 CUser::ShowItemUpgradeEffect@0x9f1a92: reads Decode1×5 (bSuccess+bCursed+bEnchantSkill+v5+v6).
// JMS has enchantResultFlag (v6) but NOT enchantCategory (Decode4). So || JMS applies only to enchantResultFlag.
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

func (m ItemUpgrade) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteBool(m.success)
		w.WriteBool(m.cursed)
		w.WriteBool(m.legendarySpirit)
		// enchantCategory and enchantResultFlag added after GMS v87 (first seen in v95 IDA).
		// IDA v83 and v87 CUser::ShowItemUpgradeEffect read only 4 × Decode1.
		// IDA JMS v185 CUser::ShowItemUpgradeEffect@0x9f1a92 reads only 5 × Decode1 (same as v83/v87).
		if t.Region() == "GMS" && t.MajorVersion() > 87 {
			w.WriteInt32(m.enchantCategory)
		}
		w.WriteBool(m.whiteScroll)
		// enchantResultFlag (v6) is present in GMS>87 AND JMS v185.
		// IDA JMS v185 CUser::ShowItemUpgradeEffect@0x9f1a92: reads Decode1(v6) — present in JMS.
		// enchantCategory (Decode4) is GMS>87 only; absent in JMS v185.
		if (t.Region() == "GMS" && t.MajorVersion() > 87) || t.Region() == "JMS" {
			w.WriteByte(m.enchantResultFlag)
		}
		return w.Bytes()
	}
}

func (m *ItemUpgrade) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.success = r.ReadBool()
		m.cursed = r.ReadBool()
		m.legendarySpirit = r.ReadBool()
		// IDA JMS v185 CUser::ShowItemUpgradeEffect@0x9f1a92: reads Decode1×5 — no Decode4(enchantCategory).
		// enchantCategory (Decode4) is GMS>87 only; absent in JMS v185.
		if t.Region() == "GMS" && t.MajorVersion() > 87 {
			m.enchantCategory = r.ReadInt32()
		}
		m.whiteScroll = r.ReadBool()
		// enchantResultFlag (v6) is present in both GMS>87 and JMS v185.
		if (t.Region() == "GMS" && t.MajorVersion() > 87) || t.Region() == "JMS" {
			m.enchantResultFlag = r.ReadByte()
		}
	}
}
