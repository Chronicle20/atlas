package writer

import (
	"time"

	fieldpkt "github.com/Chronicle20/atlas-packet/field"
	"github.com/Chronicle20/atlas-socket/packet"
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
	return fieldpkt.NewEventClock(DurationToUint32Seconds(duration)).Encode
}

func TownClockBody(t time.Time) packet.Encode {
	return fieldpkt.NewTownClock(byte(t.Hour()), byte(t.Minute()), byte(t.Second())).Encode
}

func TimerClockBody(duration time.Duration) packet.Encode {
	return fieldpkt.NewTimerClock(DurationToUint32Seconds(duration)).Encode
}

func EventTimerClockBody(duration time.Duration) packet.Encode {
	return fieldpkt.NewEventTimerClock(DurationToUint32Seconds(duration)).Encode
}

func CakePieEventTimerClockBody(duration time.Duration) packet.Encode {
	return fieldpkt.NewCakePieEventTimerClock(DurationToUint32Seconds(duration)).Encode
}
