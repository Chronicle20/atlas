package maplelife

// LookOptions is one gender's set of selectable appearance values, mirroring
// the four per-gender lists CUICharacterSaleDlg::LoadNewCharInfo reads out of
// its WZ property (design.md §11 A3: the loop is
// `for (nType = 0; nType < 4; ++nType)` -- four option types, no class
// dimension and no equipment choice). Kept beside Classes rather than inside
// each class entry for exactly that reason.
type LookOptions struct {
	Gender     byte     `json:"gender"`
	Faces      []uint32 `json:"faces"`
	Hairs      []uint32 `json:"hairs"`
	HairColors []uint32 `json:"hairColors"`
	SkinColors []uint32 `json:"skinColors"`
}

// StatBlock is the AP already spent to meet the class's first-job requirement.
// Whatever the class accumulated by its configured level and did not spend
// here is carried separately as ClassEntry.AP.
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

// ClassEntry is one (class ordinal, gender) row of the Maple Life creation
// table. Ordinal is the client's own m_nCurrentClass, which cycles 0..4
// (gms_v95 CUICharacterSaleDlg::OnButtonClicked @0x77edc0). Gender splits the
// row because equipment differs between the two, exactly as
// characters.templates carries one row per (jobIndex, subJobIndex, gender).
//
// The ordinal->class mapping is DATA, not code (design.md §11 A6): 0 and 1 are
// derived (Warrior, Magician), 2/3/4 are not, so a wrong order is a seed-data
// fix rather than a code change.
type ClassEntry struct {
	Ordinal uint32 `json:"ordinal"`
	Gender  byte   `json:"gender"`
	JobId   uint32 `json:"jobId"`
	Level   byte   `json:"level"`
	MapId   uint32 `json:"mapId"`

	Stats StatBlock `json:"stats"`
	// AP and SP are what the character has left UNSPENT at Level
	// (design.md §11 A2). SP is the ten-slot pool in the form
	// atlas-character persists it.
	AP uint16 `json:"ap"`
	SP string `json:"sp"`
	// SpSkillId is the skill the dialog offers to pre-level at creation --
	// Improved Max HP Increase for ordinal 0, Max MP Increase for ordinal 1.
	// ABSENT (zero) means this class offers no SP step, which is what the
	// client encodes by skipping step 4 for m_nCurrentClass >= 2.
	SpSkillId uint32 `json:"spSkillId,omitempty"`

	Meso      uint32           `json:"meso"`
	Equipment []EquipmentEntry `json:"equipment"`
	Inventory []InventoryEntry `json:"inventory"`
}

// RestModel is the tenant's Maple Life configuration. A tenant whose client
// has no Maple Life dialog carries no block at all, which decodes to the zero
// value.
type RestModel struct {
	Looks   []LookOptions `json:"looks"`
	Classes []ClassEntry  `json:"classes"`
}
