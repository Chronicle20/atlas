package party_quest

import (
	"strconv"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)

type TimerRestModel struct {
	Id       uint32 `json:"-"`
	Duration uint64 `json:"duration"`
}

func (r TimerRestModel) GetName() string {
	return "timers"
}

func (r TimerRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *TimerRestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r TimerRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (r TimerRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (r TimerRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

func (r *TimerRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *TimerRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *TimerRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func ExtractTimer(rm TimerRestModel) (TimerModel, error) {
	return TimerModel{
		characterId: rm.Id,
		duration:    time.Duration(rm.Duration) * time.Second,
	}, nil
}
