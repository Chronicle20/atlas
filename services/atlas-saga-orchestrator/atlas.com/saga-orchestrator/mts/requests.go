package mts

import (
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	// holdingsResource lists a character's take-home holdings, optionally scoped
	// to a world. atlas-mts exposes GET /characters/{characterId}/mts/holding.
	holdingsResource = "characters/%d/mts/holding?worldId=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("MTS")
}

// HoldingRestModel is the orchestrator's view of an atlas-mts holding row,
// carrying the full item snapshot needed to re-grant the item to a character's
// inventory on WithdrawFromMts. Mirrors atlas-mts holding/rest.go RestModel.
type HoldingRestModel struct {
	Id      string `json:"-"`
	WorldId byte   `json:"worldId"`
	OwnerId uint32 `json:"ownerId"`
	Origin  string `json:"origin"`

	TemplateId uint32 `json:"templateId"`
	Quantity   uint32 `json:"quantity"`

	Strength      uint16 `json:"strength"`
	Dexterity     uint16 `json:"dexterity"`
	Intelligence  uint16 `json:"intelligence"`
	Luck          uint16 `json:"luck"`
	HP            uint16 `json:"hp"`
	MP            uint16 `json:"mp"`
	WeaponAttack  uint16 `json:"weaponAttack"`
	MagicAttack   uint16 `json:"magicAttack"`
	WeaponDefense uint16 `json:"weaponDefense"`
	MagicDefense  uint16 `json:"magicDefense"`
	Accuracy      uint16 `json:"accuracy"`
	Avoidability  uint16 `json:"avoidability"`
	Hands         uint16 `json:"hands"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
	Slots         uint16 `json:"slots"`
	Level         byte   `json:"level"`
	ItemLevel     byte   `json:"itemLevel"`
	ItemExp       uint32 `json:"itemExp"`
	RingId        uint32 `json:"ringId"`
	ViciousCount  uint32 `json:"viciousCount"`
	Flags         uint16 `json:"flags"`

	CreatedAt time.Time `json:"createdAt"`
}

func (r HoldingRestModel) GetName() string { return "holdings" }

func (r HoldingRestModel) GetID() string { return r.Id }

func (r *HoldingRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Required JSON:API relationship stubs (libs/atlas-rest gotcha): api2go errors
// out decoding any response unless the target implements these, even with no
// relationships present.
func (r *HoldingRestModel) SetToOneReferenceID(_, _ string) error { return nil }

func (r *HoldingRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// RequestHoldings retrieves a character's holdings (optionally world-scoped)
// from atlas-mts so expansion can capture the item snapshot for the
// accept_to_character step. Mirrors cashshop.RequestCompartment.
func RequestHoldings(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, worldId byte) ([]HoldingRestModel, error) {
	return func(characterId uint32, worldId byte) ([]HoldingRestModel, error) {
		url := fmt.Sprintf(getBaseRequest()+holdingsResource, characterId, worldId)
		return requests.GetRequest[[]HoldingRestModel](url)(l, ctx)
	}
}
