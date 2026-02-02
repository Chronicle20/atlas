package worlds

type RestModel struct {
	Name              string  `json:"name"`
	Flag              string  `json:"flag"`
	ServerMessage     string  `json:"serverMessage"`
	EventMessage      string  `json:"eventMessage"`
	WhyAmIRecommended string  `json:"whyAmIRecommended"`
	ExpRate           float64 `json:"expRate"`
	MesoRate          float64 `json:"mesoRate"`
	ItemDropRate      float64 `json:"itemDropRate"`
	QuestExpRate      float64 `json:"questExpRate"`
}

// GetExpRate returns the configured exp rate, defaulting to 1.0 if not set
func (r RestModel) GetExpRate() float64 {
	if r.ExpRate == 0 {
		return 1.0
	}
	return r.ExpRate
}

// GetMesoRate returns the configured meso rate, defaulting to 1.0 if not set
func (r RestModel) GetMesoRate() float64 {
	if r.MesoRate == 0 {
		return 1.0
	}
	return r.MesoRate
}

// GetItemDropRate returns the configured item drop rate, defaulting to 1.0 if not set
func (r RestModel) GetItemDropRate() float64 {
	if r.ItemDropRate == 0 {
		return 1.0
	}
	return r.ItemDropRate
}

// GetQuestExpRate returns the configured quest exp rate, defaulting to 1.0 if not set
func (r RestModel) GetQuestExpRate() float64 {
	if r.QuestExpRate == 0 {
		return 1.0
	}
	return r.QuestExpRate
}
