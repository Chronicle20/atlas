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

func Transform(model Model) RestModel {
	rm := RestModel{
		Id:              model.id,
		ItemId:          model.itemId,
		MinimumQuantity: model.minimumQuantity,
		MaximumQuantity: model.maximumQuantity,
		Chance:          model.chance,
	}
	return rm
}
