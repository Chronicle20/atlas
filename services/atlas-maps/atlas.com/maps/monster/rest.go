package monster

type RestModel struct {
	Id                 string        `json:"-"`
	WorldId            byte          `json:"worldId"`
	ChannelId          byte          `json:"channelId"`
	MapId              int           `json:"mapId"`
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

func (m RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func (m RestModel) GetName() string {
	return "monsters"
}
