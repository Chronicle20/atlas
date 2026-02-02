package worlds

type RestModel struct {
	Name              string  `json:"name"`
	Flag              string  `json:"flag"`
	ServerMessage     string  `json:"serverMessage"`
	EventMessage      string  `json:"eventMessage"`
	WhyAmIRecommended string  `json:"whyAmIRecommended"`
	ExpRate           float64 `json:"expRate,omitempty"`
	MesoRate          float64 `json:"mesoRate,omitempty"`
	ItemDropRate      float64 `json:"itemDropRate,omitempty"`
	QuestExpRate      float64 `json:"questExpRate,omitempty"`
}
