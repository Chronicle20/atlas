package data

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const equipmentPath = "data/equipment/%d"

type EquipmentRestModel struct {
	Id uint32 `json:"-"`
}

func (e EquipmentRestModel) GetName() string { return "statistics" }
func (e EquipmentRestModel) GetID() string   { return fmt.Sprint(e.Id) }
func (e *EquipmentRestModel) SetID(id string) error {
	var x uint64
	if _, err := fmt.Sscan(id, &x); err != nil {
		return err
	}
	e.Id = uint32(x)
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs satisfy the jsonapi
// UnmarshalToOneRelations / UnmarshalToManyRelations interfaces. atlas-data's
// /data/equipment/{id} response includes a "slots" toMany relationship; without
// these stubs api2go's Unmarshal fails with "struct does not implement
// UnmarshalToManyRelations", which the caller surfaces as ErrNotFound. The
// relationship payload is irrelevant to existence checks, so the methods are
// intentionally no-ops.
func (e *EquipmentRestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (e *EquipmentRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func requestEquipmentById(ctx context.Context, id uint32) requests.Request[EquipmentRestModel] {
	root, err := getDataBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[EquipmentRestModel](err)
	}
	return requests.GetRequest[EquipmentRestModel](fmt.Sprintf("%s"+equipmentPath, root, id))
}
