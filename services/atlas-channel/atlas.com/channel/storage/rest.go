package storage

import "time"

// StorageRestModel represents the storage REST response from atlas-storage
type StorageRestModel struct {
	Id        string `json:"-"`
	WorldId   byte   `json:"world_id"`
	AccountId uint32 `json:"account_id"`
	Capacity  uint32 `json:"capacity"`
	Mesos     uint32 `json:"mesos"`
}

func (r StorageRestModel) GetName() string {
	return "storages"
}

func (r StorageRestModel) GetID() string {
	return r.Id
}

func (r *StorageRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// AssetRestModel represents an asset REST response from atlas-storage
type AssetRestModel struct {
	Id            string    `json:"-"`
	InventoryType byte      `json:"inventory_type"`
	Slot          int16     `json:"slot"`
	TemplateId    uint32    `json:"template_id"`
	Expiration    time.Time `json:"expiration"`
	ReferenceId   uint32    `json:"reference_id"`
	ReferenceType string    `json:"reference_type"`
	Quantity      uint32    `json:"quantity,omitempty"`
	OwnerId       uint32    `json:"owner_id,omitempty"`
	Flag          uint16    `json:"flag,omitempty"`
}

func (r AssetRestModel) GetName() string {
	return "storage_assets"
}

func (r AssetRestModel) GetID() string {
	return r.Id
}

func (r *AssetRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
