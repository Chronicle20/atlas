package drop

type JSONModel struct {
	ContinentId     int32  `json:"continentId"`
	ItemId          uint32 `json:"itemId"`
	MinimumQuantity uint32 `json:"minimumQuantity"`
	MaximumQuantity uint32 `json:"maximumQuantity"`
	QuestId         uint32 `json:"questId"`
	Chance          uint32 `json:"chance"`
}
