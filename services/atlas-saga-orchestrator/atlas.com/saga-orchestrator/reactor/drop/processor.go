package drop

import (
	"atlas-saga-orchestrator/kafka/producer"
	"context"
	"math/rand"
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DropType constants for drop behavior
const (
	DropTypeImmediate byte = 2 // Standard drop (same as player drops)
	DropTypeSpray     byte = 1 // Standard drop (same as player drops)
)

// Processor provides reactor drop spawning functionality
type Processor interface {
	// SpawnReactorDrops fetches reactor drop configuration, rolls chances, and spawns drops
	SpawnReactorDrops(
		transactionId uuid.UUID,
		characterId uint32,
		worldId world.Id,
		channelId channel.Id,
		mapId uint32,
		reactorId uint32,
		classification string,
		x int16,
		y int16,
		dropType string,
		mesoEnabled bool,
		mesoChance uint32,
		mesoMin uint32,
		mesoMax uint32,
		minItems uint32,
	) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) SpawnReactorDrops(
	transactionId uuid.UUID,
	characterId uint32,
	worldId world.Id,
	channelId channel.Id,
	mapId uint32,
	reactorId uint32,
	classification string,
	x int16,
	y int16,
	dropType string,
	mesoEnabled bool,
	mesoChance uint32,
	mesoMin uint32,
	mesoMax uint32,
	minItems uint32,
) error {
	// Fetch reactor drops from atlas-drop-information using classification
	drops, err := p.fetchReactorDrops(classification)
	if err != nil {
		p.l.WithError(err).Warnf("Failed to fetch drops for reactor [%d], proceeding with empty drops", reactorId)
		drops = []Model{}
	}

	// Roll chances to determine which items to drop
	itemsToDrop := p.rollDrops(drops)

	p.l.Debugf("Reactor [%d] dropped %d items out of %d possible", reactorId, len(itemsToDrop), len(drops))

	// Calculate meso padding if minItems is set
	mesoDropCount := 0
	if minItems > 0 && uint32(len(itemsToDrop)) < minItems {
		mesoDropCount = int(minItems) - len(itemsToDrop)
	} else if mesoEnabled && p.rollMesoChance(mesoChance) {
		// If meso is enabled and we didn't need padding, roll for a single meso drop
		mesoDropCount = 1
	}

	// Determine drop behavior
	isSpray := dropType == "spray"
	dropTypeByte := DropTypeImmediate
	if isSpray {
		dropTypeByte = DropTypeSpray
	}

	// Spawn item drops
	dropIndex := 0
	for _, item := range itemsToDrop {
		mod := p.calculateMod(isSpray, dropIndex)
		dropX := p.calculateDropX(x, dropIndex)

		// Calculate proper drop position using foothold data
		finalX, finalY := p.calculateDropPosition(mapId, dropX, y, x, y)

		err := p.spawnItemDrop(
			transactionId,
			worldId,
			channelId,
			mapId,
			item.ItemId(),
			1, // quantity is always 1 for reactor drops
			dropTypeByte,
			finalX,
			finalY,
			characterId,
			0, // ownerPartyId - reactor drops have no party
			reactorId,
			x,
			y,
			mod,
		)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to spawn item drop [%d] from reactor [%d]", item.ItemId(), reactorId)
			// Continue with other drops even if one fails
		}
		dropIndex++
	}

	// Spawn meso drops for padding
	for i := 0; i < mesoDropCount; i++ {
		mod := p.calculateMod(isSpray, dropIndex)
		dropX := p.calculateDropX(x, dropIndex)
		mesoAmount := p.randomMesoAmount(mesoMin, mesoMax)

		// Calculate proper drop position using foothold data
		finalX, finalY := p.calculateDropPosition(mapId, dropX, y, x, y)

		err := p.spawnMesoDrop(
			transactionId,
			worldId,
			channelId,
			mapId,
			mesoAmount,
			dropTypeByte,
			finalX,
			finalY,
			characterId,
			0, // ownerPartyId
			reactorId,
			x,
			y,
			mod,
		)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to spawn meso drop from reactor [%d]", reactorId)
		}
		dropIndex++
	}

	p.l.Debugf("Successfully spawned %d drops from reactor [%d] at (%d, %d)", dropIndex, reactorId, x, y)
	return nil
}

// fetchReactorDrops retrieves drop configuration from atlas-drop-information
func (p *ProcessorImpl) fetchReactorDrops(classification string) ([]Model, error) {
	classificationId, err := strconv.ParseUint(classification, 10, 32)
	if err != nil {
		return nil, err
	}

	reactor, err := requestReactorDrops(uint32(classificationId))(p.l, p.ctx)
	if err != nil {
		return nil, err
	}

	drops := make([]Model, len(reactor.Drops))
	for i, d := range reactor.Drops {
		drops[i] = Extract(d)
	}
	return drops, nil
}

