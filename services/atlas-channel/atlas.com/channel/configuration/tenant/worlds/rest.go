package worlds

type RestModel struct {
	Name              string `json:"name"`
	Flag              string `json:"flag"`
	ServerMessage     string `json:"serverMessage"`
	EventMessage      string `json:"eventMessage"`
	WhyAmIRecommended string `json:"whyAmIRecommended"`
}
