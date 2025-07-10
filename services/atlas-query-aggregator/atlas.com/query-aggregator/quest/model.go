package quest

// QuestStatus represents the status of a quest
type QuestStatus int

const (
	// UNDEFINED represents an undefined quest status
	UNDEFINED QuestStatus = iota
	// NOT_STARTED represents a quest that hasn't been started
	NOT_STARTED
	// STARTED represents a quest that has been started
	STARTED
	// COMPLETED represents a quest that has been completed
	COMPLETED
)

// String returns the string representation of the quest status
func (s QuestStatus) String() string {
	switch s {
	case UNDEFINED:
		return "UNDEFINED"
	case NOT_STARTED:
		return "NOT_STARTED"
	case STARTED:
		return "STARTED"
	case COMPLETED:
		return "COMPLETED"
	default:
		return "UNDEFINED"
	}
}

// FromString creates a QuestStatus from a string
func FromString(s string) QuestStatus {
	switch s {
	case "UNDEFINED":
		return UNDEFINED
	case "NOT_STARTED":
		return NOT_STARTED
	case "STARTED":
		return STARTED
	case "COMPLETED":
		return COMPLETED
	default:
		return UNDEFINED
	}
}

// Model represents a quest and its progress
type Model struct {
	id       uint32
	status   QuestStatus
	progress map[string]int
}

// NewModel creates a new quest model
func NewModel(id uint32, status QuestStatus) Model {
	return Model{
		id:       id,
		status:   status,
		progress: make(map[string]int),
	}
}

// Id returns the quest ID
func (m Model) Id() uint32 {
	return m.id
}

// Status returns the quest status
func (m Model) Status() QuestStatus {
	return m.status
}

// Progress returns the progress for a specific step
func (m Model) Progress(step string) int {
	if progress, exists := m.progress[step]; exists {
		return progress
	}
	return 0
}

// SetProgress sets the progress for a specific step
func (m Model) SetProgress(step string, value int) Model {
	newProgress := make(map[string]int)
	for k, v := range m.progress {
		newProgress[k] = v
	}
	newProgress[step] = value
	
	return Model{
		id:       m.id,
		status:   m.status,
		progress: newProgress,
	}
}

// ModelBuilder provides a builder pattern for creating quest models
type ModelBuilder struct {
	id       uint32
	status   QuestStatus
	progress map[string]int
}

// NewModelBuilder creates a new quest model builder
func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{
		progress: make(map[string]int),
	}
}

// SetId sets the quest ID
func (b *ModelBuilder) SetId(id uint32) *ModelBuilder {
	b.id = id
	return b
}

// SetStatus sets the quest status
func (b *ModelBuilder) SetStatus(status QuestStatus) *ModelBuilder {
	b.status = status
	return b
}

// SetProgress sets the progress for a specific step
func (b *ModelBuilder) SetProgress(step string, value int) *ModelBuilder {
	if b.progress == nil {
		b.progress = make(map[string]int)
	}
	b.progress[step] = value
	return b
}

// Build creates a quest model from the builder
func (b *ModelBuilder) Build() Model {
	return Model{
		id:       b.id,
		status:   b.status,
		progress: b.progress,
	}
}