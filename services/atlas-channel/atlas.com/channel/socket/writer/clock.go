package writer

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
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
func EventClockBody(duration time.Duration) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(EventClock))
			w.WriteInt(DurationToUint32Seconds(duration))
			return w.Bytes()
		}
	}
}

func TownClockBody(time time.Time) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(TownClock))
			w.WriteByte(byte(time.Hour()))
			w.WriteByte(byte(time.Minute()))
			w.WriteByte(byte(time.Second()))
			return w.Bytes()
		}
	}
}

func TimerClockBody(duration time.Duration) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(TimerClock))
			w.WriteInt(DurationToUint32Seconds(duration))
			return w.Bytes()
		}
	}
}

func EventTimerClockBody(duration time.Duration) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(EventTimerClock))
			w.WriteBool(true) // not sure what this is used for. will skip set/start if false
			w.WriteInt(DurationToUint32Seconds(duration))
			return w.Bytes()
		}
	}
}

func CakePieEventTimerClockBody(duration time.Duration) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(CakePieEventTimerClock))
			w.WriteBool(true) // not sure what this is used for. will skip set/start if false
			w.WriteBool(true) // adjusts height/width of timer window?
			w.WriteInt(DurationToUint32Seconds(duration))
			return w.Bytes()
		}
	}
}
