package escrow

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// ItemBuilder assembles an ItemModel. The three fields a row cannot be without
// — its own id, the room it backs and the owner it must be returned to — are
// constructor arguments rather than setters, because a row missing any of them
// is unreturnable and the reconciler would have nothing to act on.
//
// The stat block is all optional setters: most staged items are stackables with
// an entirely zero block.
type ItemBuilder struct {
	m ItemModel
}

func NewItemBuilder(id uuid.UUID, roomId uuid.UUID, ownerId character.Id) *ItemBuilder {
	return &ItemBuilder{m: ItemModel{id: id, roomId: roomId, ownerId: ownerId}}
}

func (b *ItemBuilder) SetTradeSlot(v byte) *ItemBuilder { b.m.tradeSlot = v; return b }

func (b *ItemBuilder) SetSource(inventoryType inventory.Type, sourceSlot slot.Position, assetId asset.Id) *ItemBuilder {
	b.m.sourceInventoryType = inventoryType
	b.m.sourceSlot = sourceSlot
	b.m.assetId = assetId
	return b
}

func (b *ItemBuilder) SetTemplateId(v item.Id) *ItemBuilder      { b.m.templateId = v; return b }
func (b *ItemBuilder) SetQuantity(v asset.Quantity) *ItemBuilder { b.m.quantity = v; return b }
func (b *ItemBuilder) SetStrength(v uint16) *ItemBuilder         { b.m.strength = v; return b }
func (b *ItemBuilder) SetDexterity(v uint16) *ItemBuilder        { b.m.dexterity = v; return b }
func (b *ItemBuilder) SetIntelligence(v uint16) *ItemBuilder     { b.m.intelligence = v; return b }
func (b *ItemBuilder) SetLuck(v uint16) *ItemBuilder             { b.m.luck = v; return b }
func (b *ItemBuilder) SetHP(v uint16) *ItemBuilder               { b.m.hp = v; return b }
func (b *ItemBuilder) SetMP(v uint16) *ItemBuilder               { b.m.mp = v; return b }
func (b *ItemBuilder) SetWeaponAttack(v uint16) *ItemBuilder     { b.m.weaponAttack = v; return b }
func (b *ItemBuilder) SetMagicAttack(v uint16) *ItemBuilder      { b.m.magicAttack = v; return b }
func (b *ItemBuilder) SetWeaponDefense(v uint16) *ItemBuilder    { b.m.weaponDefense = v; return b }
func (b *ItemBuilder) SetMagicDefense(v uint16) *ItemBuilder     { b.m.magicDefense = v; return b }
func (b *ItemBuilder) SetAccuracy(v uint16) *ItemBuilder         { b.m.accuracy = v; return b }
func (b *ItemBuilder) SetAvoidability(v uint16) *ItemBuilder     { b.m.avoidability = v; return b }
func (b *ItemBuilder) SetHands(v uint16) *ItemBuilder            { b.m.hands = v; return b }
func (b *ItemBuilder) SetSpeed(v uint16) *ItemBuilder            { b.m.speed = v; return b }
func (b *ItemBuilder) SetJump(v uint16) *ItemBuilder             { b.m.jump = v; return b }
func (b *ItemBuilder) SetSlots(v uint16) *ItemBuilder            { b.m.slots = v; return b }
func (b *ItemBuilder) SetLevel(v byte) *ItemBuilder              { b.m.level = v; return b }
func (b *ItemBuilder) SetItemLevel(v byte) *ItemBuilder          { b.m.itemLevel = v; return b }
func (b *ItemBuilder) SetItemExp(v uint32) *ItemBuilder          { b.m.itemExp = v; return b }
func (b *ItemBuilder) SetRingId(v uint32) *ItemBuilder           { b.m.ringId = v; return b }
func (b *ItemBuilder) SetViciousCount(v uint32) *ItemBuilder     { b.m.viciousCount = v; return b }
func (b *ItemBuilder) SetFlags(v uint16) *ItemBuilder            { b.m.flags = v; return b }
func (b *ItemBuilder) SetOwner(v string) *ItemBuilder            { b.m.owner = v; return b }

func (b *ItemBuilder) Build() ItemModel { return b.m }
