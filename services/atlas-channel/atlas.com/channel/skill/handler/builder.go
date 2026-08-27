package handler

// PartyRecipientBuilder is the canonical constructor for PartyRecipient.
type PartyRecipientBuilder struct {
	r PartyRecipient
}

func NewPartyRecipientBuilder() *PartyRecipientBuilder { return &PartyRecipientBuilder{} }

func (b *PartyRecipientBuilder) SetId(v uint32) *PartyRecipientBuilder    { b.r.id = v; return b }
func (b *PartyRecipientBuilder) SetX(v int16) *PartyRecipientBuilder      { b.r.x = v; return b }
func (b *PartyRecipientBuilder) SetY(v int16) *PartyRecipientBuilder      { b.r.y = v; return b }
func (b *PartyRecipientBuilder) SetHp(v uint16) *PartyRecipientBuilder    { b.r.hp = v; return b }
func (b *PartyRecipientBuilder) SetMaxHp(v uint16) *PartyRecipientBuilder { b.r.maxHp = v; return b }
func (b *PartyRecipientBuilder) SetMp(v uint16) *PartyRecipientBuilder    { b.r.mp = v; return b }
func (b *PartyRecipientBuilder) SetMaxMp(v uint16) *PartyRecipientBuilder { b.r.maxMp = v; return b }

func (b *PartyRecipientBuilder) SetLevel(v byte) *PartyRecipientBuilder { b.r.level = v; return b }
func (b *PartyRecipientBuilder) Build() PartyRecipient                  { return b.r }
