package storage

import (
	"encoding/json"
	"strconv"
	"time"
)

// AssetRestModel represents an asset from the storage service
type AssetRestModel struct {
	Id            string          `json:"-"`
	Slot          int16           `json:"slot"`
	TemplateId    uint32          `json:"templateId"`
	Expiration    time.Time       `json:"expiration"`
	ReferenceId   uint32          `json:"referenceId"`
	ReferenceType string          `json:"referenceType"`
	ReferenceData json.RawMessage `json:"referenceData"`
}

func (r AssetRestModel) GetName() string {
	return "storage_assets"
}

func (r AssetRestModel) GetID() string {
	return r.Id
}

func (r *AssetRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = strconv.Itoa(id)
	return nil
}
