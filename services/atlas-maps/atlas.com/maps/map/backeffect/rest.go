package backeffect

import "strconv"

type RestModel struct {
	Id       string `json:"-"`
	Effect   uint8  `json:"effect"`
	FieldId  uint32 `json:"fieldId"`
	PageId   uint8  `json:"pageId"`
	Duration uint32 `json:"duration"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "backEffect"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func Transform(e BackEffectEntry) (RestModel, error) {
	return RestModel{
		Id:       strconv.Itoa(int(e.PageId)),
		Effect:   uint8(e.Effect),
		FieldId:  e.FieldId,
		PageId:   uint8(e.PageId),
		Duration: e.Duration,
	}, nil
}
