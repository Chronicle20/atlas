package quest

import (
	"atlas-quest/quest/progress"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId       uuid.UUID         `gorm:"not null;index:idx_quest_tenant_char"`
	ID             uint32            `gorm:"primaryKey;autoIncrement;not null"`
	CharacterId    uint32            `gorm:"not null;index:idx_quest_tenant_char"`
	QuestId        uint32            `gorm:"not null;index"`
	State          State             `gorm:"not null;default:0"`
	StartedAt      time.Time         `gorm:"not null"`
	CompletedAt    time.Time         `gorm:""`
	ExpirationTime time.Time         `gorm:""` // For time-limited quests
	CompletedCount uint32            `gorm:"not null;default:0"`
	ForfeitCount   uint32            `gorm:"not null;default:0"`
	Progress       []progress.Entity `gorm:"foreignKey:QuestStatusId"`
}

func (e Entity) TableName() string {
	return "quest_statuses"
}

func Make(e Entity) (Model, error) {
	ps := make([]progress.Model, 0)
	for _, pe := range e.Progress {
		p, err := progress.Make(pe)
		if err != nil {
			return Model{}, err
		}
		ps = append(ps, p)
	}

	return Model{
		tenantId:       e.TenantId,
		id:             e.ID,
		characterId:    e.CharacterId,
		questId:        e.QuestId,
		state:          e.State,
		startedAt:      e.StartedAt,
		completedAt:    e.CompletedAt,
		expirationTime: e.ExpirationTime,
		completedCount: e.CompletedCount,
		forfeitCount:   e.ForfeitCount,
		progress:       ps,
	}, nil
}
