package map_

import (
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/jtumidanski/api2go/jsonapi"
)

// PortalRestModel mirrors atlas-data's portal sub-resource wire format.
type PortalRestModel struct {
	Id          string  `json:"-"`
	Name        string  `json:"name"`
	Type        uint8   `json:"type"`
	X           int16   `json:"x"`
	Y           int16   `json:"y"`
	TargetMapId _map.Id `json:"targetMapId"`
}

func (r PortalRestModel) GetName() string {
	return "portals"
}

func (r PortalRestModel) GetID() string {
	return r.Id
}

func (r *PortalRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r *PortalRestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *PortalRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// ExtractPortal converts a PortalRestModel into an immutable Portal.
// Exported so tests can call it directly without network I/O.
func ExtractPortal(rm PortalRestModel) (Portal, error) {
	var id uint32
	if rm.Id != "" {
		parsed, err := strconv.Atoi(rm.Id)
		if err != nil {
			return Portal{}, err
		}
		id = uint32(parsed)
	}
	return Portal{
		id:          id,
		name:        rm.Name,
		portalType:  rm.Type,
		x:           rm.X,
		y:           rm.Y,
		targetMapId: rm.TargetMapId,
	}, nil
}

// RestModel mirrors the subset of atlas-data's map wire format that atlas-doors
// needs (returnMapId, forcedReturnMapId, town, fieldLimit, portals).
type RestModel struct {
	Id                _map.Id           `json:"-"`
	ReturnMapId       _map.Id           `json:"returnMapId"`
	ForcedReturnMapId _map.Id           `json:"forcedReturnMapId"`
	Town              bool              `json:"town"`
	FieldLimit        uint32            `json:"fieldLimit"`
	Portals           []PortalRestModel `json:"-"`
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

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "portals" {
		res := make([]PortalRestModel, 0, len(IDs))
		for _, id := range IDs {
			rm := PortalRestModel{}
			if err := rm.SetID(id); err != nil {
				return err
			}
			res = append(res, rm)
		}
		r.Portals = res
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["portals"]; ok {
		res := make([]PortalRestModel, 0)
		for _, rid := range r.getPortalReferenceIDs() {
			if data, ok := refMap[rid]; ok {
				var rm PortalRestModel
				if err := jsonapi.ProcessIncludeData(&rm, data, references); err != nil {
					return err
				}
				_ = rm.SetID(rid)
				res = append(res, rm)
			}
		}
		r.Portals = res
	}
	return nil
}

func (r *RestModel) getPortalReferenceIDs() []string {
	ids := make([]string, 0, len(r.Portals))
	for _, p := range r.Portals {
		ids = append(ids, p.Id)
	}
	return ids
}

// Extract converts a RestModel (with portals already populated) into an
// immutable Model.
func Extract(rm RestModel) (Model, error) {
	portals := make([]Portal, 0, len(rm.Portals))
	for _, prm := range rm.Portals {
		p, err := ExtractPortal(prm)
		if err != nil {
			return Model{}, err
		}
		portals = append(portals, p)
	}
	return NewBuilder(rm.Id).
		SetReturnMapId(rm.ReturnMapId).
		SetForcedReturnMapId(rm.ForcedReturnMapId).
		SetTown(rm.Town).
		SetFieldLimit(rm.FieldLimit).
		SetPortals(portals).
		Build(), nil
}
