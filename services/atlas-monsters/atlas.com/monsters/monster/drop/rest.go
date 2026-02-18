package drop

import "strconv"

type RestModel struct {
	Id              uint32 `json:"-"`
	ItemId          uint32 `json:"itemId"`
	MinimumQuantity uint32 `json:"minimumQuantity"`
	MaximumQuantity uint32 `json:"maximumQuantity"`
	QuestId         uint32 `json:"questId"`
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
		questId:         rm.QuestId,
		chance:          rm.Chance,
	}, nil
}
