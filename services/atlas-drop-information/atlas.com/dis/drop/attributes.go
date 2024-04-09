package drop

type DataContainer struct {
	Data DataBody `json:"data"`
}

type DataListContainer struct {
	Data []DataBody `json:"data"`
}

type DataBody struct {
	Id         string     `json:"id"`
	Type       string     `json:"type"`
	Attributes Attributes `json:"attributes"`
}

type Attributes struct {
	MonsterId       uint32 `json:"monsterId"`
	ItemId          uint32 `json:"itemId"`
	MaximumQuantity uint32 `json:"maximumQuantity"`
	MinimumQuantity uint32 `json:"minimumQuantity"`
	Chance          uint32 `json:"chance"`
}
