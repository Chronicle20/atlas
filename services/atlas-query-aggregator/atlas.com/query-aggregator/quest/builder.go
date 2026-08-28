package quest

import (
	"strconv"
	"time"
)

// Builder provides a fluent API for building quest models
type Builder struct {
	characterId uint32
	questId     uint32
	state       State
	startedAt   time.Time
	completedAt time.Time
	progress    map[string]int
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{
		progress: make(map[string]int),
	}
}

// SetCharacterId sets the character ID
func (b *Builder) SetCharacterId(id uint32) *Builder {
	b.characterId = id
	return b
}

// SetId sets the quest ID
func (b *Builder) SetId(id uint32) *Builder {
	b.questId = id
	return b
}

// SetQuestId sets the quest ID (alias for SetId)
func (b *Builder) SetQuestId(id uint32) *Builder {
	b.questId = id
	return b
}

// SetStatus sets the quest state
func (b *Builder) SetStatus(state State) *Builder {
	b.state = state
	return b
}

// SetState sets the quest state (alias for SetStatus)
func (b *Builder) SetState(state State) *Builder {
	b.state = state
	return b
}

// SetStartedAt sets the started time
func (b *Builder) SetStartedAt(t time.Time) *Builder {
	b.startedAt = t
	return b
}

// SetCompletedAt sets the completed time
func (b *Builder) SetCompletedAt(t time.Time) *Builder {
	b.completedAt = t
	return b
}

// SetProgress sets a progress entry by key (info number as string)
func (b *Builder) SetProgress(key string, value int) *Builder {
	b.progress[key] = value
	return b
}

// Build creates the Model from the builder
func (b *Builder) Build() Model {
	progressModels := make([]ProgressModel, 0, len(b.progress))
	progressByKey := make(map[string]int, len(b.progress))

	for key, value := range b.progress {
		// Store in the string-keyed map for direct lookup
		progressByKey[key] = value

		// Also try to store as numeric info number for backward compatibility
		infoNumber, err := strconv.ParseUint(key, 10, 32)
		if err != nil {
			// If key is not a number, use 0
			infoNumber = 0
		}
		progressModels = append(progressModels, ProgressModel{
			infoNumber: uint32(infoNumber),
			progress:   strconv.Itoa(value),
		})
	}

	return Model{
		characterId:   b.characterId,
		questId:       b.questId,
		state:         b.state,
		startedAt:     b.startedAt,
		completedAt:   b.completedAt,
		progress:      progressModels,
		progressByKey: progressByKey,
	}
}
