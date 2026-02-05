package monster

import (
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
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
	ControlCharacterId uint32        `json:"controlCharacterId"`
	X                  int16         `json:"x"`
	Y                  int16         `json:"y"`
	Fh                 int16         `json:"fh"`
	Stance             byte          `json:"stance"`
	Team               int8          `json:"team"`
	MaxHp              uint32        `json:"maxHp"`
	Hp                 uint32        `json:"hp"`
	MaxMp              uint32        `json:"maxMp"`
	Mp                 uint32        `json:"mp"`
	DamageEntries      []DamageEntry `json:"damageEntries"`
}

type DamageEntry struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
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

func Extract(m RestModel) (Model, error) {
	id, err := strconv.Atoi(m.Id)
	if err != nil {
		return Model{}, err
	}

	return Model{
		uniqueId:           uint32(id),
		field:              field.NewBuilder(m.WorldId, m.ChannelId, m.MapId).SetInstance(m.Instance).Build(),
		maxHp:              m.MaxHp,
		hp:                 m.Hp,
		mp:                 m.Mp,
		monsterId:          m.MonsterId,
		controlCharacterId: m.ControlCharacterId,
		x:                  m.X,
		y:                  m.Y,
		fh:                 m.Fh,
		stance:             m.Stance,
		team:               m.Team,
	}, nil
}
