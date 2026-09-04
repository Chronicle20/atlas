package ops

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

func TestWarpToPortal(t *testing.T) {
	instID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	plain := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()
	withPortal := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).SetInstance(instID).Build()).SetPortalId(7).Build()

	tests := []struct {
		name        string
		target      Target
		params      map[string]string
		wantErr     string
		wantParam   *ParamError
		wantPayload saga.WarpToPortalPayload
	}{
		{
			name:    "missing mapId",
			target:  plain,
			params:  map[string]string{},
			wantErr: `warp_to_portal: parameter "mapId" is required`,
		},
		{
			name:      "bad mapId",
			target:    plain,
			params:    map[string]string{"mapId": "abc"},
			wantParam: &ParamError{Op: "warp_to_portal", Param: "mapId", Value: "abc"},
		},
		{
			name:   "defaults",
			target: plain,
			params: map[string]string{"mapId": "104000000"},
			wantPayload: saga.WarpToPortalPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       104000000,
				Instance:    uuid.Nil,
				PortalId:    0,
				PortalName:  "",
			},
		},
		{
			name:   "portalId",
			target: plain,
			params: map[string]string{"mapId": "104000000", "portalId": "3"},
			wantPayload: saga.WarpToPortalPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       104000000,
				Instance:    uuid.Nil,
				PortalId:    3,
			},
		},
		{
			name:   "portalName",
			target: plain,
			params: map[string]string{"mapId": "104000000", "portalName": "west00"},
			wantPayload: saga.WarpToPortalPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       104000000,
				Instance:    uuid.Nil,
				PortalName:  "west00",
			},
		},
		{
			name:   "instance never carried",
			target: withPortal,
			params: map[string]string{"mapId": "910010000"},
			wantPayload: saga.WarpToPortalPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       910010000,
				Instance:    uuid.Nil,
				PortalId:    0,
			},
		},
		{
			name:      "bad portalId",
			target:    plain,
			params:    map[string]string{"mapId": "1", "portalId": "abc"},
			wantParam: &ParamError{Op: "warp_to_portal", Param: "portalId", Value: "abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := WarpToPortal(tt.params, DirectResolver{}, tt.target, 7)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("got error %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if tt.wantParam != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var pe *ParamError
				if !errors.As(err, &pe) {
					t.Fatalf("expected *ParamError, got %T: %v", err, err)
				}
				if pe.Op != tt.wantParam.Op || pe.Param != tt.wantParam.Param || pe.Value != tt.wantParam.Value {
					t.Fatalf("got ParamError{Op:%q,Param:%q,Value:%q}, want {Op:%q,Param:%q,Value:%q}",
						pe.Op, pe.Param, pe.Value, tt.wantParam.Op, tt.wantParam.Param, tt.wantParam.Value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if step.Action() != saga.WarpToPortal {
				t.Fatalf("got action %v, want %v", step.Action(), saga.WarpToPortal)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.WarpToPortalPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload != tt.wantPayload {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}

func TestWarpToSavedLocation(t *testing.T) {
	plain := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()

	tests := []struct {
		name        string
		params      map[string]string
		wantErr     string
		wantPayload saga.WarpToSavedLocationPayload
	}{
		{
			name:    "missing locationType",
			params:  map[string]string{},
			wantErr: `warp_to_saved_location: parameter "locationType" is required`,
		},
		{
			name:   "ok",
			params: map[string]string{"locationType": "FREE_MARKET"},
			wantPayload: saga.WarpToSavedLocationPayload{
				CharacterId:  7,
				WorldId:      0,
				ChannelId:    1,
				LocationType: "FREE_MARKET",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := WarpToSavedLocation(tt.params, DirectResolver{}, plain, 7)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("got error %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if step.Action() != saga.WarpToSavedLocation {
				t.Fatalf("got action %v, want %v", step.Action(), saga.WarpToSavedLocation)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.WarpToSavedLocationPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload != tt.wantPayload {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}

func TestSaveLocation(t *testing.T) {
	instID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	plain := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()
	withPortal := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).SetInstance(instID).Build()).SetPortalId(7).Build()

	tests := []struct {
		name        string
		target      Target
		params      map[string]string
		wantErr     string
		wantParam   *ParamError
		wantPayload saga.SaveLocationPayload
	}{
		{
			name:    "missing locationType",
			target:  plain,
			params:  map[string]string{},
			wantErr: `save_location: parameter "locationType" is required`,
		},
		{
			name:   "defaults, no portal on target",
			target: plain,
			params: map[string]string{"locationType": "FREE_MARKET"},
			wantPayload: saga.SaveLocationPayload{
				CharacterId:  7,
				WorldId:      0,
				ChannelId:    1,
				LocationType: "FREE_MARKET",
				MapId:        910010000,
				PortalId:     0,
			},
		},
		{
			name:   "default portal from target",
			target: withPortal,
			params: map[string]string{"locationType": "FREE_MARKET"},
			wantPayload: saga.SaveLocationPayload{
				CharacterId:  7,
				WorldId:      0,
				ChannelId:    1,
				LocationType: "FREE_MARKET",
				MapId:        910010000,
				PortalId:     7,
			},
		},
		{
			name:   "explicit override",
			target: withPortal,
			params: map[string]string{"locationType": "EVENT", "mapId": "104000000", "portalId": "2"},
			wantPayload: saga.SaveLocationPayload{
				CharacterId:  7,
				WorldId:      0,
				ChannelId:    1,
				LocationType: "EVENT",
				MapId:        104000000,
				PortalId:     2,
			},
		},
		{
			name:      "bad mapId",
			target:    plain,
			params:    map[string]string{"locationType": "E", "mapId": "abc"},
			wantParam: &ParamError{Op: "save_location", Param: "mapId", Value: "abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := SaveLocation(tt.params, DirectResolver{}, tt.target, 7)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("got error %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if tt.wantParam != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var pe *ParamError
				if !errors.As(err, &pe) {
					t.Fatalf("expected *ParamError, got %T: %v", err, err)
				}
				if pe.Op != tt.wantParam.Op || pe.Param != tt.wantParam.Param || pe.Value != tt.wantParam.Value {
					t.Fatalf("got ParamError{Op:%q,Param:%q,Value:%q}, want {Op:%q,Param:%q,Value:%q}",
						pe.Op, pe.Param, pe.Value, tt.wantParam.Op, tt.wantParam.Param, tt.wantParam.Value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if step.Action() != saga.SaveLocation {
				t.Fatalf("got action %v, want %v", step.Action(), saga.SaveLocation)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.SaveLocationPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload != tt.wantPayload {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}

func TestStartInstanceTransport(t *testing.T) {
	plain := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()

	tests := []struct {
		name        string
		params      map[string]string
		wantErr     string
		wantPayload saga.StartInstanceTransportPayload
	}{
		{
			name:    "missing routeName",
			params:  map[string]string{},
			wantErr: `start_instance_transport: parameter "routeName" is required`,
		},
		{
			name:   "ok",
			params: map[string]string{"routeName": "kerning-square-subway-in"},
			wantPayload: saga.StartInstanceTransportPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				RouteName:   "kerning-square-subway-in",
			},
		},
		{
			name:   "failureMessage not read into payload",
			params: map[string]string{"routeName": "r", "failureMessage": "nope"},
			wantPayload: saga.StartInstanceTransportPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				RouteName:   "r",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := StartInstanceTransport(tt.params, DirectResolver{}, plain, 7)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("got error %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if step.Action() != saga.StartInstanceTransport {
				t.Fatalf("got action %v, want %v", step.Action(), saga.StartInstanceTransport)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.StartInstanceTransportPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload != tt.wantPayload {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}
