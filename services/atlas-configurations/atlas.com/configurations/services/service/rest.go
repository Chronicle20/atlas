package service

import "atlas-configurations/services/task"

type GenericRestModel struct {
	Id    string           `json:"-"`
	Tasks []task.RestModel `json:"tasks"`
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

type LoginRestModel struct {
	Id      string                 `json:"-"`
	Tasks   []task.RestModel       `json:"tasks"`
	Tenants []LoginTenantRestModel `json:"tenants"`
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
	Tasks   []task.RestModel         `json:"tasks"`
	Tenants []ChannelTenantRestModel `json:"tenants"`
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
