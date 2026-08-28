package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// MakerSkillHandle is the config key for the CUIItemMaker RequestItemMake
// serverbound handler.
const MakerSkillHandle = "MakerSkillHandle"

// Maker request modes, as sent in nRecipeClass. The value is the client's
// m_nRecipeClass member, which the UI sets from the recipe currently loaded in
// the maker dialog. Any value outside 1..4 produces a bodyless request: the
// client's guard (`if (v4 > 0)` on v72-v87/jms185, `switch` + `default: break`
// on v92/v95) skips every Encode, including the mode itself.
const (
	MakerModeCreate            = uint32(1)
	MakerModeCreateWithUpgrade = uint32(2)
	MakerModeMonsterCrystal    = uint32(3)
	MakerModeDisassemble       = uint32(4)
)

// MakerSkill - CUIItemMaker::RequestItemMake. The maker-skill crafting request.
//
// Wire layout, derived per version in
// docs/tasks/task-285-maker-skill-crafting/wire-derivation.md §"MAKER_SKILL —
// serverbound encode order" and IDENTICAL on all eight applicable versions
// (gms_v72 0x760cc3, gms_v79 0x795dc3, gms_v83 0x827096, gms_v84 0x8524b7,
// gms_v87 0x88afd1, gms_v92 0x7afdc0, gms_v95 0x7d58d0, jms_v185 0x8b1040 —
// only the opcode moves, and the registry already carries that). Because there
// is no divergence there is no version gate here: no tenant lookup, no
// MajorAtLeast branch, no docs/packets/gates.yaml entry.
//
//	i32 nRecipeClass                      // the mode; Encode4, NOT a byte
//	mode 1|2: i32 nTargetItemID
//	          u8  bCatalystMounted
//	          i32 nNumGemMounted          // length prefix
//	          nNumGemMounted x i32 nGemItemID
//	mode 3:   i32 nRecipeItemID           // the leftover/monster-crystal source
//	mode 4:   i32 nRecipeItemID
//	          i32 nTI_DisassembleItem     // inventory type
//	          i32 nSlotPosition_DisassembleItem
//	otherwise: empty body
//
// The mode is encoded exactly ONCE, as the first Encode4 *inside* the selected
// arm (gms_v72: 0x760de7 mode 1/2, 0x760dcd mode 3, 0x760d8c mode 4). There is
// no pre-switch mode encode; the double-encode shown in
// docs/tasks/task-285-maker-skill-crafting/evidence-maker-skill-v72-v79.md and
// in prd.md:117-136 is a transcription artefact.
//
// Nothing decoded here is trusted: the item ids, the inventory type and the
// slot position are client-supplied and must be re-validated server-side
// against the character's real inventory (PRD §8).
// packet-audit:fname CUIItemMaker::RequestItemMake
type MakerSkill struct {
	mode           uint32
	targetItemId   uint32
	useCatalyst    bool
	gemItemIds     []uint32
	leftoverItemId uint32
	itemId         uint32
	inventoryType  uint32
	slotPos        uint32
}

// NewMakerSkill builds a MakerSkill. Only the fields belonging to the given
// mode's arm are carried onto the wire; the rest are ignored by Encode and left
// zero by Decode.
func NewMakerSkill(mode uint32, targetItemId uint32, useCatalyst bool, gemItemIds []uint32, leftoverItemId uint32, itemId uint32, inventoryType uint32, slotPos uint32) MakerSkill {
	return MakerSkill{
		mode:           mode,
		targetItemId:   targetItemId,
		useCatalyst:    useCatalyst,
		gemItemIds:     append([]uint32(nil), gemItemIds...),
		leftoverItemId: leftoverItemId,
		itemId:         itemId,
		inventoryType:  inventoryType,
		slotPos:        slotPos,
	}
}

func (m MakerSkill) Mode() uint32         { return m.mode }
func (m MakerSkill) TargetItemId() uint32 { return m.targetItemId }
func (m MakerSkill) UseCatalyst() bool    { return m.useCatalyst }

// GemItemIds returns a copy so the caller cannot mutate the decoded request.
func (m MakerSkill) GemItemIds() []uint32 {
	if m.gemItemIds == nil {
		return nil
	}
	return append([]uint32(nil), m.gemItemIds...)
}

func (m MakerSkill) LeftoverItemId() uint32 { return m.leftoverItemId }
func (m MakerSkill) ItemId() uint32         { return m.itemId }
func (m MakerSkill) InventoryType() uint32  { return m.inventoryType }
func (m MakerSkill) SlotPos() uint32        { return m.slotPos }

func (m MakerSkill) Operation() string {
	return MakerSkillHandle
}

func (m MakerSkill) String() string {
	switch m.mode {
	case MakerModeCreate, MakerModeCreateWithUpgrade:
		return fmt.Sprintf("mode [%d] targetItemId [%d] useCatalyst [%t] gemItemIds %v", m.mode, m.targetItemId, m.useCatalyst, m.gemItemIds)
	case MakerModeMonsterCrystal:
		return fmt.Sprintf("mode [%d] leftoverItemId [%d]", m.mode, m.leftoverItemId)
	case MakerModeDisassemble:
		return fmt.Sprintf("mode [%d] itemId [%d] inventoryType [%d] slotPos [%d]", m.mode, m.itemId, m.inventoryType, m.slotPos)
	}
	return fmt.Sprintf("mode [%d]", m.mode)
}

func (m MakerSkill) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		switch m.mode {
		case MakerModeCreate, MakerModeCreateWithUpgrade:
			w.WriteInt(m.mode)
			w.WriteInt(m.targetItemId)
			w.WriteBool(m.useCatalyst)
			w.WriteInt(uint32(len(m.gemItemIds)))
			for _, id := range m.gemItemIds {
				w.WriteInt(id)
			}
		case MakerModeMonsterCrystal:
			w.WriteInt(m.mode)
			w.WriteInt(m.leftoverItemId)
		case MakerModeDisassemble:
			w.WriteInt(m.mode)
			w.WriteInt(m.itemId)
			w.WriteInt(m.inventoryType)
			w.WriteInt(m.slotPos)
		default:
			// The client emits the opcode with an empty body for any mode
			// outside 1..4 - the mode itself is never encoded.
		}
		return w.Bytes()
	}
}

func (m *MakerSkill) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadUint32()
		switch m.mode {
		case MakerModeCreate, MakerModeCreateWithUpgrade:
			m.targetItemId = r.ReadUint32()
			m.useCatalyst = r.ReadBool()
			// nNumGemMounted is a client-supplied length prefix, so the loop is
			// additionally bounded by what is actually left in the buffer: a
			// hostile count must not drive an unbounded allocation.
			count := r.ReadUint32()
			gems := make([]uint32, 0, min(count, uint32(r.Available()/4)))
			for i := uint32(0); i < count && r.Available() >= 4; i++ {
				gems = append(gems, r.ReadUint32())
			}
			m.gemItemIds = gems
		case MakerModeMonsterCrystal:
			m.leftoverItemId = r.ReadUint32()
		case MakerModeDisassemble:
			m.itemId = r.ReadUint32()
			m.inventoryType = r.ReadUint32()
			m.slotPos = r.ReadUint32()
		}
	}
}
