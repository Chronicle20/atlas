package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"time"
)

type ClockType byte

const (
	Clock                  = "Clock"
	EventClock             = ClockType(0x00)
	TownClock              = ClockType(0x01)
	TimerClock             = ClockType(0x02)
	EventTimerClock        = ClockType(0x03)
	CakePieEventTimerClock = ClockType(0x64)
)

func DurationToUint32Seconds(d time.Duration) uint32 {
	seconds := int64(d.Seconds())

	// Clamp the value to uint32 bounds to avoid overflow
	if seconds < 0 {
		return 0
	}
	if seconds > int64(^uint32(0)) {
		return ^uint32(0) // max value of uint32
	}
	return uint32(seconds)
}

// EventClockBody writes an event clock payload with a given duration in seconds.
func EventClockBody(l logrus.FieldLogger, t tenant.Model) func(duration time.Duration) BodyProducer {
	return func(duration time.Duration) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(byte(EventClock))
			w.WriteInt(DurationToUint32Seconds(duration))
			return w.Bytes()
		}
	}
}

func TownClockBody(l logrus.FieldLogger, t tenant.Model) func(time time.Time) BodyProducer {
	return func(time time.Time) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(byte(TownClock))
			w.WriteByte(byte(time.Hour()))
			w.WriteByte(byte(time.Minute()))
			w.WriteByte(byte(time.Second()))
			return w.Bytes()
		}
	}
}

func TimerClockBody(l logrus.FieldLogger, t tenant.Model) func(duration time.Duration) BodyProducer {
	return func(duration time.Duration) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(byte(TimerClock))
			w.WriteInt(DurationToUint32Seconds(duration))
			return w.Bytes()
		}
	}
}

func EventTimerClockBody(l logrus.FieldLogger, t tenant.Model) func(duration time.Duration) BodyProducer {
	return func(duration time.Duration) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(byte(EventTimerClock))
			w.WriteBool(true) // not sure what this is used for. will skip set/start if false
			w.WriteInt(DurationToUint32Seconds(duration))
			return w.Bytes()
		}
	}
}

func CakePieEventTimerClockBody(l logrus.FieldLogger, t tenant.Model) func(duration time.Duration) BodyProducer {
	return func(duration time.Duration) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(byte(CakePieEventTimerClock))
			w.WriteBool(true) // not sure what this is used for. will skip set/start if false
			w.WriteBool(true) // adjusts height/width of timer window?
			w.WriteInt(DurationToUint32Seconds(duration))
			return w.Bytes()
		}
	}
}
