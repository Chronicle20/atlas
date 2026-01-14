package reactor

import "strconv"

type RestModel struct {
	Id             uint32 `json:"-"`
	WorldId        byte   `json:"worldId"`
	ChannelId      byte   `json:"channelId"`
	MapId          uint32 `json:"mapId"`
	Classification uint32 `json:"classification"`
	Name           string `json:"name"`
	State          int8   `json:"state"`
	EventState     byte   `json:"eventState"`
	X              int16  `json:"x"`
	Y              int16  `json:"y"`
	Delay          uint32 `json:"delay"`
	Direction      byte   `json:"direction"`
}

func (r RestModel) GetName() string {
	return "reactors"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}

	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:             m.Id(),
		WorldId:        m.WorldId(),
		ChannelId:      m.ChannelId(),
		MapId:          m.MapId(),
		Classification: m.Classification(),
		Name:           m.Name(),
		State:          m.State(),
		EventState:     m.EventState(),
		X:              m.X(),
		Y:              m.Y(),
		Delay:          m.Delay(),
		Direction:      m.Direction(),
	}, nil
}
