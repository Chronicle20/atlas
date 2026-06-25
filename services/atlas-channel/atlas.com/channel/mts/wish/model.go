package wish

// Model is the channel-side view of an atlas-mts wish-list entry, read over REST
// for the VIEW_WISH / CANCEL_WISH ITC_OPERATION arms.
//
// Serial is the wish entry's per-(tenant, world) ITC serial (the client's
// nITCSN). VIEW_WISH (LoadWishSaleListDone) renders Serial into each wish
// ITCITEM's itcSn field; the client echoes it back verbatim on CANCEL_WISH
// (IDA: CITC::OnCancelWish, v83 0x59fb07, Encode4 of the item's nITCSN), so the
// channel resolves the CANCEL_WISH serial straight back to this wish entry (and
// its Id, the wish UUID, for the REMOVE_WISH command). DELETE_ZZIM operates on
// the favorites tab, which shows real LISTINGS (listing serials), so it keeps
// the listing-serial resolution path and does not use this model.
// Wish entry kinds (mirror atlas-mts wish.Type*): a Cart entry (added-to-cart,
// SET_ZZIM) vs a Wanted entry (a want-ad, REGISTER_WISH_ENTRY). Used to scope the
// Cart and Wanted MTS views to disjoint sets.
const (
	TypeCart   = "cart"
	TypeWanted = "wanted"
)

type Model struct {
	id          string
	worldId     byte
	serial      uint32
	characterId uint32
	itemId      uint32
	price       uint32
}

func (m Model) Id() string          { return m.id }
func (m Model) WorldId() byte        { return m.worldId }
func (m Model) Serial() uint32       { return m.serial }
func (m Model) CharacterId() uint32  { return m.characterId }
func (m Model) ItemId() uint32       { return m.itemId }
func (m Model) Price() uint32        { return m.price }
