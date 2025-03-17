package reactor

type Model struct {
	Id             uint32 `json:"id"`
	Classification uint32 `json:"classification"`
	Name           string `json:"name"`
	X              int16  `json:"x"`
	Y              int16  `json:"y"`
	Delay          uint32 `json:"delay"`
	Direction      byte   `json:"direction"`
}
