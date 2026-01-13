package thread

import (
	"atlas-guilds/thread/reply"
	"errors"
	"github.com/google/uuid"
	"time"
)

// Builder provides fluent construction of thread models
type Builder struct {
	tenantId   *uuid.UUID
	guildId    *uint32
	id         *uint32
	posterId   *uint32
	title      *string
	message    *string
	emoticonId *uint32
	notice     *bool
	createdAt  *time.Time
	replies    []reply.Model
}

// NewBuilder creates a new builder with required parameters
func NewBuilder(tenantId uuid.UUID, guildId uint32, id uint32, posterId uint32, title string, message string) *Builder {
	return &Builder{
		tenantId: &tenantId,
		guildId:  &guildId,
		id:       &id,
		posterId: &posterId,
		title:    &title,
		message:  &message,
		replies:  make([]reply.Model, 0),
	}
}

// SetEmoticonId sets the thread emoticon ID
func (b *Builder) SetEmoticonId(emoticonId uint32) *Builder {
	b.emoticonId = &emoticonId
	return b
}

// SetNotice sets whether this thread is a notice
func (b *Builder) SetNotice(notice bool) *Builder {
	b.notice = &notice
	return b
}

// SetCreatedAt sets the creation timestamp
func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = &createdAt
	return b
}

// SetReplies sets the thread replies
func (b *Builder) SetReplies(replies []reply.Model) *Builder {
	b.replies = make([]reply.Model, len(replies))
	copy(b.replies, replies)
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
	if b.id == nil {
		return Model{}, errors.New("thread ID is required")
	}
	if *b.id == 0 {
		return Model{}, errors.New("thread ID must be greater than 0")
	}
	if b.posterId == nil {
		return Model{}, errors.New("poster ID is required")
	}
	if *b.posterId == 0 {
		return Model{}, errors.New("poster ID must be greater than 0")
	}
	if b.title == nil || *b.title == "" {
		return Model{}, errors.New("thread title is required")
	}
	if b.message == nil {
		return Model{}, errors.New("thread message is required")
	}

	// Default optional values
	emoticonId := uint32(0)
	if b.emoticonId != nil {
		emoticonId = *b.emoticonId
	}

	notice := false
	if b.notice != nil {
		notice = *b.notice
	}

	createdAt := time.Now()
	if b.createdAt != nil {
		createdAt = *b.createdAt
	}

	return Model{
		tenantId:   *b.tenantId,
		guildId:    *b.guildId,
		id:         *b.id,
		posterId:   *b.posterId,
		title:      *b.title,
		message:    *b.message,
		emoticonId: emoticonId,
		notice:     notice,
		createdAt:  createdAt,
		replies:    b.replies,
	}, nil
}

// Builder returns a builder initialized with the current model's values
func (m Model) Builder() *Builder {
	// Create value copies to preserve immutability of the original model
	tenantId := m.tenantId
	guildId := m.guildId
	id := m.id
	posterId := m.posterId
	title := m.title
	message := m.message
	emoticonId := m.emoticonId
	notice := m.notice
	createdAt := m.createdAt

	return &Builder{
		tenantId:   &tenantId,
		guildId:    &guildId,
		id:         &id,
		posterId:   &posterId,
		title:      &title,
		message:    &message,
		emoticonId: &emoticonId,
		notice:     &notice,
		createdAt:  &createdAt,
		replies:    append([]reply.Model{}, m.replies...),
	}
}
