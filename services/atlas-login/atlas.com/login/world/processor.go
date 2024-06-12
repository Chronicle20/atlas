package world

import (
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func GetAll(l logrus.FieldLogger, span opentracing.Span) ([]Model, error) {
	//return requests.SliceProvider[attributes, Model](l, span)(requestWorlds(), makeWorld)()
	var worlds = []Model{{
		id:                 0,
		name:               "Scania",
		state:              StateNew,
		message:            "Scania Message",
		eventMessage:       "Server opening!",
		recommended:        true,
		recommendedMessage: "Its brand new!",
		capacityStatus:     0,
		channelLoad:        nil,
	}}
	return worlds, nil
}

func GetById(l logrus.FieldLogger, span opentracing.Span) func(worldId byte) (Model, error) {
	return func(worldId byte) (Model, error) {
		//return requests.Provider[attributes, Model](l, span)(requestWorld(worldId), makeWorld)()
		var world = Model{
			id:                 0,
			name:               "",
			state:              0,
			message:            "",
			eventMessage:       "",
			recommended:        false,
			recommendedMessage: "",
			capacityStatus:     0,
			channelLoad:        nil,
		}
		return world, nil
	}
}

func GetCapacityStatus(l logrus.FieldLogger, span opentracing.Span) func(worldId byte) Status {
	return func(worldId byte) Status {
		w, err := GetById(l, span)(worldId)
		if err != nil {
			return StatusFull
		}
		return w.CapacityStatus()
	}
}
