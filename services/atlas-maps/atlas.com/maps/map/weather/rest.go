package weather

import "strconv"

type RestModel struct {
	Id      string `json:"-"`
	ItemId  uint32 `json:"itemId"`
	Message string `json:"message"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "weather"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func Transform(e WeatherEntry) (RestModel, error) {
	return RestModel{
		Id:      strconv.Itoa(int(e.ItemId)),
		ItemId:  e.ItemId,
		Message: e.Message,
	}, nil
}
