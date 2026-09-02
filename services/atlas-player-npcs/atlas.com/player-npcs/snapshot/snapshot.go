// Package snapshot captures the appearance/equipment/rank inputs a Player
// NPC deploy freezes at deploy time (design §6.1). Capture fans out to the
// Task 13 read clients (character, inventory, ranking) and assembles the
// frozen snapshot; it does not decide eligibility (see eligibility/) or
// resolve a script id, position, or object id (Task 15/16's job).
package snapshot

import (
	"atlas-player-npcs/character"
	"atlas-player-npcs/inventory"
	"atlas-player-npcs/ranking"
	"atlas-player-npcs/routing"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// EquipmentRow is one frozen equipment slot on the snapshot (design §3.1,
// FR-5.2): Slot is the signed deploy-snapshot slot, 1-11 for a visible
// equip or 101-111 for a real equip masked by a cash equip in the same
// visible slot.
type EquipmentRow struct {
	slot   int16
	itemId uint32
}

func (r EquipmentRow) Slot() int16    { return r.slot }
func (r EquipmentRow) ItemId() uint32 { return r.itemId }

// Model is the frozen appearance/equipment/rank snapshot captured at
// deploy (design §6.1, §6.3).
type Model struct {
	gender      byte
	skinColor   byte
	face        uint32
	hair        uint32
	jobId       job.Id
	equipment   []EquipmentRow
	worldRank   uint32
	overallRank uint32
}

func (m Model) Gender() byte              { return m.gender }
func (m Model) SkinColor() byte           { return m.skinColor }
func (m Model) Face() uint32              { return m.face }
func (m Model) Hair() uint32              { return m.hair }
func (m Model) JobId() job.Id             { return m.jobId }
func (m Model) Equipment() []EquipmentRow { return m.equipment }
func (m Model) WorldRank() uint32         { return m.worldRank }
func (m Model) OverallRank() uint32       { return m.overallRank }

// equipmentPositions is the FR-5.2 boundary: the classic 11 equip-body
// slots (visible 1-11), derived from libs/atlas-constants/inventory/slot
// rather than re-declared here. Everything else (rings, belt, pendant,
// medal, shoulder, pet equipment, ...) is dropped at this boundary so no
// out-of-range slot ever reaches the codec.
var equipmentPositions = func() map[slot.Position]struct{} {
	m := make(map[slot.Position]struct{})
	for _, s := range slot.Slots {
		if s.Position >= -11 && s.Position <= -1 {
			m[s.Position] = struct{}{}
		}
	}
	return m
}()

// Capture fetches a character's appearance, equipped items and current
// ranks and assembles the frozen deploy snapshot. Equipment comes from
// atlas-inventory's equip compartment, not atlas-character (design §6.1
// correction; see the Task 13/14 fix round brief). jobId is stored as the
// job *category* (design §6.1, `(jobId/100)*100`), reusing
// routing.JobCategory rather than re-deriving the same arithmetic. A
// character with no computed ranking yields WorldRank()/OverallRank() of
// 0 with no error -- ranking.Processor already turns a 404 into the
// zero-value Model (design §6.3).
func Capture(characterId uint32, worldId world.Id, cp character.Processor, ip inventory.Processor, rp ranking.Processor) (Model, error) {
	c, err := cp.GetById(characterId)
	if err != nil {
		return Model{}, err
	}

	inv, err := ip.GetByCharacterId(characterId)
	if err != nil {
		return Model{}, err
	}

	r, err := rp.GetByCharacterId(characterId, worldId)
	if err != nil {
		return Model{}, err
	}

	return Model{
		gender:      c.Gender(),
		skinColor:   c.SkinColor(),
		face:        c.Face(),
		hair:        c.Hair(),
		jobId:       job.Id(routing.JobCategory(c.JobId())),
		equipment:   captureEquipment(inv.Equipment()),
		worldRank:   r.Rank(),
		overallRank: r.Rank(),
	}, nil
}

// equipmentPair tracks the real and/or cash item occupying one body
// position, keyed by the position's normal (uncashed) slot number.
type equipmentPair struct {
	real *uint32
	cash *uint32
}

// captureEquipment applies the repo's raw-asset-slot cash convention
// (services/atlas-channel/atlas.com/channel/character/model.go: `cash :=
// s < -100; s += 100`) to the unfiltered equip-compartment assets
// atlas-inventory returns, then re-expresses each occupied body position
// as the deploy-snapshot's visible/masked slot pair (avatar.go:10-31's
// convention, restated for a flat signed-slot schema instead of two
// packet maps): the visible slot (Position*-1) prefers the cash item when
// one is present; the masked slot (Position*-1+100) holds the real item
// only when a cash item is masking it. Positions outside the classic
// 1-11 equip range are dropped (FR-5.2).
func captureEquipment(items []inventory.Asset) []EquipmentRow {
	byPosition := make(map[slot.Position]*equipmentPair)

	for _, item := range items {
		s := item.Slot()
		cash := s < -100
		normalized := s
		if cash {
			normalized += 100
		}

		pos := slot.Position(normalized)
		if _, ok := equipmentPositions[pos]; !ok {
			continue
		}

		p, ok := byPosition[pos]
		if !ok {
			p = &equipmentPair{}
			byPosition[pos] = p
		}
		tid := item.TemplateId()
		if cash {
			p.cash = &tid
		} else {
			p.real = &tid
		}
	}

	rows := make([]EquipmentRow, 0, len(byPosition)*2)
	for pos, p := range byPosition {
		visible := int16(pos) * -1
		switch {
		case p.cash != nil:
			rows = append(rows, EquipmentRow{slot: visible, itemId: *p.cash})
			if p.real != nil {
				rows = append(rows, EquipmentRow{slot: visible + 100, itemId: *p.real})
			}
		case p.real != nil:
			rows = append(rows, EquipmentRow{slot: visible, itemId: *p.real})
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].slot < rows[j].slot })
	return rows
}
