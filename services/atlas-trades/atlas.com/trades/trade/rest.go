package trade

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is the JSON:API projection of a live trade room. Resource type
// "rooms" (PRD §5). Read-only: rooms are driven exclusively by Kafka commands,
// so there is no input model and no SetToOne/SetToMany plumbing.
//
// Handle is deliberately not projected: it is the uint32 wire serial the
// client's invite carries, meaningful only inside the trade protocol, and the
// REST identity of a room is its uuid.
type RestModel struct {
	Id           string                 `json:"-"`
	RoomType     byte                   `json:"roomType"`
	WorldId      world.Id               `json:"worldId"`
	ChannelId    channel.Id             `json:"channelId"`
	MapId        _map.Id                `json:"mapId"`
	State        string                 `json:"state"`
	Participants []ParticipantRestModel `json:"participants"`
	CreatedAt    time.Time              `json:"createdAt"`
}

// ParticipantRestModel is one side of the room as PRD §5's participants array.
type ParticipantRestModel struct {
	CharacterId character.Id          `json:"characterId"`
	Position    byte                  `json:"position"`
	Confirmed   bool                  `json:"confirmed"`
	MesoStaged  uint32                `json:"mesoStaged"`
	Items       []StagedItemRestModel `json:"items"`
}

// StagedItemRestModel is one item claimed for trade. TradeSlot is the client
// dialog's 1..9 slot, NOT the inventory position the asset came from.
type StagedItemRestModel struct {
	TradeSlot byte           `json:"tradeSlot"`
	ItemId    item.Id        `json:"itemId"`
	Quantity  asset.Quantity `json:"quantity"`
	AssetId   asset.Id       `json:"assetId"`
}

func (r RestModel) GetID() string { return r.Id }

func (r *RestModel) SetID(id string) error { r.Id = id; return nil }

func (r RestModel) GetName() string { return "rooms" }

// Transform projects a Room into its REST shape.
func Transform(m Room) (RestModel, error) {
	ps := m.Participants()
	participants := make([]ParticipantRestModel, 0, len(ps))
	for _, p := range ps {
		is := p.Items()
		items := make([]StagedItemRestModel, 0, len(is))
		for _, i := range is {
			items = append(items, StagedItemRestModel{
				TradeSlot: i.TradeSlot(),
				ItemId:    i.TemplateId(),
				Quantity:  i.Quantity(),
				AssetId:   i.AssetId(),
			})
		}
		participants = append(participants, ParticipantRestModel{
			CharacterId: p.CharacterId(),
			Position:    p.Position(),
			Confirmed:   p.Confirmed(),
			MesoStaged:  p.MesoStaged(),
			Items:       items,
		})
	}
	return RestModel{
		Id:           m.Id().String(),
		RoomType:     m.RoomType(),
		WorldId:      m.Field().WorldId(),
		ChannelId:    m.Field().ChannelId(),
		MapId:        m.Field().MapId(),
		State:        string(m.State()),
		Participants: participants,
		CreatedAt:    m.CreatedAt(),
	}, nil
}
