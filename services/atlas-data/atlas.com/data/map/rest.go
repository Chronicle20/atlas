package _map

import (
	"atlas-data/map/monster"
	"atlas-data/map/npc"
	"atlas-data/map/portal"
	"atlas-data/map/reactor"
	"atlas-data/point"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Id                uint32                    `json:"-"`
	Name              string                    `json:"name"`
	StreetName        string                    `json:"streetName"`
	ReturnMapId       uint32                    `json:"returnMapId"`
	MonsterRate       float64                   `json:"monsterRate"`
	OnFirstUserEnter  string                    `json:"onFirstUserEnter"`
	OnUserEnter       string                    `json:"onUserEnter"`
	FieldLimit        uint32                    `json:"fieldLimit"`
	MobInterval       uint32                    `json:"mobInterval"`
	Portals           []portal.RestModel        `json:"-"`
	TimeMob           *TimeMobRestModel         `json:"time_mob"`
	MapArea           RectangleRestModel        `json:"mapArea"`
	FootholdTree      FootholdTreeRestModel     `json:"footholdTree"`
	Areas             []RectangleRestModel      `json:"areas"`
	Seats             uint32                    `json:"seats"`
	Clock             bool                      `json:"clock"`
	EverLast          bool                      `json:"everLast"`
	Town              bool                      `json:"town"`
	DecHP             uint32                    `json:"decHP"`
	ProtectItem       uint32                    `json:"protectItem"`
	ForcedReturnMapId uint32                    `json:"forcedReturnMapId"`
	Boat              bool                      `json:"boat"`
	TimeLimit         int32                     `json:"timeLimit"`
	FieldType         uint32                    `json:"fieldType"`
	MobCapacity       uint32                    `json:"mobCapacity"`
	Recovery          float64                   `json:"recovery"`
	BackgroundTypes   []BackgroundTypeRestModel `json:"backgroundTypes"`
	XLimit            XLimitRestModel           `json:"x_limit"`
	Reactors          []reactor.RestModel       `json:"-"`
	NPCs              []npc.RestModel           `json:"-"`
	Monsters          []monster.RestModel       `json:"-"`
}

func (r RestModel) GetName() string {
	return "maps"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r RestModel) GetId() uint32 {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r RestModel) GetCustomLinks(url string) jsonapi.Links {
	lnks := make(map[string]jsonapi.Link)
	lnks["self"] = jsonapi.Link{Href: url}
	return lnks
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	rfs := make([]jsonapi.Reference, 0)
	rfs = append(rfs, jsonapi.Reference{Type: "portals", Name: "portals"})
	rfs = append(rfs, jsonapi.Reference{Type: "reactors", Name: "reactors"})
	rfs = append(rfs, jsonapi.Reference{Type: "npcs", Name: "npcs"})
	rfs = append(rfs, jsonapi.Reference{Type: "monsters", Name: "monsters"})
	return rfs
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	rfs := make([]jsonapi.ReferenceID, 0)
	for _, x := range r.Portals {
		rfs = append(rfs, jsonapi.ReferenceID{
			ID:   x.Id,
			Type: "portals",
			Name: "portals",
		})
	}
	for _, x := range r.Reactors {
		rfs = append(rfs, jsonapi.ReferenceID{
			ID:   strconv.Itoa(int(x.Id)),
			Type: "reactors",
			Name: "reactors",
		})
	}
	for _, x := range r.NPCs {
		rfs = append(rfs, jsonapi.ReferenceID{
			ID:   strconv.Itoa(int(x.Id)),
			Type: "npcs",
			Name: "npcs",
		})
	}
	for _, x := range r.Monsters {
		rfs = append(rfs, jsonapi.ReferenceID{
			ID:   strconv.Itoa(int(x.Id)),
			Type: "monsters",
			Name: "monsters",
		})
	}
	return rfs
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	rfs := make([]jsonapi.MarshalIdentifier, 0)
	for _, x := range r.Portals {
		rfs = append(rfs, x)
	}
	for _, x := range r.Reactors {
		rfs = append(rfs, x)
	}
	for _, x := range r.NPCs {
		rfs = append(rfs, x)
	}
	for _, x := range r.Monsters {
		rfs = append(rfs, x)
	}
	return rfs
}

