package transport

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"time"
)

// RestModel is the JSON:API resource for a transport route
type RestModel struct {
	ID               uuid.UUID               `json:"-"`
	Name             string                  `json:"name"`
	StartMapID       _map.Id                 `json:"startMapId"`
	StagingMapID     _map.Id                 `json:"stagingMapId"`
	EnRouteMapIDs    []_map.Id               `json:"enRouteMapIds"`
	DestinationMapID _map.Id                 `json:"destinationMapId"`
	ObservationMapID _map.Id                 `json:"observationMapId"`
	State            string                  `json:"state"`
	CycleInterval    time.Duration           `json:"cycleInterval"`
	Schedule         []TripScheduleRestModel `json:"-"`
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.ID.String()
}

// SetID sets the resource ID
func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	r.ID = id
	return nil
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return "routes"
}

// GetReferences returns the resource's relationships
func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Name:         "schedule",
			Type:         "trip-schedule",
			Relationship: jsonapi.ToManyRelationship,
		},
	}
}

// GetReferencedIDs returns the resource's relationship IDs
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	result := []jsonapi.ReferenceID{}

	// Add schedule relationships if they exist
	for _, schedule := range r.Schedule {
		result = append(result, jsonapi.ReferenceID{
			ID:           schedule.GetID(),
			Name:         "schedule",
			Type:         "trip-schedule",
			Relationship: jsonapi.ToManyRelationship,
		})
	}

	return result
}

// GetReferencedStructs returns the resource's relationship structs
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	result := []jsonapi.MarshalIdentifier{}

	// Add schedule relationships if they exist
	for i := range r.Schedule {
		result = append(result, &r.Schedule[i])
	}

	return result
}

// SetToOneReferenceID sets a to-one relationship
func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

// SetToManyReferenceIDs sets a to-many relationship
func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "schedule" {
		r.Schedule = make([]TripScheduleRestModel, len(IDs))
		for i, ID := range IDs {
			err := r.Schedule[i].SetID(ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Transform converts a Model to a RestModel
func Transform(m Model) (RestModel, error) {
	schedule, err := model.SliceMap(TransformSchedule)(model.FixedProvider(m.Schedule()))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		ID:               m.Id(),
		Name:             m.Name(),
		StartMapID:       m.StartMapId(),
		StagingMapID:     m.StagingMapId(),
		EnRouteMapIDs:    m.EnRouteMapIds(),
		DestinationMapID: m.DestinationMapId(),
		ObservationMapID: m.ObservationMapId(),
		State:            string(m.State()),
		CycleInterval:    m.CycleInterval(),
		Schedule:         schedule,
	}, nil
}

func Extract(r RestModel) (Model, error) {
	var schedule []TripScheduleModel
	for _, s := range r.Schedule {
		sm, err := ExtractSchedule(s)
		if err != nil {
			return Model{}, err
		}
		sm = sm.Builder().SetRouteId(r.ID).Build()
		schedule = append(schedule, sm)
	}

	return NewBuilder(r.Name).
		SetStartMapId(r.StartMapID).
		SetStagingMapId(r.StagingMapID).
		SetEnRouteMapIds(r.EnRouteMapIDs).
		SetDestinationMapId(r.DestinationMapID).
		SetObservationMapId(r.ObservationMapID).
		SetState(RouteState(r.State)).
		SetSchedule(schedule).
		SetCycleInterval(r.CycleInterval).
		Build(), nil
}

// TripScheduleRestModel is the JSON:API resource for a trip schedule
type TripScheduleRestModel struct {
	ID             uuid.UUID `json:"-"`
	BoardingOpen   time.Time `json:"boardingOpen"`
	BoardingClosed time.Time `json:"boardingClosed"`
	Departure      time.Time `json:"departure"`
	Arrival        time.Time `json:"arrival"`
}

// GetID returns the resource ID
func (r TripScheduleRestModel) GetID() string {
	return r.ID.String()
}

// SetID sets the resource ID
func (r *TripScheduleRestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	r.ID = id
	return nil
}

// GetName returns the resource name
func (r TripScheduleRestModel) GetName() string {
	return "trip-schedule"
}

// TransformSchedule converts a TripScheduleModel to a TripScheduleRestModel
func TransformSchedule(m TripScheduleModel) (TripScheduleRestModel, error) {
	return TripScheduleRestModel{
		ID:             m.TripId(),
		BoardingOpen:   m.BoardingOpen(),
		BoardingClosed: m.BoardingClosed(),
		Departure:      m.Departure(),
		Arrival:        m.Arrival(),
	}, nil
}

func ExtractSchedule(r TripScheduleRestModel) (TripScheduleModel, error) {
	return NewTripScheduleBuilder().
		SetTripId(r.ID).
		SetBoardingOpen(r.BoardingOpen).
		SetBoardingClosed(r.BoardingClosed).
		SetDeparture(r.Departure).
		SetArrival(r.Arrival).
		Build(), nil
}
