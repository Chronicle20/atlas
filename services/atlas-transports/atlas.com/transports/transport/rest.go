package transport

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// RestModel is the JSON:API resource for a transport route.
//
// CycleInterval is a time.Duration and therefore serialises as an integer
// nanosecond count. It is retained unchanged for existing consumers; new
// consumers read the unit-explicit *Seconds fields below.
type RestModel struct {
	ID                    uuid.UUID     `json:"-"`
	Name                  string        `json:"name"`
	StartMapID            _map.Id       `json:"startMapId"`
	StagingMapID          _map.Id       `json:"stagingMapId"`
	EnRouteMapIDs         []_map.Id     `json:"enRouteMapIds"`
	DestinationMapID      _map.Id       `json:"destinationMapId"`
	ObservationMapID      _map.Id       `json:"observationMapId"`
	State                 string        `json:"state"`
	CycleInterval         time.Duration `json:"cycleInterval"`
	BoardingWindowSeconds uint32        `json:"boardingWindowSeconds"`
	PreDepartureSeconds   uint32        `json:"preDepartureSeconds"`
	TravelDurationSeconds uint32        `json:"travelDurationSeconds"`
	CycleIntervalSeconds  uint32        `json:"cycleIntervalSeconds"`
	// NextTransitionAt is the absolute instant of the next state change,
	// projected from the schedule's time-of-day boundaries onto the first
	// instant after the server's `now`. Empty when the route is out of
	// service. Clients count down to this rather than reconstructing the
	// scheduler.
	NextTransitionAt string `json:"nextTransitionAt"`
	NextState        string `json:"nextState"`
	// VoyageID identifies the concrete trip currently under way. Empty unless
	// State is in_transit. A consumer holding a voyage id from VOYAGE_DEPARTED
	// tests "is my voyage still under way?" by comparing this for equality —
	// comparing State alone would read the NEXT trip's transit as its own
	// (design §7.4).
	VoyageID string                  `json:"voyageId,omitempty"`
	Schedule []TripScheduleRestModel `json:"-"`
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
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
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

// TransformSummaryAt converts a Model to a RestModel without its trip
// schedule, evaluating state at `now`. The board renders entirely from these
// attributes; a full day's schedule is ~96 rows per route and is fetched
// only where it is actually read.
//
// State and NextState/NextTransitionAt all come from one Evaluate call on one
// `now`, so a response can never report a state that disagrees with its own
// countdown. VoyageID is populated only when the evaluated transition is
// in_transit: a route that is out of service, boarding, or between legs is
// not on any voyage, and a client comparing a stale voyage id for equality
// must see an unambiguous "no voyage" rather than a fabricated zero id.
func TransformSummaryAt(ctx context.Context, m Model, now time.Time) (RestModel, error) {
	transition := m.Evaluate(now)

	nextAt := ""
	nextState := ""
	if transition.State != OutOfService && !transition.NextAt.IsZero() {
		nextAt = transition.NextAt.Format(time.RFC3339)
		nextState = string(transition.NextState)
	}

	voyageId := ""
	if transition.State == InTransit {
		voyageId = VoyageId(tenant.MustFromContext(ctx), m.Id(), transition.TripId, transition.DepartedAt).String()
	}

	return RestModel{
		ID:                    m.Id(),
		Name:                  m.Name(),
		StartMapID:            m.StartMapId(),
		StagingMapID:          m.StagingMapId(),
		EnRouteMapIDs:         m.EnRouteMapIds(),
		DestinationMapID:      m.DestinationMapId(),
		ObservationMapID:      m.ObservationMapId(),
		State:                 string(transition.State),
		CycleInterval:         m.CycleInterval(),
		BoardingWindowSeconds: uint32(m.BoardingWindowDuration().Seconds()),
		PreDepartureSeconds:   uint32(m.PreDepartureDuration().Seconds()),
		TravelDurationSeconds: uint32(m.TravelDuration().Seconds()),
		CycleIntervalSeconds:  uint32(m.CycleInterval().Seconds()),
		NextTransitionAt:      nextAt,
		NextState:             nextState,
		VoyageID:              voyageId,
	}, nil
}

// TransformSummary converts a Model to a RestModel without its trip
// schedule, evaluating state at the current time. See TransformSummaryAt.
func TransformSummary(ctx context.Context, m Model) (RestModel, error) {
	return TransformSummaryAt(ctx, m, timeNow().UTC())
}

// TransformAt converts a Model to a RestModel with its trip schedule
// attached, evaluating state at `now`.
func TransformAt(ctx context.Context, m Model, now time.Time) (RestModel, error) {
	rm, err := TransformSummaryAt(ctx, m, now)
	if err != nil {
		return RestModel{}, err
	}

	schedule, err := model.SliceMap(TransformSchedule)(model.FixedProvider(m.Schedule()))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}
	rm.Schedule = schedule

	return rm, nil
}

// Transform converts a Model to a RestModel with its trip schedule attached,
// evaluating state at the current time. See TransformAt.
func Transform(ctx context.Context, m Model) (RestModel, error) {
	return TransformAt(ctx, m, timeNow().UTC())
}

func Extract(r RestModel) (Model, error) {
	var schedule []TripScheduleModel
	for _, s := range r.Schedule {
		sm, err := ExtractSchedule(s)
		if err != nil {
			return Model{}, err
		}
		smWithRoute, err := sm.Builder().SetRouteId(r.ID).Build()
		if err != nil {
			return Model{}, err
		}
		schedule = append(schedule, smWithRoute)
	}

	return NewBuilder(r.Name).
		SetStartMapId(r.StartMapID).
		SetStagingMapId(r.StagingMapID).
		SetEnRouteMapIds(r.EnRouteMapIDs).
		SetDestinationMapId(r.DestinationMapID).
		SetObservationMapId(r.ObservationMapID).
		SetState(RouteState(r.State)).
		SetSchedule(schedule).
		SetBoardingWindowDuration(time.Duration(r.BoardingWindowSeconds) * time.Second).
		SetPreDepartureDuration(time.Duration(r.PreDepartureSeconds) * time.Second).
		SetTravelDuration(time.Duration(r.TravelDurationSeconds) * time.Second).
		SetCycleInterval(r.CycleInterval).
		Build()
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
		Build()
}
