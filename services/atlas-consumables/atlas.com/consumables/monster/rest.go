package monster

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type RestModel struct {
	Id                 string        `json:"-"`
	WorldId            world.Id      `json:"worldId"`
	ChannelId          channel.Id    `json:"channelId"`
	MapId              _map.Id       `json:"mapId"`
	Instance           uuid.UUID     `json:"instance"`
	MonsterId          uint32        `json:"monsterId"`
	ControlCharacterId int           `json:"controlCharacterId"`
	X                  int16         `json:"x"`
	Y                  int16         `json:"y"`
	Fh                 uint16        `json:"fh"`
	Stance             int           `json:"stance"`
	Team               int32         `json:"team"`
	Hp                 int           `json:"hp"`
	DamageEntries      []damageEntry `json:"damageEntries"`
}

type damageEntry struct {
	CharacterId int   `json:"characterId"`
	Damage      int64 `json:"damage"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func (m RestModel) GetName() string {
	return "monsters"
}
