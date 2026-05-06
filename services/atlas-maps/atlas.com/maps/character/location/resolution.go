package location

type ResolutionReason string

const (
	ReasonForcedReturn ResolutionReason = "forced_return"
	ReasonStayPut      ResolutionReason = "stay_put"
)
