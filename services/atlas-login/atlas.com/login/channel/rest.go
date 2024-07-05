package channel

import "strconv"

type RestModel struct {
	Id        uint32 `json:"-"`
	IpAddress string `json:"ipAddress"`
	Port      uint16 `json:"port"`
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

func Extract(m RestModel) (Model, error) {
	return Model{
		id:        byte(m.Id),
		capacity:  0,
		ipAddress: m.IpAddress,
		port:      m.Port,
	}, nil
}
