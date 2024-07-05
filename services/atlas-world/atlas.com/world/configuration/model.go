package configuration

import "errors"

type Model struct {
	Data Data `json:"data"`
}

func (m *Model) FindWorld(tenantId string, index byte) (World, error) {
	var found = false
	var server Server
	for _, s := range m.Data.Attributes.Servers {
		if s.Tenant == tenantId {
			found = true
			server = s
			break
		}
	}
	if !found {
		return World{}, errors.New("tenant not found")
	}
	if len(server.Worlds) < 0 || int(index) >= len(server.Worlds) {
		return World{}, errors.New("index out of bounds")
	}
	return server.Worlds[index], nil
}

// Data contains the main data configuration.
type Data struct {
	Type       string     `json:"type"`
	Attributes Attributes `json:"attributes"`
}

// Attributes contain all settings under attributes key.
type Attributes struct {
	Servers []Server `json:"servers"`
}

// Server represents a server in the configuration.
type Server struct {
	Tenant string  `json:"tenant"`
	Worlds []World `json:"worlds"`
}

type World struct {
	Name              string `json:"name"`
	Flag              string `json:"flag"`
	ServerMessage     string `json:"serverMessage"`
	EventMessage      string `json:"eventMessage"`
	WhyAmIRecommended string `json:"whyAmIRecommended"`
}
