package bad

import "time"

type statChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

// BD-1 fingerprint: sourceId + duration + changes.
type applyDiseaseBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Level    byte         `json:"level"`
	Duration int32        `json:"duration"`
	Changes  []statChange `json:"changes"`
}

// BD-2 fingerprint: diseaseDuration + tickIntervalMs.
type createCommandBody struct {
	DiseaseDuration int64 `json:"diseaseDuration"`
	Duration        int64 `json:"duration"`
	TickIntervalMs  int64 `json:"tickIntervalMs"`
}

// historicalMistTick is atlas-maps tasks/mist_tick.go:86 as it stood before
// task-190: it divided milliseconds back down to seconds.
func historicalMistTick(d time.Duration) applyDiseaseBody {
	return applyDiseaseBody{
		SourceId: 1,
		Duration: int32(d / time.Second), // want "duration fields .* MILLISECONDS"
		Changes:  []statChange{{Type: "POISON", Amount: 80}},
	}
}

// historicalMist is atlas-monsters monster/processor.go:1068 as it stood before
// task-190: it multiplied an already-ms value by 1000. The scaling lives in a
// local, one level away from the composite literal.
func historicalMist(dur int64) createCommandBody {
	durMs := dur * int64(time.Second/time.Millisecond) // want "duration fields .* MILLISECONDS"
	return createCommandBody{
		DiseaseDuration: durMs,
		Duration:        durMs,
		TickIntervalMs:  1000,
	}
}

// inlineThousand is the other shape of the same defect.
func inlineThousand(sec int32) applyDiseaseBody {
	return applyDiseaseBody{
		SourceId: 3,
		Duration: sec * 1000, // want "duration fields .* MILLISECONDS"
		Changes:  nil,
	}
}
