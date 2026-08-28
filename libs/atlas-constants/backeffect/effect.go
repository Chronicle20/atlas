package backeffect

// Effect is the semantic back-effect state carried across the map/channel
// domain, Kafka bodies, saga payloads, and the REST surface (task-281 DOM-25
// fix). The client's raw wire byte -- 0=show, 1=hide, per
// libs/atlas-packet/field/clientbound/set_back_effect.go -- is resolved from
// this value only at the atlas-channel codec boundary; every other layer
// carries the semantic name.
type Effect string

const (
	EffectShow Effect = "SHOW"
	EffectHide Effect = "HIDE"
)
