package channel

import (
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func GetAll(l logrus.FieldLogger, span opentracing.Span) ([]Model, error) {
	//return requests.SliceProvider[attributes, Model](l, span)(requestChannels(), makeChannel)()
	var channels = []Model{
		Model{
			worldId:   0,
			channelId: 0,
			capacity:  0,
			ipAddress: "127.0.0.1",
			port:      7575,
		},
		Model{
			worldId:   0,
			channelId: 1,
			capacity:  0,
			ipAddress: "127.0.0.1",
			port:      7576,
		},
		Model{
			worldId:   0,
			channelId: 2,
			capacity:  0,
			ipAddress: "127.0.0.1",
			port:      7577,
		},
	}
	return channels, nil
}

func GetChannelLoadByWorld(l logrus.FieldLogger, span opentracing.Span) (map[int][]Load, error) {
	cs, err := GetAll(l, span)
	if err != nil {
		return nil, err
	}

	var cls = make(map[int][]Load)
	for _, x := range cs {
		cl := NewChannelLoad(x.ChannelId(), x.Capacity())
		if _, ok := cls[int(x.WorldId())]; ok {
			cls[int(x.WorldId())] = append(cls[int(x.WorldId())], cl)
		} else {
			cls[int(x.WorldId())] = append([]Load{}, cl)
		}
	}
	return cls, nil
}
