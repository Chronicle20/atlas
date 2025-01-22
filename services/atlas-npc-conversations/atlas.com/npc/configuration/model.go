package configuration

import "errors"

type Model struct {
	Data Data `json:"data"`
}

func (d *Model) FindServer(tenantId string) (Server, error) {
	for _, v := range d.Data.Attributes.Servers {
		if v.Tenant == tenantId {
			return v, nil
		}
	}
	return Server{}, errors.New("server not found")
}

// Data contains the main data configuration.
type Data struct {
	Id         string     `json:"id"`
	Type       string     `json:"type"`
	Attributes Attributes `json:"attributes"`
}

// Attributes contain all settings under attributes key.
type Attributes struct {
	Servers []Server `json:"servers"`
}

// Server represents a server in the configuration.
type Server struct {
	Tenant  string   `json:"tenant"`
	Region  string   `json:"region"`
	Version Version  `json:"version"`
	Scripts []Script `json:"scripts"`
}

// Version represents a server version.
type Version struct {
	Major string `json:"major"`
	Minor string `json:"minor"`
}

// Script represents a npc script.
type Script struct {
	NPCId uint32 `json:"npcId"`
	Impl  string `json:"impl"`
}
