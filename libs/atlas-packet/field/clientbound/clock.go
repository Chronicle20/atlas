package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ClockWriter = "Clock"

type ClockType byte

const (
	EventClock             ClockType = 0x00
	TownClock              ClockType = 0x01
	TimerClock             ClockType = 0x02
	EventTimerClock        ClockType = 0x03
	CakePieEventTimerClock ClockType = 0x64
)

type Clock struct {
	clockType ClockType
	seconds   uint32
	hour      byte
	minute    byte
	second    byte
	flag1     bool
	flag2     bool
}

func NewEventClock(seconds uint32) Clock {
	return Clock{clockType: EventClock, seconds: seconds}
}

func NewTownClock(hour byte, minute byte, second byte) Clock {
	return Clock{clockType: TownClock, hour: hour, minute: minute, second: second}
}

func NewTimerClock(seconds uint32) Clock {
	return Clock{clockType: TimerClock, seconds: seconds}
}

func NewEventTimerClock(seconds uint32) Clock {
	return Clock{clockType: EventTimerClock, seconds: seconds, flag1: true}
}

func NewCakePieEventTimerClock(seconds uint32) Clock {
	return Clock{clockType: CakePieEventTimerClock, seconds: seconds, flag1: true, flag2: true}
}

func (m Clock) Operation() string { return ClockWriter }
func (m Clock) String() string {
	return fmt.Sprintf("clockType [%d]", m.clockType)
}

func (m Clock) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(m.clockType))
		switch m.clockType {
		case EventClock, TimerClock:
			w.WriteInt(m.seconds)
		case TownClock:
			w.WriteByte(m.hour)
			w.WriteByte(m.minute)
			w.WriteByte(m.second)
		case EventTimerClock:
			w.WriteBool(m.flag1)
			w.WriteInt(m.seconds)
		case CakePieEventTimerClock:
			w.WriteBool(m.flag1)
			w.WriteBool(m.flag2)
			w.WriteInt(m.seconds)
		}
		return w.Bytes()
	}
}

func (m *Clock) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.clockType = ClockType(r.ReadByte())
		switch m.clockType {
		case EventClock, TimerClock:
			m.seconds = r.ReadUint32()
		case TownClock:
			m.hour = r.ReadByte()
			m.minute = r.ReadByte()
			m.second = r.ReadByte()
		case EventTimerClock:
			m.flag1 = r.ReadBool()
			m.seconds = r.ReadUint32()
		case CakePieEventTimerClock:
			m.flag1 = r.ReadBool()
			m.flag2 = r.ReadBool()
			m.seconds = r.ReadUint32()
		}
	}
}
