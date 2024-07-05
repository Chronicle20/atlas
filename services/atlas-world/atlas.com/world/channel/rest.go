package channel

import "strconv"

type RestModel struct {
	Id        uint32 `json:"-"`
	IpAddress string `json:"ipAddress"`
	Port      int    `json:"port"`
}

func (r RestModel) GetName() string {
	return "channels"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:        uint32(m.channelId),
		IpAddress: m.ipAddress,
		Port:      m.port,
	}, nil
}
