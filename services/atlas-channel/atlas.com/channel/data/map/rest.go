package map_

import (
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id                _map.Id               `json:"-"`
	Name              string                `json:"name"`
	StreetName        string                `json:"streetName"`
	ReturnMapId       _map.Id               `json:"returnMapId"`
	MonsterRate       float64               `json:"monsterRate"`
	OnFirstUserEnter  string                `json:"onFirstUserEnter"`
	OnUserEnter       string                `json:"onUserEnter"`
	FieldLimit        uint32                `json:"fieldLimit"`
	MobInterval       uint32                `json:"mobInterval"`
	Seats             uint32                `json:"seats"`
	Clock             bool                  `json:"clock"`
	EverLast          bool                  `json:"everLast"`
	Town              bool                  `json:"town"`
	DecHP             uint32                `json:"decHP"`
	ProtectItem       uint32                `json:"protectItem"`
	ForcedReturnMapId _map.Id               `json:"forcedReturnMapId"`
	Boat              bool                  `json:"boat"`
	TimeLimit         int32                 `json:"timeLimit"`
	FieldType         uint32                `json:"fieldType"`
	MobCapacity       uint32                `json:"mobCapacity"`
	Recovery          float64               `json:"recovery"`
	FootholdTree      FootholdTreeRestModel `json:"footholdTree"`
}

// FootholdTreeRestModel mirrors atlas-data's recursive quadtree node. Only
// the Footholds array is needed by the channel — child nodes are walked
// during flatten.
type FootholdTreeRestModel struct {
	NorthWest *FootholdTreeRestModel `json:"north_west,omitempty"`
	NorthEast *FootholdTreeRestModel `json:"north_east,omitempty"`
	SouthWest *FootholdTreeRestModel `json:"south_west,omitempty"`
	SouthEast *FootholdTreeRestModel `json:"south_east,omitempty"`
	Footholds []FootholdRestModel    `json:"footholds"`
}

type FootholdRestModel struct {
	Id     uint32        `json:"id"`
	First  *PointRestModel `json:"first"`
	Second *PointRestModel `json:"second"`
}

type PointRestModel struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

func (r RestModel) GetName() string {
	return "maps"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = _map.Id(id)
	return nil
}

func (r *RestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	footholds := make(map[uint32]Foothold)
	flattenFootholds(&rm.FootholdTree, footholds)
	return Model{
		clock:       rm.Clock,
		returnMapId: rm.ReturnMapId,
		fieldLimit:  rm.FieldLimit,
		town:        rm.Town,
		footholds:   footholds,
	}, nil
}

// flattenFootholds walks the recursive quadtree and collects every foothold
// keyed by id, so the channel-side Model can do O(1) lookups.
func flattenFootholds(node *FootholdTreeRestModel, out map[uint32]Foothold) {
	if node == nil {
		return
	}
	for _, fh := range node.Footholds {
		if fh.First == nil || fh.Second == nil {
			continue
		}
		out[fh.Id] = Foothold{
			Id:      fh.Id,
			FirstX:  fh.First.X,
			FirstY:  fh.First.Y,
			SecondX: fh.Second.X,
			SecondY: fh.Second.Y,
		}
	}
	flattenFootholds(node.NorthWest, out)
	flattenFootholds(node.NorthEast, out)
	flattenFootholds(node.SouthWest, out)
	flattenFootholds(node.SouthEast, out)
}
