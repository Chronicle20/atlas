package inventory

type ItemListDataContainer struct {
	Data []ItemDataBody `json:"data"`
}

type ItemDataBody struct {
	Id         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes ItemAttributes `json:"attributes"`
}

type ItemAttributes struct {
	InventoryType string `json:"inventoryType"`
	Slot          int16  `json:"slot"`
	Quantity      uint32 `json:"quantity"`
}
