package jukebox

type RestModel struct {
	Id         string `json:"-"`
	ItemId     uint32 `json:"itemId"`
	PlayerName string `json:"playerName"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "jukebox"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}
