package exclude

import "strconv"

type RestModel struct {
	Id     uint32 `json:"-"`
	ItemId uint32 `json:"itemId"`
}

func (r RestModel) GetName() string {
	return "excludes"
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
		Id:     m.Id(),
		ItemId: m.ItemId(),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return NewModelBuilder(rm.ItemId).
		SetId(rm.Id).
		Build()
}
