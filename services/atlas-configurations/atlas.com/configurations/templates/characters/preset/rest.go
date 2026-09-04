package preset

type StatBlock struct {
	Str uint16 `json:"str"`
	Dex uint16 `json:"dex"`
	Int uint16 `json:"int"`
	Luk uint16 `json:"luk"`
	Hp  uint16 `json:"hp"`
	Mp  uint16 `json:"mp"`
}

type EquipmentEntry struct {
	TemplateId      uint32 `json:"templateId"`
	UseAverageStats bool   `json:"useAverageStats"`
}

type InventoryEntry struct {
	TemplateId uint32 `json:"templateId"`
	Quantity   uint32 `json:"quantity"`
}

type SkillEntry struct {
	SkillId uint32 `json:"skillId"`
	Level   uint8  `json:"level"`
}

type Attributes struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	JobId       uint32    `json:"jobId"`
	Gender      byte      `json:"gender"`
	Face        uint32    `json:"face"`
	Hair        uint32    `json:"hair"`
	HairColor   uint32    `json:"hairColor"`
	SkinColor   byte      `json:"skinColor"`
	MapId       uint32    `json:"mapId"`
	Level       byte      `json:"level"`
	Meso        uint32    `json:"meso"`
	Gm          int       `json:"gm"`
	Stats       StatBlock `json:"stats"`
	// AP and SP are the unspent ability/skill points the created character
	// should start with. They exist here solely so this model's JSON key set
	// matches tenants/characters/preset.Attributes: the drift comparison
	// (task-289) hashes the marshaled document, and a key present on one
	// side only would report permanent `characters` drift for every tenant.
	// Zero (the marshaled default) reproduces prior behaviour for every
	// existing producer -- neither the admin preset UI nor any shipped seed
	// file sets these today.
	AP          uint16           `json:"ap"`
	SP          string           `json:"sp"`
	DefaultName string           `json:"defaultName"`
	Equipment   []EquipmentEntry `json:"equipment"`
	Inventory   []InventoryEntry `json:"inventory"`
	Skills      []SkillEntry     `json:"skills"`
}

type RestModel struct {
	Id         string     `json:"id"`
	Attributes Attributes `json:"attributes"`
}
