package npc

import "strconv"

type RestModel struct {
	Id            string `json:"-"`
	NpcId         uint32 `json:"npcId"`
	X             int16  `json:"x"`
	Y             int16  `json:"y"`
	Fh            int16  `json:"fh"`
	SpawnIfAbsent bool   `json:"spawnIfAbsent,omitempty"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "npcs"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:    strconv.Itoa(int(m.UniqueId())),
		NpcId: m.NpcId(),
		X:     m.X(),
		Y:     m.Y(),
		Fh:    m.Fh(),
	}, nil
}
