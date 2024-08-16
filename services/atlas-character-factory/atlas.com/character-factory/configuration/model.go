package configuration

import "errors"

type Model struct {
	Data Data `json:"data"`
}

func (d *Model) FindTemplate(tenantId string, jobIndex uint32, subJobIndex uint32, gender byte) (Template, error) {
	for _, s := range d.Data.Attributes.Servers {
		if s.Tenant == tenantId {
			for _, t := range s.Templates {
				if t.JobIndex == jobIndex && t.SubJobIndex == subJobIndex && t.Gender == gender {
					return t, nil
				}
			}
			return Template{}, errors.New("template configuration not found")
		}
	}
	return Template{}, errors.New("tenant not found")
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
	Tenant    string     `json:"tenant"`
	Templates []Template `json:"templates"`
}

type Template struct {
	JobIndex          uint32   `json:"jobIndex"`
	SubJobIndex       uint32   `json:"subJobIndex"`
	MapId             uint32   `json:"mapId"`
	Gender            byte     `json:"gender"`
	Face              []uint32 `json:"face"`
	Hair              []uint32 `json:"hair"`
	HairColor         []uint32 `json:"hairColor"`
	SkinColor         []uint32 `json:"skinColor"`
	Top               []uint32 `json:"top"`
	Bottom            []uint32 `json:"bottom"`
	Shoes             []uint32 `json:"shoes"`
	Weapon            []uint32 `json:"weapon"`
	StartingInventory []uint32 `json:"startingInventory"`
}
