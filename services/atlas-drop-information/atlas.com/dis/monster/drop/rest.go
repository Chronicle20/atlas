package drop

import "strconv"

type RestModel struct {
	Id              uint32 `json:"-"`
	ItemId          uint32 `json:"item_id"`
	MinimumQuantity uint32 `json:"minimum_quantity"`
	MaximumQuantity uint32 `json:"maximum_quantity"`
	QuestId         uint32 `json:"quest_id"`
	Chance          uint32 `json:"chance"`
}

func (r RestModel) GetName() string {
	return "drops"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(model Model) (RestModel, error) {
	rm := RestModel{
		Id:              model.Id(),
		ItemId:          model.ItemId(),
		MinimumQuantity: model.MinimumQuantity(),
		MaximumQuantity: model.MaximumQuantity(),
		QuestId:         model.QuestId(),
		Chance:          model.Chance(),
	}
	return rm, nil
}
