package _map

import (
	"atlas-data/map/monster"
	"atlas-data/map/npc"
	"atlas-data/map/portal"
	"atlas-data/map/reactor"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Id                string                    `json:"-"`
	Name              string                    `json:"name"`
	StreetName        string                    `json:"streetName"`
	ReturnMapId       uint32                    `json:"returnMapId"`
	MonsterRate       float64                   `json:"monsterRate"`
	OnFirstUserEnter  string                    `json:"onFirstUserEnter"`
	OnUserEnter       string                    `json:"onUserEnter"`
	FieldLimit        uint32                    `json:"fieldLimit"`
	MobInterval       uint32                    `json:"mobInterval"`
	Portals           []portal.RestModel        `json:"-"`
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
	Reactors          []reactor.RestModel       `json:"-"`
	NPCs              []npc.RestModel           `json:"-"`
	Monsters          []monster.RestModel       `json:"-"`
}

func (r RestModel) GetName() string {
	return "maps"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
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
			ID:   x.Id,
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
			if data, ok = refMap[rid.ID]; ok {
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
			if data, ok = refMap[rid.ID]; ok {
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
			if data, ok = refMap[rid.ID]; ok {
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
			if data, ok = refMap[rid.ID]; ok {
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

func Transform(m Model) (RestModel, error) {
	ps, err := model.SliceMap(portal.Transform)(model.FixedProvider(m.Portals))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	ma, err := TransformRectangle(m.MapArea)
	if err != nil {
		return RestModel{}, err
	}

	as, err := model.SliceMap(TransformRectangle)(model.FixedProvider(m.Areas))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	bt, err := model.SliceMap(TransformBackgroundType)(model.FixedProvider(m.BackgroundTypes))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	rs, err := model.SliceMap(reactor.Transform)(model.FixedProvider(m.Reactors))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	ns, err := model.SliceMap(npc.Transform)(model.FixedProvider(m.Npcs))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	ms, err := model.SliceMap(monster.Transform)(model.FixedProvider(m.Monsters))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:                strconv.Itoa(int(m.Id)),
		Name:              m.Name,
		StreetName:        m.StreetName,
		ReturnMapId:       m.ReturnMapId,
		MonsterRate:       m.MonsterRate,
		OnFirstUserEnter:  m.OnFirstUserEnter,
		OnUserEnter:       m.OnUserEnter,
		FieldLimit:        m.FieldLimit,
		MobInterval:       m.MobInterval,
		Portals:           ps,
		MapArea:           ma,
		Areas:             as,
		Seats:             m.Seats,
		Clock:             m.Clock,
		EverLast:          m.EverLast,
		Town:              m.Town,
		DecHP:             m.DecHp,
		ProtectItem:       m.ProtectItem,
		ForcedReturnMapId: m.ForcedReturnMapId,
		Boat:              m.Boat,
		TimeLimit:         m.TimeLimit,
		FieldType:         m.FieldType,
		MobCapacity:       m.MobCapacity,
		Recovery:          m.Recovery,
		BackgroundTypes:   bt,
		Reactors:          rs,
		NPCs:              ns,
		Monsters:          ms,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	id, err := strconv.Atoi(rm.Id)
	if err != nil {
		return Model{}, err
	}

	ps, err := model.SliceMap(portal.Extract)(model.FixedProvider(rm.Portals))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	ma, err := ExtractRectangle(rm.MapArea)
	if err != nil {
		return Model{}, err
	}

	as, err := model.SliceMap(ExtractRectangle)(model.FixedProvider(rm.Areas))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	bt, err := model.SliceMap(ExtractBackgroundType)(model.FixedProvider(rm.BackgroundTypes))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	rs, err := model.SliceMap(reactor.Extract)(model.FixedProvider(rm.Reactors))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	ns, err := model.SliceMap(npc.Extract)(model.FixedProvider(rm.NPCs))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	ms, err := model.SliceMap(monster.Extract)(model.FixedProvider(rm.Monsters))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	return Model{
		Id:                uint32(id),
		Name:              rm.Name,
		StreetName:        rm.StreetName,
		ReturnMapId:       rm.ReturnMapId,
		MonsterRate:       rm.MonsterRate,
		OnFirstUserEnter:  rm.OnFirstUserEnter,
		OnUserEnter:       rm.OnUserEnter,
		FieldLimit:        rm.FieldLimit,
		MobInterval:       rm.MobInterval,
		Portals:           ps,
		TimeMob:           nil,
		MapArea:           ma,
		FootholdTree:      nil,
		Areas:             as,
		Seats:             rm.Seats,
		Clock:             rm.Clock,
		EverLast:          rm.EverLast,
		Town:              rm.Town,
		DecHp:             rm.DecHP,
		ProtectItem:       rm.ProtectItem,
		ForcedReturnMapId: rm.ForcedReturnMapId,
		Boat:              rm.Boat,
		TimeLimit:         rm.TimeLimit,
		FieldType:         rm.FieldType,
		MobCapacity:       rm.MobCapacity,
		Recovery:          rm.Recovery,
		BackgroundTypes:   bt,
		XLimit:            XLimit{},
		Reactors:          rs,
		Npcs:              ns,
		Monsters:          ms,
	}, nil
}

type FootholdTreeRestModel struct {
}

type RectangleRestModel struct {
	X      int16 `json:"x"`
	Y      int16 `json:"y"`
	Width  int16 `json:"width"`
	Height int16 `json:"height"`
}

func TransformRectangle(m Rectangle) (RectangleRestModel, error) {
	return RectangleRestModel{
		X:      m.X,
		Y:      m.Y,
		Width:  m.Width,
		Height: m.Height,
	}, nil
}

func ExtractRectangle(rm RectangleRestModel) (Rectangle, error) {
	return Rectangle{
		X:      rm.X,
		Y:      rm.Y,
		Width:  rm.Width,
		Height: rm.Height,
	}, nil
}

type BackgroundTypeRestModel struct {
	LayerNumber    uint32 `json:"layerNumber"`
	BackgroundType uint32 `json:"backgroundType"`
}

func TransformBackgroundType(m BackgroundType) (BackgroundTypeRestModel, error) {
	return BackgroundTypeRestModel{
		LayerNumber:    m.LayerNumber,
		BackgroundType: m.BackgroundType,
	}, nil
}

func ExtractBackgroundType(m BackgroundTypeRestModel) (BackgroundType, error) {
	return BackgroundType{
		LayerNumber:    m.LayerNumber,
		BackgroundType: m.BackgroundType,
	}, nil
}
