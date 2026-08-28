package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Discrete per-mode body codecs for the CUserLocal::OnMakerResult dispatcher
// (MAKER_RESULT). The client reads an i32 nResult FIRST; only when
// nResult ∈ {0, 1} does it read the i32 nMode and switch-dispatch to one of the
// four arm bodies. A nResult outside {0, 1} makes the packet bodyless — the
// client falls straight through to CUIItemMaker::OnItemMakeResult(nResult,0,0,0)
// and reads nothing further. That bodyless shape is MakerResultFailed, which is
// why it is the one arm whose constructor takes no mode.
//
// The mode is an Encode4 on the wire, but every constructor takes `mode byte`:
// atlas_packet.WithResolvedCode resolves the tenant "operations" table value as
// a byte (libs/atlas-packet/resolve.go:41 — `factory func(byte) packet.Encoder`),
// and the struct widens it on write. Never hard-code a mode literal here.
//
// Derived from the live IDBs in
// docs/tasks/task-285-maker-skill-crafting/wire-derivation.md. All EIGHT
// versions are field-for-field IDENTICAL, so these codecs carry no version gate:
//
//	gms_v72  CUserLocal::OnMakerResult @ 0x86a152
//	gms_v79                            @ 0x8b5af5
//	gms_v83                            @ 0x95dad3
//	gms_v84                            @ 0x99bdbc
//	gms_v87                            @ 0x9e01b2
//	gms_v92                            @ 0x8f5d70
//	gms_v95                            @ 0x9102f0  (reference; best-typed IDB)
//	jms_v185                           @ 0xa29527

// MakerResultWriter is the registry writer name (Operation()) shared by every
// per-mode CUserLocal::OnMakerResult (MAKER_RESULT) arm codec in this file.
const MakerResultWriter = "MakerResult"

// MakerMaterial is the repeated (itemId, count) pair the create arms use for
// nNumUsedItem entries and the disassemble arm uses for nNumRewardItem entries.
// Both loops read `Decode4 nItemID; Decode4 nCount` per iteration
// (gms_v95 0x91082a/0x91083d for create, 0x9105a9/0x9105bc for disassemble).
type MakerMaterial struct {
	itemId uint32
	count  uint32
}

func NewMakerMaterial(itemId uint32, count uint32) MakerMaterial {
	return MakerMaterial{itemId: itemId, count: count}
}

func (m MakerMaterial) ItemId() uint32 { return m.itemId }
func (m MakerMaterial) Count() uint32  { return m.count }
func (m MakerMaterial) String() string {
	return fmt.Sprintf("itemId [%d] count [%d]", m.itemId, m.count)
}

// writeMakerCreateBody writes the shared mode-1 / mode-2 arm body (the two arms
// are wire-identical; they remain separate structs because they are distinct
// operations to the client and a struct may carry only one #-entry).
//
// gms_v95 read order: Decode1 bNoItemGain @0x910717; when it is ZERO,
// Decode4 nTargetItemID @0x910732 and Decode4 nItemCount @0x910746 follow —
// a TRUTHY byte SUPPRESSES the pair. Then Decode4 nNumUsedItem @0x91080e and
// its loop, Decode4 nNumUsedGem @0x9108ef and its id-only loop @0x910904,
// Decode1 bUsedCatalyst @0x9109c6 (gating Decode4 nCatalystItemID @0x9109d8),
// and finally Decode4 nMesoCost @0x910aa2.
func writeMakerCreateBody(w *response.Writer, noItemAwarded bool, targetItemId uint32, itemNum uint32, materials []MakerMaterial, gemItemIds []uint32, catalystUsed bool, catalystItemId uint32, mesoCost uint32) {
	w.WriteBool(noItemAwarded) // Decode1 bNoItemGain — inverted: truthy suppresses the pair
	if !noItemAwarded {
		w.WriteInt(targetItemId) // Decode4 nTargetItemID
		w.WriteInt(itemNum)      // Decode4 nItemCount
	}
	w.WriteInt(uint32(len(materials))) // Decode4 nNumUsedItem
	for _, mat := range materials {
		w.WriteInt(mat.itemId)
		w.WriteInt(mat.count)
	}
	w.WriteInt(uint32(len(gemItemIds))) // Decode4 nNumUsedGem
	for _, id := range gemItemIds {
		w.WriteInt(id)
	}
	w.WriteBool(catalystUsed) // Decode1 bUsedCatalyst
	if catalystUsed {
		w.WriteInt(catalystItemId) // Decode4 nCatalystItemID
	}
	w.WriteInt(mesoCost) // Decode4 nMesoCost — a COST, rendered as a loss by the client
}

