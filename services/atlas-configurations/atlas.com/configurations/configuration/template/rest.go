package template

type RestModel struct {
	JobIndex          uint32   `json:"jobIndex"`
	SubJobIndex       uint32   `json:"subJobIndex"`
	MapId             uint32   `json:"mapId"`
	Gender            byte     `json:"gender"`
	Face              []uint32 `json:"face"`
	Hair              []uint32 `json:"hair"`
	HairColor         []uint32 `json:"hairColor"`
	SkinColor         []uint32 `json:"skinColor"`
	Top               []uint32 `json:"top"`
	Bottom            []uint32 `json:"bottom"`
	Shoes             []uint32 `json:"shoes"`
	Weapon            []uint32 `json:"weapon"`
	StartingInventory []uint32 `json:"startingInventory"`
}
