package key

import "strconv"

type RestModel struct {
	Key    int32 `json:"-"`
	Type   int8  `json:"type"`
	Action int32 `json:"action"`
}

func (r RestModel) GetName() string {
	return "keys"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Key))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Key = int32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Key:    m.key,
		Type:   m.theType,
		Action: m.action,
	}, nil
}
