package surprise

// RestModel carries the Cash Shop Surprise box configuration. BoxTemplateIds
// is the set of cash item template ids that open as a Surprise box; the
// pool a box draws from is the atlas-reward-pools pool whose id equals the
// box's template id. A list rather than a scalar so a tenant can designate
// additional boxes beyond the stock 5222000 (task-207 FR-2.2 / DOM-25).
type RestModel struct {
	BoxTemplateIds []uint32 `json:"boxTemplateIds,omitempty"`
}
