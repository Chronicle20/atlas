package drop

type RestModel struct {
	ItemId          uint32 `json:"itemId"`
	MinimumQuantity uint32 `json:"minimumQuantity"`
	MaximumQuantity uint32 `json:"maximumQuantity"`
	QuestId         uint32 `json:"questId"`
	Chance          uint32 `json:"chance"`
}
