package member

import (
	"errors"
	"github.com/google/uuid"
)

// Builder provides fluent construction of member models
type Builder struct {
	tenantId      *uuid.UUID
	guildId       *uint32
	characterId   *uint32
	name          *string
	jobId         *uint16
	level         *byte
	title         *byte
	online        *bool
	allianceTitle *byte
}

// NewBuilder creates a new builder with required parameters
func NewBuilder(tenantId uuid.UUID, guildId uint32, characterId uint32, name string) *Builder {
	return &Builder{
		tenantId:    &tenantId,
		guildId:     &guildId,
		characterId: &characterId,
		name:        &name,
	}
}

// SetJobId sets the member's job ID
func (b *Builder) SetJobId(jobId uint16) *Builder {
	b.jobId = &jobId
	return b
}

// SetLevel sets the member's level
func (b *Builder) SetLevel(level byte) *Builder {
	b.level = &level
	return b
}

// SetTitle sets the member's guild title index
func (b *Builder) SetTitle(title byte) *Builder {
	b.title = &title
	return b
}

// SetOnline sets the member's online status
func (b *Builder) SetOnline(online bool) *Builder {
	b.online = &online
	return b
}

// SetAllianceTitle sets the member's alliance title
func (b *Builder) SetAllianceTitle(allianceTitle byte) *Builder {
	b.allianceTitle = &allianceTitle
	return b
}

// Build validates invariants and constructs the final immutable model
func (b *Builder) Build() (Model, error) {
	if b.tenantId == nil {
		return Model{}, errors.New("tenant ID is required")
	}
	if b.guildId == nil {
		return Model{}, errors.New("guild ID is required")
	}
	if *b.guildId == 0 {
		return Model{}, errors.New("guild ID must be greater than 0")
	}
	if b.characterId == nil {
		return Model{}, errors.New("character ID is required")
	}
	if *b.characterId == 0 {
		return Model{}, errors.New("character ID must be greater than 0")
	}
	if b.name == nil || *b.name == "" {
		return Model{}, errors.New("member name is required")
	}

	// Default optional values
	jobId := uint16(0)
	if b.jobId != nil {
		jobId = *b.jobId
	}

	level := byte(0)
	if b.level != nil {
		level = *b.level
	}

	title := byte(0)
	if b.title != nil {
		title = *b.title
	}

	online := false
	if b.online != nil {
		online = *b.online
	}

	allianceTitle := byte(0)
	if b.allianceTitle != nil {
		allianceTitle = *b.allianceTitle
	}

	return Model{
		tenantId:      *b.tenantId,
		guildId:       *b.guildId,
		characterId:   *b.characterId,
		name:          *b.name,
		jobId:         jobId,
		level:         level,
		title:         title,
		online:        online,
		allianceTitle: allianceTitle,
	}, nil
}

// Builder returns a builder initialized with the current model's values
func (m Model) Builder() *Builder {
	// Create value copies to preserve immutability of the original model
	tenantId := m.tenantId
	guildId := m.guildId
	characterId := m.characterId
	name := m.name
	jobId := m.jobId
	level := m.level
	title := m.title
	online := m.online
	allianceTitle := m.allianceTitle

	return &Builder{
		tenantId:      &tenantId,
		guildId:       &guildId,
		characterId:   &characterId,
		name:          &name,
		jobId:         &jobId,
		level:         &level,
		title:         &title,
		online:        &online,
		allianceTitle: &allianceTitle,
	}
}
