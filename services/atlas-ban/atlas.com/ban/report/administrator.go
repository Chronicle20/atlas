package report

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB) func(tenantId uuid.UUID, kind Kind, reporterId uint32, reporterName string, accusedId uint32, accusedName string, reasonType byte, description string, chatLog *string, transcript []TranscriptLine) (Model, error) {
	return func(tenantId uuid.UUID, kind Kind, reporterId uint32, reporterName string, accusedId uint32, accusedName string, reasonType byte, description string, chatLog *string, transcript []TranscriptLine) (Model, error) {
		m, err := NewBuilder(tenantId, kind, reporterId).
			SetId(uuid.New()).
			SetReporterName(reporterName).
			SetAccusedId(accusedId).
			SetAccusedName(accusedName).
			SetReasonType(reasonType).
			SetDescription(description).
			SetChatLog(chatLog).
			SetServerTranscript(transcript).
			Build()
		if err != nil {
			return Model{}, err
		}

		e, err := m.ToEntity()
		if err != nil {
			return Model{}, err
		}

		if err := db.Create(&e).Error; err != nil {
			return Model{}, err
		}
		return Make(e)
	}
}

func updateStatus(db *gorm.DB) func(id uuid.UUID, status Status) error {
	return func(id uuid.UUID, status Status) error {
		result := db.Model(&Entity{}).Where("id = ?", id).Update("status", string(status))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	}
}

func Make(e Entity) (Model, error) {
	var transcript []TranscriptLine
	if len(e.ServerTranscript) > 0 {
		if err := json.Unmarshal(e.ServerTranscript, &transcript); err != nil {
			return Model{}, err
		}
	}
	return NewBuilder(e.TenantId, Kind(e.Kind), e.ReporterId).
		SetId(e.Id).
		SetReporterName(e.ReporterName).
		SetAccusedId(e.AccusedId).
		SetAccusedName(e.AccusedName).
		SetReasonType(e.ReasonType).
		SetDescription(e.Description).
		SetChatLog(e.ChatLog).
		SetServerTranscript(transcript).
		SetStatus(Status(e.Status)).
		SetCreatedAt(e.CreatedAt).
		SetUpdatedAt(e.UpdatedAt).
		Build()
}
