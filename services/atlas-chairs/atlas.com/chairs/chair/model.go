package chair

import "encoding/json"

type Model struct {
	id        uint32
	chairType string
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Type() string {
	return m.chairType
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Id        uint32 `json:"id"`
		ChairType string `json:"chairType"`
	}{
		Id:        m.id,
		ChairType: m.chairType,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	t := &struct {
		Id        uint32 `json:"id"`
		ChairType string `json:"chairType"`
	}{}
	if err := json.Unmarshal(data, t); err != nil {
		return err
	}
	m.id = t.Id
	m.chairType = t.ChairType
	return nil
}
