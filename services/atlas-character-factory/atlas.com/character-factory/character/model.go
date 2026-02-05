package character

import "github.com/Chronicle20/atlas-constants/world"

type Model struct {
	id                 uint32
	accountId          uint32
	worldId            world.Id
	name               string
	level              byte
	experience         uint32
	gachaponExperience uint32
	strength           uint16
	dexterity          uint16
	intelligence       uint16
	luck               uint16
	hp                 uint16
	mp                 uint16
	maxHp              uint16
	maxMp              uint16
	meso               uint32
	hpMpUsed           int
	jobId              uint16
	skinColor          byte
	gender             byte
	fame               int16
	hair               uint32
	face               uint32
	ap                 uint16
	sp                 string
	mapId              uint32
	spawnPoint         uint32
	gm                 int
}

func (m Model) Id() uint32 {
	return m.id
}

type ItemGained struct {
	ItemId uint32
	Slot   int16
}
