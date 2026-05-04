package equipment

import "strconv"

// RestModel mirrors the requirement subset of atlas-data's equipment endpoint.
// atlas-data exposes many more fields (per-stat bonuses, classification, etc.)
// but atlas-effective-stats only needs the requirement gate inputs — per-asset
// stats already arrive via atlas-inventory.
type RestModel struct {
	Id       uint32 `json:"-"`
	ReqLevel byte   `json:"reqLevel"`
	ReqJob   uint16 `json:"reqJob"`
	ReqStr   uint16 `json:"reqStr"`
	ReqDex   uint16 `json:"reqDex"`
	ReqInt   uint16 `json:"reqInt"`
	ReqLuk   uint16 `json:"reqLuk"`
}

// GetName must match the JSON:API `type` atlas-data emits for the equipment
// endpoint. atlas-data's equipment RestModel returns "statistics" (see
// services/atlas-data/atlas.com/data/equipment/rest.go) — using "equipment"
// here causes api2go.Unmarshal to fail with a type-mismatch, which the caller
// surfaces as ErrNotFound and silently downgrades every player's equipment to
// "unqualified" in production.
func (r RestModel) GetName() string { return "statistics" }

func (r RestModel) GetID() string { return strconv.Itoa(int(r.Id)) }

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs satisfy the jsonapi
// UnmarshalToOneRelations / UnmarshalToManyRelations interfaces. atlas-data's
// /data/equipment/{id} response includes a "slots" toMany relationship; without
// these stubs api2go's Unmarshal fails with "struct does not implement
// UnmarshalToManyRelations", which the caller surfaces as ErrNotFound. The
// relationship payload is irrelevant to requirement-gate inputs, so the
// methods are intentionally no-ops.
func (r *RestModel) SetToOneReferenceID(_, _ string) error             { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
