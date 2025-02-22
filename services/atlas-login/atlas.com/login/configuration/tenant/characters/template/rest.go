package template

type RestModel struct {
	JobIndex    uint32   `json:"jobIndex"`
	SubJobIndex uint32   `json:"subJobIndex"`
	MapId       uint32   `json:"mapId"`
	Gender      byte     `json:"gender"`
	Faces       []uint32 `json:"faces"`
	Hairs       []uint32 `json:"hairs"`
	HairColors  []uint32 `json:"hairColors"`
	SkinColors  []uint32 `json:"skinColors"`
	Tops        []uint32 `json:"tops"`
	Bottoms     []uint32 `json:"bottoms"`
	Shoes       []uint32 `json:"shoes"`
	Weapons     []uint32 `json:"weapons"`
	Items       []uint32 `json:"items"`
	Skills      []uint32 `json:"skills"`
}
