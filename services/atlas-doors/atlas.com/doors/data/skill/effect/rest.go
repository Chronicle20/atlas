package effect

// RestModel is the JSON wire shape for one skill-effect level as returned by
// atlas-data's GET /data/skills/{id} endpoint. Only the fields used by
// atlas-doors are decoded; the JSON decoder ignores the rest.
type RestModel struct {
	MPConsume   uint16 `json:"MPConsume"`
	Duration    int32  `json:"duration"`
	ItemConsume uint32 `json:"itemConsume"`
}

// Extract converts a RestModel into an immutable Model.
// Exported so tests can call it directly without network I/O.
func Extract(rm RestModel) (Model, error) {
	return Model{
		duration:    rm.Duration,
		mpConsume:   rm.MPConsume,
		itemConsume: rm.ItemConsume,
	}, nil
}
