package session

type Model struct {
	Code   string `json:"code"`
	Reason byte   `json:"reason"`
	Until  uint64 `json:"until"`
}

func ErrorModel(code string) Model {
	return Model{Code: code}
}

func OkModel() Model {
	return Model{Code: "OK"}
}
