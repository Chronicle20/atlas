package field

import (
	"encoding/json"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	worldId   world.Id
	channelId channel.Id
	mapId     _map.Id
	instance  uuid.UUID
}

func (m Model) Id() Id {
	return Id(fmt.Sprintf(IdFormat, m.worldId, m.channelId, m.mapId, m.instance))
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

func (m Model) Channel() channel.Model {
	return channel.NewModel(m.WorldId(), m.ChannelId())
}

func (m Model) MapId() _map.Id {
	return m.mapId
}

func (m Model) Instance() uuid.UUID {
	return m.instance
}

func (m Model) Clone() *Builder {
	return NewBuilder(m.worldId, m.channelId, m.mapId).SetInstance(m.instance)
}

// DataTransferObject is a serializable representation of a field.Model.
// It can be embedded in Kafka message bodies or used in REST API responses.
type DataTransferObject struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

// ToDTO converts the Model to a DataTransferObject for embedding in messages.
func (m Model) ToDTO() DataTransferObject {
	return DataTransferObject{
		WorldId:   m.worldId,
		ChannelId: m.channelId,
		MapId:     m.mapId,
		Instance:  m.instance,
	}
}

// FromDTO creates a Model from a DataTransferObject.
func FromDTO(dto DataTransferObject) Model {
	return Model{
		worldId:   dto.WorldId,
		channelId: dto.ChannelId,
		mapId:     dto.MapId,
		instance:  dto.Instance,
	}
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.ToDTO())
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var dto DataTransferObject
	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}
	*m = FromDTO(dto)
	return nil
}

// Equals returns true if both models represent the exact same field location,
// including the instance UUID.
func (m Model) Equals(other Model) bool {
	return m.worldId == other.worldId &&
		m.channelId == other.channelId &&
		m.mapId == other.mapId &&
		m.instance == other.instance
}

// SameMap returns true if both models are on the same map (world, channel, mapId)
// regardless of instance. Useful for checking if two characters are on the same map type.
func (m Model) SameMap(other Model) bool {
	return m.worldId == other.worldId &&
		m.channelId == other.channelId &&
		m.mapId == other.mapId
}

// IsInstanced returns true if this field represents an instanced map (non-nil instance UUID).
func (m Model) IsInstanced() bool {
	return m.instance != uuid.Nil
}

// WithInstance returns a new Model with the same world/channel/map but a different instance.
func (m Model) WithInstance(instance uuid.UUID) Model {
	return Model{
		worldId:   m.worldId,
		channelId: m.channelId,
		mapId:     m.mapId,
		instance:  instance,
	}
}

// WithoutInstance returns a new Model with the same world/channel/map but uuid.Nil instance.
func (m Model) WithoutInstance() Model {
	return m.WithInstance(uuid.Nil)
}

func FromId(id Id) (Model, bool) {
	var worldId world.Id
	var channelId channel.Id
	var mapId _map.Id
	var instanceStr string

	// Parse the first three fields and the UUID string
	count, err := fmt.Sscanf(string(id), "%d:%d:%d:%s", &worldId, &channelId, &mapId, &instanceStr)
	if err != nil {
		return Model{}, false
	}
	if count != 4 {
		return Model{}, false
	}

	// Parse the UUID string
	instance, err := uuid.Parse(instanceStr)
	if err != nil {
		return Model{}, false
	}

	return Model{
		worldId:   worldId,
		channelId: channelId,
		mapId:     mapId,
		instance:  instance,
	}, true
}

type Builder struct {
	worldId   world.Id
	channelId channel.Id
	mapId     _map.Id
	instance  uuid.UUID
}

func NewBuilder(worldId world.Id, channelId channel.Id, mapId _map.Id) *Builder {
	return &Builder{
		worldId:   worldId,
		channelId: channelId,
		mapId:     mapId,
		instance:  uuid.Nil,
	}
}

func (m *Builder) SetWorldId(worldId world.Id) *Builder {
	m.worldId = worldId
	return m
}

func (m *Builder) SetChannelId(channelId channel.Id) *Builder {
	m.channelId = channelId
	return m
}

func (m *Builder) SetMapId(mapId _map.Id) *Builder {
	m.mapId = mapId
	return m
}

func (m *Builder) SetInstance(instance uuid.UUID) *Builder {
	m.instance = instance
	return m
}

func (m *Builder) Build() Model {
	return Model{
		worldId:   m.worldId,
		channelId: m.channelId,
		mapId:     m.mapId,
		instance:  m.instance,
	}
}
