package cash

import (
	"strconv"
)

type RestModel struct {
	Id              uint32 `json:"-"`
	StateChangeItem uint32 `json:"stateChangeItem"`
	BgmPath         string `json:"bgmPath"`
	ProtectTime     uint32 `json:"protectTime"`
	// AddTime is info/addTime in SECONDS — the expiration grant of an
	// item-expiration extender (Magical Sandglass, classification 550). The
	// client multiplies it by 10^7 into FILETIME 100ns units
	// (CDraggableItem::ModifyEquipItem, gms_v83 @0x4F4BB7), which is what
	// fixes the unit as seconds. Mirrors atlas-data's cash resource.
	AddTime uint32 `json:"addTime,omitempty"`
	// MaxDays is info/maxDays in DAYS — the ceiling, anchored to now, past
	// which an extender may not push a target's expiration. Mirrors
	// atlas-data's cash resource.
	MaxDays uint32 `json:"maxDays,omitempty"`
	// Meso is the 0520 meso-sack award amount (atlas-data info/meso). Absent
	// or 0 means "no payout" and the type-19 handler rejects the use.
	Meso uint32 `json:"meso"`
	// Npc is the WZ info/npc value served by atlas-data: the NPC template a
	// remote-merchant cash item (classification 545) opens. 0 when none.
	Npc uint32 `json:"npc"`
	// Life is info/life in DAYS served by atlas-data: the lifespan a Water of
	// Life (classification 518) grants a revived pet. 0 or absent means the WZ
	// node is missing, which the handler treats as "reject, consume nothing".
	// The channel reads it ONLY as a pre-flight check -- the authoritative
	// derivation happens in atlas-pets and is re-bounded in atlas-inventory.
	Life uint32 `json:"life,omitempty"`
}

func (r RestModel) GetName() string {
	return "cash_items"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
