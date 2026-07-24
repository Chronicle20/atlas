package teleport_rock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

const (
	ListTypeRegular = "regular"
	ListTypeVip     = "vip"
	RegularCapacity = 5
	VipCapacity     = 10
)

// ListType maps the wire-level VIP flag to the persisted list discriminator.
func ListType(vip bool) string {
	if vip {
		return ListTypeVip
	}
	return ListTypeRegular
}

// Capacity is enforced in the processor, not the schema (PRD §6).
func Capacity(vip bool) int {
	if vip {
		return VipCapacity
	}
	return RegularCapacity
}

// EligibleForRegistration is the client's numeric save rule (design §1 Q2):
// a map may be saved iff mapId/100000000 != 0 && (mapId/1000000)%100 != 9.
// This bars all sub-9-digit maps (Maple Island, Masteria, GM maps) and every
// x09xxxxxxx event block. It is NOT a fieldLimit check.
func EligibleForRegistration(mapId _map.Id) bool {
	return uint32(mapId)/100000000 != 0 && (uint32(mapId)/1000000)%100 != 9
}

// Model holds both saved-map lists for one character (unpadded, ordered).
type Model struct {
	characterId uint32
	regular     []_map.Id
	vip         []_map.Id
}

func (m Model) CharacterId() uint32 { return m.characterId }
func (m Model) Regular() []_map.Id  { return m.regular }
func (m Model) Vip() []_map.Id      { return m.vip }

func (m Model) List(vip bool) []_map.Id {
	if vip {
		return m.vip
	}
	return m.regular
}

func (m Model) Contains(vip bool, mapId _map.Id) bool {
	for _, v := range m.List(vip) {
		if v == mapId {
			return true
		}
	}
	return false
}
