package commodities

// JSONModel is the JSON representation for loading commodity seed data
type JSONModel struct {
	TemplateId      uint32 `json:"templateId"`
	MesoPrice       uint32 `json:"mesoPrice"`
	DiscountRate    byte   `json:"discountRate"`
	TokenTemplateId uint32 `json:"tokenTemplateId"`
	TokenPrice      uint32 `json:"tokenPrice"`
	Period          uint32 `json:"period"`
	LevelLimit      uint32 `json:"levelLimit"`
}
