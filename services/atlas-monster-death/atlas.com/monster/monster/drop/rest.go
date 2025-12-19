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
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}

	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		itemId:          rm.ItemId,
		minimumQuantity: rm.MinimumQuantity,
		maximumQuantity: rm.MaximumQuantity,
		chance:          rm.Chance,
		questId:         rm.QuestId,
	}, nil
}
