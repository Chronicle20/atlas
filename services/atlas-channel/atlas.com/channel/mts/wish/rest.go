package wish

import "time"

// RestModel mirrors atlas-mts's wish.RestModel (the JSON:API "wish-entries"
// resource). Only Id, CharacterId, and ItemId are consumed channel-side; the
// To-One/To-Many relationship stubs are required boilerplate for the api2go
// unmarshal even though wish entries carry no relationships block (see
// libs/atlas-rest/CLAUDE.md).
type RestModel struct {
	Id            string     `json:"-"`
	WorldId       byte       `json:"worldId"`
	Serial        uint32     `json:"serial"`
	CharacterId   uint32     `json:"characterId"`
	ItemId        uint32     `json:"itemId"`
	ListingSerial uint32     `json:"listingSerial"`
	Price         uint32     `json:"price"`
	Count         uint32     `json:"count"`
	ExpiresAt     *time.Time `json:"expiresAt"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func (r RestModel) GetName() string { return "wish-entries" }
func (r RestModel) GetID() string   { return r.Id }

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// Required api2go relationship stubs (wish entries carry no relationships, but the
// unmarshal path walks the interfaces — see libs/atlas-rest/CLAUDE.md).
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// Transform converts a domain Model into its RestModel. ExpiresAt is a
// *time.Time on both sides; a nil source means the wish entry never expires,
// so nil propagates through unchanged. When non-nil, the pointed-to value is
// copied into a new pointer so the RestModel cannot mutate the Model's
// time.Time via a shared pointer. CreatedAt has no Model counterpart (the
// Model does not track it), so it is left at its zero value rather than
// invented.
func Transform(m Model) (RestModel, error) {
	var expiresAt *time.Time
	if m.expiresAt != nil {
		v := *m.expiresAt
		expiresAt = &v
	}

	return RestModel{
		Id:            m.id,
		WorldId:       m.worldId,
		Serial:        m.serial,
		CharacterId:   m.characterId,
		ItemId:        m.itemId,
		ListingSerial: m.listingSerial,
		Price:         m.price,
		Count:         m.count,
		ExpiresAt:     expiresAt,
	}, nil
}

func Extract(r RestModel) (Model, error) {
	return Model{
		id:            r.Id,
		worldId:       r.WorldId,
		serial:        r.Serial,
		characterId:   r.CharacterId,
		itemId:        r.ItemId,
		listingSerial: r.ListingSerial,
		price:         r.Price,
		count:         r.Count,
		expiresAt:     r.ExpiresAt,
	}, nil
}
