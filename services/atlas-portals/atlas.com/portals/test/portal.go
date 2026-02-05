package test

import (
	"atlas-portals/portal"

	_map "github.com/Chronicle20/atlas-constants/map"
)

// PortalFixture contains test data for creating portal models
type PortalFixture struct {
	Id          string
	Name        string
	Target      string
	Type        uint8
	X           int16
	Y           int16
	TargetMapId _map.Id
	ScriptName  string
}

// DefaultPortalFixture returns a basic portal fixture with default values
func DefaultPortalFixture() PortalFixture {
	return PortalFixture{
		Id:          "1",
		Name:        "portal_test",
		Target:      "",
		Type:        0,
		X:           100,
		Y:           200,
		TargetMapId: 999999999, // No target map
		ScriptName:  "",
	}
}

// PortalWithScript returns a portal fixture that has a script
func PortalWithScript(scriptName string) PortalFixture {
	f := DefaultPortalFixture()
	f.ScriptName = scriptName
	return f
}

// PortalWithTarget returns a portal fixture that has a target map
func PortalWithTarget(targetMapId _map.Id, targetName string) PortalFixture {
	f := DefaultPortalFixture()
	f.TargetMapId = targetMapId
	f.Target = targetName
	return f
}

// ToRestModel converts a fixture to a RestModel for testing Extract
func (f PortalFixture) ToRestModel() portal.RestModel {
	rm := portal.RestModel{
		Name:        f.Name,
		Target:      f.Target,
		Type:        f.Type,
		X:           f.X,
		Y:           f.Y,
		TargetMapId: f.TargetMapId,
		ScriptName:  f.ScriptName,
	}
	rm.SetID(f.Id)
	return rm
}