func (r *RestModel) SetToOneReferenceID(name string, ID string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "portals" {
		res := make([]portal.RestModel, 0)
		for _, x := range IDs {
			rm := portal.RestModel{}
			err := rm.SetID(x)
			if err != nil {
				return err
			}
			res = append(res, rm)
		}
		r.Portals = res
	}
	if name == "reactors" {
		res := make([]reactor.RestModel, 0)
		for _, x := range IDs {
			rm := reactor.RestModel{}
			err := rm.SetID(x)
			if err != nil {
				return err
			}
			res = append(res, rm)
		}
		r.Reactors = res
	}
	if name == "npcs" {
		res := make([]npc.RestModel, 0)
		for _, x := range IDs {
			rm := npc.RestModel{}
			err := rm.SetID(x)
			if err != nil {
				return err
			}
			res = append(res, rm)
		}
		r.NPCs = res
	}
	if name == "monsters" {
		res := make([]monster.RestModel, 0)
		for _, x := range IDs {
			rm := monster.RestModel{}
			err := rm.SetID(x)
			if err != nil {
				return err
			}
			res = append(res, rm)
		}
		r.Monsters = res
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["portals"]; ok {
		res := make([]portal.RestModel, 0)
		for _, rid := range r.GetReferencedIDs() {
			var data jsonapi.Data
			if data, ok = refMap[rid.ID]; ok && data.Type == rid.Type {
				var rm portal.RestModel
				err := jsonapi.ProcessIncludeData(&rm, data, references)
				if err != nil {
					return err
				}
				_ = rm.SetID(rid.ID)
				res = append(res, rm)
			}
		}
		r.Portals = res
	}
	if refMap, ok := references["reactors"]; ok {
		res := make([]reactor.RestModel, 0)
		for _, rid := range r.GetReferencedIDs() {
			var data jsonapi.Data
			if data, ok = refMap[rid.ID]; ok && data.Type == rid.Type {
				var rm reactor.RestModel
				err := jsonapi.ProcessIncludeData(&rm, data, references)
				if err != nil {
					return err
				}
				_ = rm.SetID(rid.ID)
				res = append(res, rm)
			}
		}
		r.Reactors = res
	}
	if refMap, ok := references["npcs"]; ok {
		res := make([]npc.RestModel, 0)
		for _, rid := range r.GetReferencedIDs() {
			var data jsonapi.Data
			if data, ok = refMap[rid.ID]; ok && data.Type == rid.Type {
				var rm npc.RestModel
				err := jsonapi.ProcessIncludeData(&rm, data, references)
				if err != nil {
					return err
				}
				_ = rm.SetID(rid.ID)
				res = append(res, rm)
			}
		}
		r.NPCs = res
	}
	if refMap, ok := references["monsters"]; ok {
		res := make([]monster.RestModel, 0)
		for _, rid := range r.GetReferencedIDs() {
			var data jsonapi.Data
			if data, ok = refMap[rid.ID]; ok && data.Type == rid.Type {
				var rm monster.RestModel
				err := jsonapi.ProcessIncludeData(&rm, data, references)
				if err != nil {
					return err
				}
				_ = rm.SetID(rid.ID)
				res = append(res, rm)
			}
		}
		r.Monsters = res
	}
	return nil
}

type FootholdTreeRestModel struct {
	NorthWest *FootholdTreeRestModel `json:"north_west,omitempty"`
	NorthEast *FootholdTreeRestModel `json:"north_east,omitempty"`
	SouthWest *FootholdTreeRestModel `json:"south_west,omitempty"`
	SouthEast *FootholdTreeRestModel `json:"south_east,omitempty"`
	Footholds []FootholdRestModel    `json:"footholds"`
	P1        *point.RestModel       `json:"p1,omitempty"`
	P2        *point.RestModel       `json:"p2,omitempty"`
	Center    *point.RestModel       `json:"center,omitempty"`
	Depth     uint32                 `json:"depth"`
	MaxDropX  int16                  `json:"max_drop_x"`
	MinDropX  int16                  `json:"min_drop_x"`
}

type RectangleRestModel struct {
	X      int16 `json:"x"`
	Y      int16 `json:"y"`
	Width  int16 `json:"width"`
	Height int16 `json:"height"`
}

type BackgroundTypeRestModel struct {
	LayerNumber    uint32 `json:"layerNumber"`
	BackgroundType uint32 `json:"backgroundType"`
}
