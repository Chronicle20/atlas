package holding

import "github.com/Chronicle20/atlas/libs/atlas-constants/world"

// Model is the channel-side view of an atlas-mts take-home holding, read over
// REST for the ENTER_MTS holding announce (GET_USER_PURCHASE_ITEM_DONE). ItcSn is
// the holding's persistent per-(tenant, world) ITC serial (the client's nITCSN) —
// the channel emits it as MtsItem.itcSn so the client can address this holding in
// the take-home ITC_OPERATION arm.
type Model struct {
	id         string
	worldId    world.Id
	itcSn      uint32
	ownerId    uint32
	origin     string
	templateId uint32
	quantity   uint32
}

func (m Model) Id() string         { return m.id }
func (m Model) WorldId() world.Id  { return m.worldId }
func (m Model) ItcSn() uint32      { return m.itcSn }
func (m Model) OwnerId() uint32    { return m.ownerId }
func (m Model) Origin() string     { return m.origin }
func (m Model) TemplateId() uint32 { return m.templateId }
func (m Model) Quantity() uint32   { return m.quantity }