// readMakerCreateBody is the exact field-for-field mirror of writeMakerCreateBody.
func readMakerCreateBody(r *request.Reader) (noItemAwarded bool, targetItemId uint32, itemNum uint32, materials []MakerMaterial, gemItemIds []uint32, catalystUsed bool, catalystItemId uint32, mesoCost uint32) {
	noItemAwarded = r.ReadBool()
	if !noItemAwarded {
		targetItemId = r.ReadUint32()
		itemNum = r.ReadUint32()
	}
	materialCount := r.ReadUint32()
	materials = make([]MakerMaterial, 0, materialCount)
	for i := uint32(0); i < materialCount; i++ {
		id := r.ReadUint32()
		count := r.ReadUint32()
		materials = append(materials, NewMakerMaterial(id, count))
	}
	gemCount := r.ReadUint32()
	gemItemIds = make([]uint32, 0, gemCount)
	for i := uint32(0); i < gemCount; i++ {
		gemItemIds = append(gemItemIds, r.ReadUint32())
	}
	catalystUsed = r.ReadBool()
	if catalystUsed {
		catalystItemId = r.ReadUint32()
	}
	mesoCost = r.ReadUint32()
	return
}

// MakerResultCreate — the mode-1 arm (plain item creation). Writes the shared
// i32 nResult header, the i32 nMode, then the create body.
//
// packet-audit:fname CUserLocal::OnMakerResult#Create
type MakerResultCreate struct {
	mode           byte
	result         uint32
	noItemAwarded  bool
	targetItemId   uint32
	itemNum        uint32
	materials      []MakerMaterial
	gemItemIds     []uint32
	catalystUsed   bool
	catalystItemId uint32
	mesoCost       uint32
}

func NewMakerResultCreate(mode byte, result uint32, noItemAwarded bool, targetItemId uint32, itemNum uint32, materials []MakerMaterial, gemItemIds []uint32, catalystUsed bool, catalystItemId uint32, mesoCost uint32) MakerResultCreate {
	return MakerResultCreate{
		mode:           mode,
		result:         result,
		noItemAwarded:  noItemAwarded,
		targetItemId:   targetItemId,
		itemNum:        itemNum,
		materials:      materials,
		gemItemIds:     gemItemIds,
		catalystUsed:   catalystUsed,
		catalystItemId: catalystItemId,
		mesoCost:       mesoCost,
	}
}

func (m MakerResultCreate) Mode() byte                 { return m.mode }
func (m MakerResultCreate) Result() uint32             { return m.result }
func (m MakerResultCreate) NoItemAwarded() bool        { return m.noItemAwarded }
func (m MakerResultCreate) TargetItemId() uint32       { return m.targetItemId }
func (m MakerResultCreate) ItemNum() uint32            { return m.itemNum }
func (m MakerResultCreate) Materials() []MakerMaterial { return m.materials }
func (m MakerResultCreate) GemItemIds() []uint32       { return m.gemItemIds }
func (m MakerResultCreate) CatalystUsed() bool         { return m.catalystUsed }
func (m MakerResultCreate) CatalystItemId() uint32     { return m.catalystItemId }
func (m MakerResultCreate) MesoCost() uint32           { return m.mesoCost }
func (m MakerResultCreate) Operation() string          { return MakerResultWriter }
func (m MakerResultCreate) String() string {
	return fmt.Sprintf("maker result create mode [%d] result [%d] noItemAwarded [%t] targetItemId [%d] itemNum [%d] materials [%d] gems [%d] catalystUsed [%t] catalystItemId [%d] mesoCost [%d]", m.mode, m.result, m.noItemAwarded, m.targetItemId, m.itemNum, len(m.materials), len(m.gemItemIds), m.catalystUsed, m.catalystItemId, m.mesoCost)
}

func (m MakerResultCreate) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.result)       // Decode4 nResult
		w.WriteInt(uint32(m.mode)) // Decode4 nMode (resolved byte, widened)
		writeMakerCreateBody(w, m.noItemAwarded, m.targetItemId, m.itemNum, m.materials, m.gemItemIds, m.catalystUsed, m.catalystItemId, m.mesoCost)
		return w.Bytes()
	}
}

func (m *MakerResultCreate) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.result = r.ReadUint32()
		m.mode = byte(r.ReadUint32())
		m.noItemAwarded, m.targetItemId, m.itemNum, m.materials, m.gemItemIds, m.catalystUsed, m.catalystItemId, m.mesoCost = readMakerCreateBody(r)
	}
}

// MakerResultCreateWithUpgrade — the mode-2 arm (creation with an equip upgrade
// applied). Its body is wire-identical to the mode-1 arm; it is a separate
// struct because one struct may carry only one #-entry (INV-1) and the two modes
// are distinct operations to the client.
//
// packet-audit:fname CUserLocal::OnMakerResult#CreateWithUpgrade
type MakerResultCreateWithUpgrade struct {
	mode           byte
	result         uint32
	noItemAwarded  bool
	targetItemId   uint32
	itemNum        uint32
	materials      []MakerMaterial
	gemItemIds     []uint32
	catalystUsed   bool
	catalystItemId uint32
	mesoCost       uint32
}