// rollDrops determines which items drop based on their chance values
// Drop probability formula: dropRate / chance
// - chance=1 → 100%, chance=2 → 50%, chance=100 → 1%
// - Higher chance values = rarer drops
// - dropRate is a player multiplier (TODO: implement player drop rate bonuses)
func (p *ProcessorImpl) rollDrops(drops []Model) []Model {
	var result []Model
	dropRate := 1.0 // TODO: Get from player data when drop rate bonuses are implemented
	for _, d := range drops {
		if d.Chance() == 0 {
			continue // Skip invalid chance values
		}
		probability := dropRate / float64(d.Chance())
		if rand.Float64() < probability {
			result = append(result, d)
		}
	}
	return result
}

// rollMesoChance determines if meso should drop
// Same formula as item drops: probability = dropRate / mesoChance
// - mesoChance=1 → 100%, mesoChance=2 → 50%, mesoChance=100 → 1%
func (p *ProcessorImpl) rollMesoChance(mesoChance uint32) bool {
	if mesoChance == 0 {
		return false // No meso drop if chance is 0
	}
	dropRate := 1.0 // TODO: Get from player data when drop rate bonuses are implemented
	probability := dropRate / float64(mesoChance)
	return rand.Float64() < probability
}

// randomMesoAmount generates a random meso amount between min and max
func (p *ProcessorImpl) randomMesoAmount(min, max uint32) uint32 {
	if min >= max {
		return min
	}
	return min + rand.Uint32()%(max-min+1)
}

// calculateMod calculates the Mod value for spray timing
// Each increment of mod adds 200ms delay before the drop appears
func (p *ProcessorImpl) calculateMod(isSpray bool, index int) byte {
	if !isSpray {
		return 0
	}
	return byte(index)
}

// calculateDropX calculates the X position for spray positioning
// Drops alternate left and right from center with factor of 25 pixels
func (p *ProcessorImpl) calculateDropX(centerX int16, index int) int16 {
	if index == 0 {
		return centerX
	}
	// Alternating pattern: +25, -25, +50, -50, +75, -75, etc.
	offset := int16(((index + 1) / 2) * 25)
	if index%2 == 1 {
		return centerX + offset
	}
	return centerX - offset
}

// calculateDropPosition calculates the proper drop position using foothold data from the data service
func (p *ProcessorImpl) calculateDropPosition(mapId uint32, initialX, initialY, fallbackX, fallbackY int16) (int16, int16) {
	pos, err := requestDropPosition(mapId, initialX, initialY, fallbackX, fallbackY)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Warnf("Failed to calculate drop position for map [%d], using fallback (%d, %d)", mapId, fallbackX, fallbackY)
		return fallbackX, fallbackY
	}
	return pos.X, pos.Y
}

// spawnItemDrop sends a Kafka command to spawn an item drop
func (p *ProcessorImpl) spawnItemDrop(
	transactionId uuid.UUID,
	worldId world.Id,
	channelId channel.Id,
	mapId uint32,
	itemId uint32,
	quantity uint32,
	dropType byte,
	x int16,
	y int16,
	ownerId uint32,
	ownerPartyId uint32,
	dropperId uint32,
	dropperX int16,
	dropperY int16,
	mod byte,
) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(
		SpawnDropCommandProvider(
			transactionId,
			worldId,
			channelId,
			_map.Id(mapId),
			itemId,
			quantity,
			0, // mesos = 0 for item drops
			dropType,
			x,
			y,
			ownerId,
			ownerPartyId,
			dropperId,
			dropperX,
			dropperY,
			false, // playerDrop = false for reactor drops
			mod,
		),
	)
}

// spawnMesoDrop sends a Kafka command to spawn a meso drop
func (p *ProcessorImpl) spawnMesoDrop(
	transactionId uuid.UUID,
	worldId world.Id,
	channelId channel.Id,
	mapId uint32,
	mesos uint32,
	dropType byte,
	x int16,
	y int16,
	ownerId uint32,
	ownerPartyId uint32,
	dropperId uint32,
	dropperX int16,
	dropperY int16,
	mod byte,
) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(
		SpawnDropCommandProvider(
			transactionId,
			worldId,
			channelId,
			_map.Id(mapId),
			0, // itemId = 0 for meso drops
			0, // quantity = 0 for meso drops
			mesos,
			dropType,
			x,
			y,
			ownerId,
			ownerPartyId,
			dropperId,
			dropperX,
			dropperY,
			false, // playerDrop = false for reactor drops
			mod,
		),
	)
}
