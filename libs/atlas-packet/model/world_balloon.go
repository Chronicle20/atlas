package model

import (
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// WorldBalloon is a floating announcement bubble shown on the world-select
// screen. Each balloon has an absolute (x, y) screen position and a message
// payload. Sent inside the `OnWorldInformation` packet after the per-world
// channel list, prefixed by a uint16 count.
type WorldBalloon struct {
	x       int16
	y       int16
	message string
}

func NewWorldBalloon(x int16, y int16, message string) WorldBalloon {
	return WorldBalloon{x: x, y: y, message: message}
}

func (m WorldBalloon) X() int16        { return m.x }
func (m WorldBalloon) Y() int16        { return m.y }
func (m WorldBalloon) Message() string { return m.message }

func (m WorldBalloon) Write(w *response.Writer) {
	w.WriteShort(uint16(m.x))
	w.WriteShort(uint16(m.y))
	w.WriteAsciiString(m.message)
}

func (m *WorldBalloon) Read(r *request.Reader) {
	m.x = int16(r.ReadUint16())
	m.y = int16(r.ReadUint16())
	m.message = r.ReadAsciiString()
}
