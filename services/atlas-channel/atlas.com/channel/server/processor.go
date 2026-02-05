package server

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-tenant"
)

func Register(t tenant.Model, ch channel.Model, ipAddress string, port int) Model {
	m := Model{
		tenant:    t,
		ch:        ch,
		ipAddress: ipAddress,
		port:      port,
	}
	getRegistry().Register(m)
	return m
}

func GetAll() []Model {
	return getRegistry().GetAll()
}
