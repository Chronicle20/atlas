package server

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

func Register(t tenant.Model, worldId world.Id, channelId channel.Id, ipAddress string, port int) Model {
	m := Model{
		tenant:    t,
		worldId:   worldId,
		channelId: channelId,
		ipAddress: ipAddress,
		port:      port,
	}
	getRegistry().Register(m)
	return m
}

func GetAll() []Model {
	return getRegistry().GetAll()
}
