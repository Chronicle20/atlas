package wish

// Model is the channel-side view of an atlas-mts wish-list entry, read over REST
// for the VIEW_WISH / DELETE_ZZIM / CANCEL_WISH ITC_OPERATION arms. The zzim/wish
// remove arms carry only a listing serial on the wire; the channel resolves the
// serial -> templateId and then finds the wish entry whose ItemId matches to get
// the Id (the wish UUID) needed to address a REMOVE_WISH command.
type Model struct {
	id          string
	characterId uint32
	itemId      uint32
}

func (m Model) Id() string          { return m.id }
func (m Model) CharacterId() uint32 { return m.characterId }
func (m Model) ItemId() uint32      { return m.itemId }