func NewMakerResultCreateWithUpgrade(mode byte, result uint32, noItemAwarded bool, targetItemId uint32, itemNum uint32, materials []MakerMaterial, gemItemIds []uint32, catalystUsed bool, catalystItemId uint32, mesoCost uint32) MakerResultCreateWithUpgrade {
	return MakerResultCreateWithUpgrade{
		mode:           mode,
		result:         result,
		noItemAwarded:  noItemAwarded,
		targetItemId:   targetItemId,
		itemNum:        itemNum,
		materials:      materials,
		gemItemIds:     gemItemIds,
		catalystUsed:   catalystUsed,
		catalystItemId: catalystItemId,
		mesoCost:       mesoCost,
	}
}

func (m MakerResultCreateWithUpgrade) Mode() byte                 { return m.mode }
func (m MakerResultCreateWithUpgrade) Result() uint32             { return m.result }
func (m MakerResultCreateWithUpgrade) NoItemAwarded() bool        { return m.noItemAwarded }
func (m MakerResultCreateWithUpgrade) TargetItemId() uint32       { return m.targetItemId }
func (m MakerResultCreateWithUpgrade) ItemNum() uint32            { return m.itemNum }
func (m MakerResultCreateWithUpgrade) Materials() []MakerMaterial { return m.materials }
func (m MakerResultCreateWithUpgrade) GemItemIds() []uint32       { return m.gemItemIds }
func (m MakerResultCreateWithUpgrade) CatalystUsed() bool         { return m.catalystUsed }
func (m MakerResultCreateWithUpgrade) CatalystItemId() uint32     { return m.catalystItemId }
func (m MakerResultCreateWithUpgrade) MesoCost() uint32           { return m.mesoCost }
func (m MakerResultCreateWithUpgrade) Operation() string          { return MakerResultWriter }
func (m MakerResultCreateWithUpgrade) String() string {
	return fmt.Sprintf("maker result create with upgrade mode [%d] result [%d] noItemAwarded [%t] targetItemId [%d] itemNum [%d] materials [%d] gems [%d] catalystUsed [%t] catalystItemId [%d] mesoCost [%d]", m.mode, m.result, m.noItemAwarded, m.targetItemId, m.itemNum, len(m.materials), len(m.gemItemIds), m.catalystUsed, m.catalystItemId, m.mesoCost)
}

func (m MakerResultCreateWithUpgrade) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.result)
		w.WriteInt(uint32(m.mode))
		writeMakerCreateBody(w, m.noItemAwarded, m.targetItemId, m.itemNum, m.materials, m.gemItemIds, m.catalystUsed, m.catalystItemId, m.mesoCost)
		return w.Bytes()
	}
}

func (m *MakerResultCreateWithUpgrade) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.result = r.ReadUint32()
		m.mode = byte(r.ReadUint32())
		m.noItemAwarded, m.targetItemId, m.itemNum, m.materials, m.gemItemIds, m.catalystUsed, m.catalystItemId, m.mesoCost = readMakerCreateBody(r)
	}
}

// MakerResultMonsterCrystal — the mode-3 arm (monster-crystal refinement). Two
// Decode4 reads and nothing else; there is NO meso field on this arm's wire.
//
//	gms_v95 Decode4 nTargetItemID @0x91037a, Decode4 nSourceItemID @0x91038d
//
// packet-audit:fname CUserLocal::OnMakerResult#MonsterCrystal
type MakerResultMonsterCrystal struct {
	mode           byte
	result         uint32
	crystalItemId  uint32
	leftoverItemId uint32
}

func NewMakerResultMonsterCrystal(mode byte, result uint32, crystalItemId uint32, leftoverItemId uint32) MakerResultMonsterCrystal {
	return MakerResultMonsterCrystal{mode: mode, result: result, crystalItemId: crystalItemId, leftoverItemId: leftoverItemId}
}

func (m MakerResultMonsterCrystal) Mode() byte             { return m.mode }
func (m MakerResultMonsterCrystal) Result() uint32         { return m.result }
func (m MakerResultMonsterCrystal) CrystalItemId() uint32  { return m.crystalItemId }
func (m MakerResultMonsterCrystal) LeftoverItemId() uint32 { return m.leftoverItemId }
func (m MakerResultMonsterCrystal) Operation() string      { return MakerResultWriter }
func (m MakerResultMonsterCrystal) String() string {
	return fmt.Sprintf("maker result monster crystal mode [%d] result [%d] crystalItemId [%d] leftoverItemId [%d]", m.mode, m.result, m.crystalItemId, m.leftoverItemId)
}

