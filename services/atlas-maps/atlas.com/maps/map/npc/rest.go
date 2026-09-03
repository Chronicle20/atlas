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

// TransformSlice maps a slice of domain Models to their REST projections.
// Returns the first transform error encountered, if any.
func TransformSlice(ms []Model) ([]RestModel, error) {
	out := make([]RestModel, 0, len(ms))
	for _, m := range ms {
		rm, err := Transform(m)
		if err != nil {
			return nil, err
		}
		out = append(out, rm)
	}
	return out, nil
}
