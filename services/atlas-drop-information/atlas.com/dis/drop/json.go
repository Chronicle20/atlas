package drop

type JSONModel struct {
	MonsterId       uint32 `json:"monsterId"`
	ItemId          uint32 `json:"itemId"`
	MinimumQuantity uint32 `json:"minimumQuantity"`
	MaximumQuantity uint32 `json:"maximumQuantity"`
	Chance          uint32 `json:"chance"`
}