func (m MakerResultMonsterCrystal) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.result)         // Decode4 nResult
		w.WriteInt(uint32(m.mode))   // Decode4 nMode
		w.WriteInt(m.crystalItemId)  // Decode4 nTargetItemID
		w.WriteInt(m.leftoverItemId) // Decode4 nSourceItemID
		return w.Bytes()
	}
}

func (m *MakerResultMonsterCrystal) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.result = r.ReadUint32()
		m.mode = byte(r.ReadUint32())
		m.crystalItemId = r.ReadUint32()
		m.leftoverItemId = r.ReadUint32()
	}
}

// MakerResultDisassemble — the mode-4 arm (equipment disassembly).
//
//	gms_v95 Decode4 nDisassembledItemID @0x910516, Decode4 nNumRewardItem
//	@0x91058b with its {id @0x9105a9, count @0x9105bc} loop, then Decode4
//	nMesoCost @0x91068f. The meso is what the operation CHARGED — the client
//	renders it via Format(SP_292_YOU_HAVE_LOST_MESOS_D, -v).
//
// packet-audit:fname CUserLocal::OnMakerResult#Disassemble
type MakerResultDisassemble struct {
	mode               byte
	result             uint32
	disassembledItemId uint32
	crystals           []MakerMaterial
	mesoCost           uint32
}

func NewMakerResultDisassemble(mode byte, result uint32, disassembledItemId uint32, crystals []MakerMaterial, mesoCost uint32) MakerResultDisassemble {
	return MakerResultDisassemble{mode: mode, result: result, disassembledItemId: disassembledItemId, crystals: crystals, mesoCost: mesoCost}
}

func (m MakerResultDisassemble) Mode() byte                 { return m.mode }
func (m MakerResultDisassemble) Result() uint32             { return m.result }
func (m MakerResultDisassemble) DisassembledItemId() uint32 { return m.disassembledItemId }
func (m MakerResultDisassemble) Crystals() []MakerMaterial  { return m.crystals }
func (m MakerResultDisassemble) MesoCost() uint32           { return m.mesoCost }
func (m MakerResultDisassemble) Operation() string          { return MakerResultWriter }
func (m MakerResultDisassemble) String() string {
	return fmt.Sprintf("maker result disassemble mode [%d] result [%d] disassembledItemId [%d] crystals [%d] mesoCost [%d]", m.mode, m.result, m.disassembledItemId, len(m.crystals), m.mesoCost)
}

func (m MakerResultDisassemble) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.result)                // Decode4 nResult
		w.WriteInt(uint32(m.mode))          // Decode4 nMode
		w.WriteInt(m.disassembledItemId)    // Decode4 nDisassembledItemID
		w.WriteInt(uint32(len(m.crystals))) // Decode4 nNumRewardItem
		for _, c := range m.crystals {
			w.WriteInt(c.itemId)
			w.WriteInt(c.count)
		}
		w.WriteInt(m.mesoCost) // Decode4 nMesoCost
		return w.Bytes()
	}
}

func (m *MakerResultDisassemble) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.result = r.ReadUint32()
		m.mode = byte(r.ReadUint32())
		m.disassembledItemId = r.ReadUint32()
		count := r.ReadUint32()
		m.crystals = make([]MakerMaterial, 0, count)
		for i := uint32(0); i < count; i++ {
			id := r.ReadUint32()
			n := r.ReadUint32()
			m.crystals = append(m.crystals, NewMakerMaterial(id, n))
		}
		m.mesoCost = r.ReadUint32()
	}
}

// MakerResultFailed — the bodyless arm. When nResult ∉ {0, 1} the client's
// guard (gms_v95: Decode4 @0x910372-ish, `cmp eax,0 / jz` + `cmp eax,1 / jnz`)
// skips the mode read entirely and jumps to CUIItemMaker::OnItemMakeResult with
// zeroed arguments. So this arm writes nResult and NOTHING else — no mode field
// — which is why its constructor takes no mode. It is still a discrete struct
// with its own #-entry.
//
// packet-audit:fname CUserLocal::OnMakerResult#Failed
type MakerResultFailed struct {
	result uint32
}

func NewMakerResultFailed(result uint32) MakerResultFailed {
	return MakerResultFailed{result: result}
}

func (m MakerResultFailed) Result() uint32    { return m.result }
func (m MakerResultFailed) Operation() string { return MakerResultWriter }
func (m MakerResultFailed) String() string {
	return fmt.Sprintf("maker result failed result [%d]", m.result)
}

func (m MakerResultFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.result) // Decode4 nResult; body suppressed for nResult ∉ {0,1}
		return w.Bytes()
	}
}

func (m *MakerResultFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.result = r.ReadUint32()
	}
}
