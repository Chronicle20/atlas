package craft

import (
	"atlas-maker/recipe"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// MaterialRestModel is one ordered (item, count) entry of a recipe's
// material list, mirrored from recipe.Material.
type MaterialRestModel struct {
	ItemId uint32 `json:"itemId"`
	Count  uint32 `json:"count"`
}

// RecipeRestModel is one recipe as returned by the two GET routes. Eligible
// and Reason carry the caller's per-character verdict computed by
// Processor.Evaluate; Reason is empty when Eligible is true.
type RecipeRestModel struct {
	Id            string              `json:"-"`
	ItemId        uint32              `json:"itemId"`
	ReqLevel      uint32              `json:"reqLevel"`
	ReqSkillLevel uint32              `json:"reqSkillLevel"`
	ItemNum       uint32              `json:"itemNum"`
	Tuc           uint32              `json:"tuc"`
	Meso          uint32              `json:"meso"`
	Catalyst      uint32              `json:"catalyst,omitempty"`
	ReqItem       uint32              `json:"reqItem,omitempty"`
	ReqEquip      uint32              `json:"reqEquip,omitempty"`
	Materials     []MaterialRestModel `json:"materials"`
	Eligible      bool                `json:"eligible"`
	Reason        string              `json:"reason,omitempty"`
}

func (r RecipeRestModel) GetName() string {
	return "makerRecipes"
}

func (r RecipeRestModel) GetID() string {
	return r.Id
}

func (r *RecipeRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// TransformRecipe builds a RecipeRestModel from r and its per-character
// eligibility verdict elig.
func TransformRecipe(r recipe.Model, elig Eligibility) RecipeRestModel {
	materials := make([]MaterialRestModel, 0, len(r.Materials()))
	for _, m := range r.Materials() {
		materials = append(materials, MaterialRestModel{ItemId: uint32(m.ItemId), Count: m.Count})
	}
	return RecipeRestModel{
		Id:            strconv.Itoa(int(r.Id())),
		ItemId:        uint32(r.Id()),
		ReqLevel:      r.ReqLevel(),
		ReqSkillLevel: r.ReqSkillLevel(),
		ItemNum:       r.ItemNum(),
		Tuc:           r.Tuc(),
		Meso:          r.Meso(),
		Catalyst:      uint32(r.Catalyst()),
		ReqItem:       uint32(r.ReqItem()),
		ReqEquip:      uint32(r.ReqEquip()),
		Materials:     materials,
		Eligible:      elig.Eligible,
		Reason:        string(elig.Reason),
	}
}

// CraftRequestRestModel is the incoming POST /crafts body. WorldId and
// ChannelId ride along because atlas-channel's handler knows its own
// session's world and channel and Request needs both scoped for
// AwardMesosPayload (see craft.Request's own doc); Task 24 is what
// populates them onto the Request it hands the Processor.
type CraftRequestRestModel struct {
	Id             string   `json:"-"`
	Mode           uint32   `json:"mode"`
	WorldId        byte     `json:"worldId"`
	ChannelId      byte     `json:"channelId"`
	TargetItemId   uint32   `json:"targetItemId,omitempty"`
	UseCatalyst    bool     `json:"useCatalyst,omitempty"`
	GemItemIds     []uint32 `json:"gemItemIds,omitempty"`
	LeftoverItemId uint32   `json:"leftoverItemId,omitempty"`
	EquipItemId    uint32   `json:"equipItemId,omitempty"`
	SlotPos        int16    `json:"slotPos,omitempty"`
}

func (r CraftRequestRestModel) GetName() string {
	return "makerCrafts"
}

func (r CraftRequestRestModel) GetID() string {
	return r.Id
}

func (r *CraftRequestRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// ToRequest builds the craft.Request Processor.Create consumes.
func (r CraftRequestRestModel) ToRequest() Request {
	gems := make([]item.Id, 0, len(r.GemItemIds))
	for _, g := range r.GemItemIds {
		gems = append(gems, item.Id(g))
	}
	return Request{
		Mode:           Mode(r.Mode),
		WorldId:        world.Id(r.WorldId),
		ChannelId:      channel.Id(r.ChannelId),
		TargetItemId:   item.Id(r.TargetItemId),
		UseCatalyst:    r.UseCatalyst,
		GemItemIds:     gems,
		LeftoverItemId: item.Id(r.LeftoverItemId),
		EquipItemId:    item.Id(r.EquipItemId),
		SlotPos:        r.SlotPos,
	}
}

// CraftResponseRestModel is POST /crafts's success body: the emitted saga's
// transaction id.
type CraftResponseRestModel struct {
	Id            string `json:"-"`
	TransactionId string `json:"transactionId"`
}

func (r CraftResponseRestModel) GetName() string {
	return "makerCrafts"
}

func (r CraftResponseRestModel) GetID() string {
	return r.Id
}

func (r *CraftResponseRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}
