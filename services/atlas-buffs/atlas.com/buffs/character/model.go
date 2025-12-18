package character

import (
	"atlas-buffs/buff"
	"github.com/Chronicle20/atlas-tenant"
)

type Model struct {
	tenant      tenant.Model
	worldId     byte
	characterId uint32
	buffs       map[int32]buff.Model
}

func (m Model) Buffs() map[int32]buff.Model {
	return m.buffs
}

func (m Model) Id() uint32 {
	return m.characterId
}

func (m Model) WorldId() byte {
	return m.worldId
}
