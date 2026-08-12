package ledger

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is the JSON:API projection of one settled trade. Resource type
// "ledgerEntries" (PRD §5). Read-only: ledger rows are immutable (FR-7.4), so
// there is no input model and no SetToOne/SetToMany plumbing.
type RestModel struct {
	Id            string          `json:"-"`
	TransactionId string          `json:"transactionId"`
	WorldId       world.Id        `json:"worldId"`
	ChannelId     channel.Id      `json:"channelId"`
	MapId         _map.Id         `json:"mapId"`
	RoomType      byte            `json:"roomType"`
	SettledAt     time.Time       `json:"settledAt"`
	Sides         []SideRestModel `json:"sides"`
}

// SideRestModel is one participant's recorded contribution. mesoStaged/mesoTax
// describe what this side gave; mesoDelivered what it received (FR-7.1).
//
// ORDERING: sides read back from the ledger are ordered by character id. That
// is a determinism guarantee, not a role guarantee — a consumer must match on
// characterId, never on position.
type SideRestModel struct {
	CharacterId   character.Id    `json:"characterId"`
	CharacterName string          `json:"characterName"`
	MesoStaged    uint32          `json:"mesoStaged"`
	MesoTax       uint32          `json:"mesoTax"`
	MesoDelivered uint32          `json:"mesoDelivered"`
	Items         []ItemRestModel `json:"items"`
}

// ItemRestModel is one asset a side gave. AssetId and ReferenceId are pointers
// so an asset with no identity of its own (a plain stackable) renders null
// rather than 0, which a consumer would read as a real id.
type ItemRestModel struct {
	ItemId      item.Id        `json:"itemId"`
	Quantity    asset.Quantity `json:"quantity"`
	AssetId     *asset.Id      `json:"assetId"`
	ReferenceId *uint32        `json:"referenceId"`
}

func (r RestModel) GetID() string { return r.Id }

func (r *RestModel) SetID(id string) error { r.Id = id; return nil }

func (r RestModel) GetName() string { return "ledgerEntries" }

// Transform projects a ledger Model into its REST shape.
func Transform(m Model) (RestModel, error) {
	ss := m.Sides()
	sides := make([]SideRestModel, 0, len(ss))
	for _, s := range ss {
		is := s.Items()
		items := make([]ItemRestModel, 0, len(is))
		for _, i := range is {
			ri := ItemRestModel{ItemId: i.ItemId(), Quantity: i.Quantity()}
			if assetId, ok := i.AssetId(); ok {
				ri.AssetId = &assetId
			}
			if referenceId, ok := i.ReferenceId(); ok {
				ri.ReferenceId = &referenceId
			}
			items = append(items, ri)
		}
		sides = append(sides, SideRestModel{
			CharacterId:   s.CharacterId(),
			CharacterName: s.CharacterName(),
			MesoStaged:    s.MesoStaged(),
			MesoTax:       s.MesoTax(),
			MesoDelivered: s.MesoDelivered(),
			Items:         items,
		})
	}
	return RestModel{
		Id:            m.Id().String(),
		TransactionId: m.TransactionId().String(),
		WorldId:       m.Field().WorldId(),
		ChannelId:     m.Field().ChannelId(),
		MapId:         m.Field().MapId(),
		RoomType:      m.RoomType(),
		SettledAt:     m.SettledAt(),
		Sides:         sides,
	}, nil
}
