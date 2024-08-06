package monster

import "strconv"

type RestModel struct {
	Id                 string        `json:"-"`
	WorldId            byte          `json:"worldId"`
	ChannelId          byte          `json:"channelId"`
	MapId              uint32        `json:"mapId"`
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

func Extract(m RestModel) (Model, error) {
	id, err := strconv.Atoi(m.Id)
	if err != nil {
		return Model{}, err
	}

	return Model{
		uniqueId:           uint32(id),
		worldId:            m.WorldId,
		channelId:          m.ChannelId,
		mapId:              m.MapId,
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
