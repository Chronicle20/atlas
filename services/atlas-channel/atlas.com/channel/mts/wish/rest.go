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
