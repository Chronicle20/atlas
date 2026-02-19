package monster

import (
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

// SpawnInputRestModel represents the input for spawning a monster.
type SpawnInputRestModel struct {
	Id        string `json:"-"`
	MonsterId uint32 `json:"monsterId"`
	X         int16  `json:"x"`
	Y         int16  `json:"y"`
	Fh        int16  `json:"fh"`
	Team      int8   `json:"team"`
}

func (r SpawnInputRestModel) GetName() string {
	return "monsters"
}

func (r SpawnInputRestModel) GetID() string {
	return r.Id
}

// SpawnResponseRestModel represents the response from spawning a monster.
type SpawnResponseRestModel struct {
	Id                 string     `json:"-"`
	UniqueId           uint32     `json:"uniqueId"`
	WorldId            world.Id   `json:"worldId"`
	ChannelId          channel.Id `json:"channelId"`
	MapId              _map.Id    `json:"mapId"`
	MonsterId          uint32     `json:"monsterId"`
	ControlCharacterId uint32     `json:"controlCharacterId"`
	X                  int16      `json:"x"`
	Y                  int16      `json:"y"`
	Fh                 int16      `json:"fh"`
	Stance             byte       `json:"stance"`
	Team               int8       `json:"team"`
	MaxHp              uint32     `json:"maxHp"`
	Hp                 uint32     `json:"hp"`
	MaxMp              uint32     `json:"maxMp"`
	Mp                 uint32     `json:"mp"`
}

func (r SpawnResponseRestModel) GetName() string {
	return "monsters"
}

func (r SpawnResponseRestModel) GetID() string {
	return r.Id
}

func (r *SpawnResponseRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// SpawnRequest contains the parameters needed to spawn a monster.
type SpawnRequest struct {
	WorldId   world.Id
	ChannelId channel.Id
	MapId     _map.Id
	MonsterId uint32
	X         int16
	Y         int16
	Fh        int16
	Team      int8
}

func (r SpawnRequest) ToRestModel() SpawnInputRestModel {
	return SpawnInputRestModel{
		Id:        strconv.FormatUint(uint64(r.MonsterId), 10),
		MonsterId: r.MonsterId,
		X:         r.X,
		Y:         r.Y,
		Fh:        r.Fh,
		Team:      r.Team,
	}
}
