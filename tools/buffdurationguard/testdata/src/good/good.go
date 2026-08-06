package good

import "time"

type statChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type applyDiseaseBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Level    byte         `json:"level"`
	Duration int32        `json:"duration"`
	Changes  []statChange `json:"changes"`
}

type createCommandBody struct {
	DiseaseDuration int64 `json:"diseaseDuration"`
	Duration        int64 `json:"duration"`
	TickIntervalMs  int64 `json:"tickIntervalMs"`
}

const mistDurationCapMs int64 = 60_000

// fixedMistTick is atlas-maps after task-190.
func fixedMistTick(d time.Duration) applyDiseaseBody {
	return applyDiseaseBody{
		SourceId: 1,
		Duration: int32(d.Milliseconds()),
		Changes:  []statChange{{Type: "POISON", Amount: 80}},
	}
}

// fixedMist is atlas-monsters after task-190. The 60_000 cap is a named const,
// not a scaling factor, and TickIntervalMs is not a guarded field.
func fixedMist(dur int64) createCommandBody {
	durMs := dur
	if durMs > mistDurationCapMs {
		durMs = mistDurationCapMs
	}
	return createCommandBody{
		DiseaseDuration: durMs,
		Duration:        durMs,
		TickIntervalMs:  1000,
	}
}

// justified shows the escape hatch: an annotated site stays legal and visible.
func justified(sec int32) applyDiseaseBody {
	return applyDiseaseBody{
		SourceId: 2,
		//buffdurationguard:allow upstream field is authored in seconds by design
		Duration: sec * 1000,
		Changes:  nil,
	}
}
