package cash

import (
	"atlas-cashshop/cash/wishlist"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Id       uint32               `json:"-"`
	Credit   uint32               `json:"credit"`
	Points   uint32               `json:"points"`
	Prepaid  uint32               `json:"prepaid"`
	Wishlist []wishlist.RestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "cashes"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "wishlist_items",
			Name: "wishlist_items",
		},
	}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, w := range r.Wishlist {
		result = append(result, jsonapi.ReferenceID{
			ID:   w.GetID(),
			Type: w.GetName(),
			Name: w.GetName(),
		})
	}
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for _, w := range r.Wishlist {
		result = append(result, w)
	}
	return result
}

func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "wishlist_items" {
		if r.Wishlist == nil {
			r.Wishlist = make([]wishlist.RestModel, 0)
		}
		for _, id := range IDs {
			wli := &wishlist.RestModel{}
			err := wli.SetID(id)
			if err != nil {
				continue
			}
			r.Wishlist = append(r.Wishlist, *wli)
		}
		return nil
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["wishlist_items"]; ok {
		res := make([]wishlist.RestModel, 0)
		for _, rid := range r.GetReferencedIDs() {
			var data jsonapi.Data
			if data, ok = refMap[rid.ID]; ok {
				var srm = wishlist.RestModel{}
				err := jsonapi.ProcessIncludeData(&srm, data, references)
				if err != nil {
					return err
				}
				res = append(res, srm)
			}
		}
		r.Wishlist = res
	}
	return nil
}

func Transform(m Model) (RestModel, error) {
	wl, err := model.SliceMap(wishlist.Transform)(model.FixedProvider(m.wishlist))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}
	return RestModel{
		Id:       m.characterId,
		Credit:   m.credit,
		Points:   m.points,
		Prepaid:  m.prepaid,
		Wishlist: wl,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	wl, err := model.SliceMap(wishlist.Extract)(model.FixedProvider(rm.Wishlist))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}
	return Model{
		characterId: rm.Id,
		credit:      rm.Credit,
		points:      rm.Points,
		prepaid:     rm.Prepaid,
		wishlist:    wl,
	}, nil
}
