package service

import (
	"atlas-configurations/services/task"
	"encoding/json"
)

type GenericRestModel struct {
	Id    string           `json:"-"`
	Type  string           `json:"type,omitempty"`
	Tasks []task.RestModel `json:"tasks"`
	// Environment is server-owned and read-only (task-232 FR-7.3, mirroring
	// tenants.RestModel/templates.RestModel): it always reflects
	// Entity.Environment, set by services/processor.go's Make(). Create and
	// UpdateById never read this field from a client-supplied body — see
	// configurations/tenants/processor.go:100-113 for the pattern this
	// copies. Added for task-48 fix round 2 Critical 1: cleanup.sh's
	// _dcp_reclaim filters this collection on attributes.environment, which
	// was always null before this field existed.
	Environment string `json:"environment"`
}

func (r GenericRestModel) GetName() string {
	return "services"
}

func (r GenericRestModel) GetID() string {
	return r.Id
}

func (r *GenericRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// InputRestModel is used for creating and updating services.
// It includes the type field to determine which service type to create.
// Tenants is stored as raw JSON since it differs by service type.
type InputRestModel struct {
	Id      string           `json:"-"`
	Type    string           `json:"type"`
	Tasks   []task.RestModel `json:"tasks"`
	Tenants json.RawMessage  `json:"tenants,omitempty"`
}

func (r InputRestModel) GetName() string {
	return "services"
}

func (r InputRestModel) GetID() string {
	return r.Id
}

func (r *InputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

type LoginRestModel struct {
	Id      string                 `json:"-"`
	Type    string                 `json:"type"`
	Tasks   []task.RestModel       `json:"tasks"`
	Tenants []LoginTenantRestModel `json:"tenants"`
	// Environment is server-owned and read-only; see GenericRestModel's
	// Environment field comment above.
	Environment string `json:"environment"`
}

func (r LoginRestModel) GetName() string {
	return "services"
}

func (r LoginRestModel) GetID() string {
	return r.Id
}

func (r *LoginRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

type LoginTenantRestModel struct {
	Id   string `json:"id"`
	Port int    `json:"port"`
}

type ChannelRestModel struct {
	Id      string                   `json:"-"`
	Type    string                   `json:"type"`
	Tasks   []task.RestModel         `json:"tasks"`
	Tenants []ChannelTenantRestModel `json:"tenants"`
	// Environment is server-owned and read-only; see GenericRestModel's
	// Environment field comment above.
	Environment string `json:"environment"`
}

func (r ChannelRestModel) GetName() string {
	return "services"
}

func (r ChannelRestModel) GetID() string {
	return r.Id
}

func (r *ChannelRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

type ChannelTenantRestModel struct {
	Id        string                  `json:"id"`
	IPAddress string                  `json:"ipAddress"`
	Worlds    []ChannelWorldRestModel `json:"worlds"`
}

type ChannelWorldRestModel struct {
	Id       byte                      `json:"id"`
	Channels []ChannelChannelRestModel `json:"channels"`
}

type ChannelChannelRestModel struct {
	Id   byte `json:"id"`
	Port int  `json:"port"`
}
