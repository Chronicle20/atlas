package kite

// Model is one kite (cash category 508 message box) as atlas-channel sees it.
// This replaces the previous model-only scaffold, whose `ft`/Type() field does
// not exist on the wire: FieldKiteSpawn's sixth int16 is the spawn Y, and the
// banner's appearance comes from templateId (CItemInfo::GetItemProp).
type Model struct {
	id          uint32
	characterId uint32
	name        string
	templateId  uint32
	message     string
	x           int16
	y           int16
}

func (m Model) Id() uint32          { return m.id }
func (m Model) CharacterId() uint32 { return m.characterId }
func (m Model) Name() string        { return m.name }
func (m Model) TemplateId() uint32  { return m.templateId }
func (m Model) Message() string     { return m.message }
func (m Model) X() int16            { return m.x }
func (m Model) Y() int16            { return m.y }
