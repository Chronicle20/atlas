package guild

import (
	"atlas-guilds/guild/member"
	"atlas-guilds/guild/title"
	"errors"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

// Builder provides fluent construction of guild models
type Builder struct {
	tenantId            *uuid.UUID
	id                  *uint32
	worldId             *world.Id
	name                *string
	notice              *string
	points              *uint32
	capacity            *uint32
	logo                *uint16
	logoColor           *byte
	logoBackground      *uint16
	logoBackgroundColor *byte
	leaderId            *uint32
	members             []member.Model
	titles              []title.Model
}

// NewBuilder creates a new builder with required parameters
func NewBuilder(tenantId uuid.UUID, id uint32, worldId world.Id, name string, leaderId uint32) *Builder {
	return &Builder{
		tenantId: &tenantId,
		id:       &id,
		worldId:  &worldId,
		name:     &name,
		leaderId: &leaderId,
		members:  make([]member.Model, 0),
		titles:   make([]title.Model, 0),
	}
}

// SetNotice sets the guild notice
func (b *Builder) SetNotice(notice string) *Builder {
	b.notice = &notice
	return b
}

// SetPoints sets the guild points
func (b *Builder) SetPoints(points uint32) *Builder {
	b.points = &points
	return b
}

// SetCapacity sets the guild member capacity
func (b *Builder) SetCapacity(capacity uint32) *Builder {
	b.capacity = &capacity
	return b
}

// SetLogo sets the guild logo
func (b *Builder) SetLogo(logo uint16) *Builder {
	b.logo = &logo
	return b
}

// SetLogoColor sets the guild logo color
func (b *Builder) SetLogoColor(logoColor byte) *Builder {
	b.logoColor = &logoColor
	return b
}

// SetLogoBackground sets the guild logo background
func (b *Builder) SetLogoBackground(logoBackground uint16) *Builder {
	b.logoBackground = &logoBackground
	return b
}

// SetLogoBackgroundColor sets the guild logo background color
func (b *Builder) SetLogoBackgroundColor(logoBackgroundColor byte) *Builder {
	b.logoBackgroundColor = &logoBackgroundColor
	return b
}

// SetMembers sets the guild members
func (b *Builder) SetMembers(members []member.Model) *Builder {
	b.members = make([]member.Model, len(members))
	copy(b.members, members)
	return b
}

// SetTitles sets the guild titles
func (b *Builder) SetTitles(titles []title.Model) *Builder {
	b.titles = make([]title.Model, len(titles))
	copy(b.titles, titles)
	return b
}

// Build validates invariants and constructs the final immutable model
func (b *Builder) Build() (Model, error) {
	if b.tenantId == nil {
		return Model{}, errors.New("tenant ID is required")
	}
	if b.id == nil {
		return Model{}, errors.New("guild ID is required")
	}
	if *b.id == 0 {
		return Model{}, errors.New("guild ID must be greater than 0")
	}
	if b.worldId == nil {
		return Model{}, errors.New("world ID is required")
	}
	if b.name == nil || *b.name == "" {
		return Model{}, errors.New("guild name is required")
	}
	if b.leaderId == nil {
		return Model{}, errors.New("leader ID is required")
	}
	if *b.leaderId == 0 {
		return Model{}, errors.New("leader ID must be greater than 0")
	}

	// Default capacity to 30 if not set
	capacity := uint32(30)
	if b.capacity != nil {
		if *b.capacity == 0 {
			return Model{}, errors.New("capacity must be greater than 0")
		}
		capacity = *b.capacity
	}

	// Default optional values
	notice := ""
	if b.notice != nil {
		notice = *b.notice
	}

	points := uint32(0)
	if b.points != nil {
		points = *b.points
	}

	logo := uint16(0)
	if b.logo != nil {
		logo = *b.logo
	}

	logoColor := byte(0)
	if b.logoColor != nil {
		logoColor = *b.logoColor
	}

	logoBackground := uint16(0)
	if b.logoBackground != nil {
		logoBackground = *b.logoBackground
	}

	logoBackgroundColor := byte(0)
	if b.logoBackgroundColor != nil {
		logoBackgroundColor = *b.logoBackgroundColor
	}

	return Model{
		tenantId:            *b.tenantId,
		id:                  *b.id,
		worldId:             *b.worldId,
		name:                *b.name,
		notice:              notice,
		points:              points,
		capacity:            capacity,
		logo:                logo,
		logoColor:           logoColor,
		logoBackground:      logoBackground,
		logoBackgroundColor: logoBackgroundColor,
		leaderId:            *b.leaderId,
		members:             b.members,
		titles:              b.titles,
	}, nil
}

// Builder returns a builder initialized with the current model's values
func (m Model) Builder() *Builder {
	// Create value copies to preserve immutability of the original model
	tenantId := m.tenantId
	id := m.id
	worldId := m.worldId
	name := m.name
	notice := m.notice
	points := m.points
	capacity := m.capacity
	logo := m.logo
	logoColor := m.logoColor
	logoBackground := m.logoBackground
	logoBackgroundColor := m.logoBackgroundColor
	leaderId := m.leaderId

	return &Builder{
		tenantId:            &tenantId,
		id:                  &id,
		worldId:             &worldId,
		name:                &name,
		notice:              &notice,
		points:              &points,
		capacity:            &capacity,
		logo:                &logo,
		logoColor:           &logoColor,
		logoBackground:      &logoBackground,
		logoBackgroundColor: &logoBackgroundColor,
		leaderId:            &leaderId,
		members:             append([]member.Model{}, m.members...),
		titles:              append([]title.Model{}, m.titles...),
	}
}
